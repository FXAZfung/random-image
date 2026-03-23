package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FXAZfung/random-image/internal/alist"
	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/limiter"
	"github.com/FXAZfung/random-image/internal/picker"
	"github.com/FXAZfung/random-image/internal/proxy"
	"github.com/FXAZfung/random-image/internal/server"
	"github.com/FXAZfung/random-image/internal/storage"
)

var (
	configPath string
	debug      bool
	version    = "dev"
)

func main() {
	flag.StringVar(&configPath, "config", "config.yaml", "config file path")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.Parse()

	if debug {
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				slog.Debug("pprof server stopped", "error", err)
			}
		}()
	}

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("random-image starting", "version", version)

	slog.Info("loading config", "path", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"address", cfg.Server.Address,
		"alist_enabled", cfg.Alist.Enabled,
		"local_enabled", cfg.Local.Enabled,
		"relay_mode", cfg.Relay.Mode,
		"categories", len(cfg.Categories),
	)

	storageMap := make(map[string]storage.Storage)

	if cfg.Local.Enabled {
		localStorage, err := storage.NewLocalStorage(cfg.Local.BasePath)
		if err != nil {
			slog.Error("failed to init local storage", "error", err)
			os.Exit(1)
		}
		storageMap["local"] = localStorage
		slog.Info("local storage initialized", "base_path", cfg.Local.BasePath)
	}

	if cfg.Alist.Enabled {
		apiClient, err := proxy.NewHTTPClient(
			cfg.OutboundProxy.Enabled,
			cfg.OutboundProxy.URL,
			cfg.Alist.Timeout,
		)
		if err != nil {
			slog.Error("failed to create api http client", "error", err)
			os.Exit(1)
		}

		downloadClient, err := proxy.NewDownloadClient(
			cfg.OutboundProxy.Enabled,
			cfg.OutboundProxy.URL,
			60*time.Second,
		)
		if err != nil {
			slog.Error("failed to create download http client", "error", err)
			os.Exit(1)
		}

		alistClient := alist.NewClient(
			cfg.Alist.URL,
			cfg.Alist.Token,
			cfg.Alist.Username,
			cfg.Alist.Password,
			apiClient,
			downloadClient,
			cfg.Relay.UserAgent,
			cfg.Relay.MaxBodySizeMB,
		)

		alistStorage := storage.NewAlistStorage(alistClient)
		storageMap["alist"] = alistStorage
		slog.Info("alist storage initialized", "url", cfg.Alist.URL)
	}

	for _, cat := range cfg.Categories {
		if _, ok := storageMap[cat.Storage]; !ok {
			slog.Error("category references unavailable storage",
				"category", cat.Name,
				"storage", cat.Storage,
			)
			os.Exit(1)
		}
	}

	imgCache := cache.New(
		cfg.Cache.MaxSize,
		cfg.Cache.MaxMemoryMB,
		cfg.Cache.TTL,
	)

	lim := limiter.New(
		cfg.Limiter.Rate,
		cfg.Limiter.Burst,
		cfg.Limiter.BanThreshold,
		cfg.Limiter.BanDuration,
		cfg.Limiter.CleanupInterval,
	)

	imgPicker := picker.New(
		imgCache,
		cfg.Categories,
		storageMap,
		cfg.Cache.PrefetchCount,
		cfg.Cache.PrefetchInterval,
		cfg.Selection.AvoidRepeats,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	if err := imgPicker.Start(startCtx); err != nil {
		startCancel()
		slog.Error("failed to start picker", "error", err)
		os.Exit(1)
	}
	startCancel()

	if cfg.Startup.RequireReadyCategory && imgPicker.ReadyCategoryCount() == 0 {
		slog.Error("no ready categories available after startup scan")
		os.Exit(1)
	}

	for _, cat := range imgPicker.Categories() {
		slog.Info("category ready",
			"name", cat.Name,
			"storage", cat.Storage,
			"images", cat.Count,
			"description", cat.Description,
		)
	}

	srv := server.New(cfg, imgPicker, imgCache, lim, version)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received signal", "signal", sig)
	case err := <-errCh:
		if err != nil {
			slog.Error("server error", "error", err)
		}
	}

	slog.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	imgPicker.Stop()

	slog.Info("server stopped, bye!")
}
