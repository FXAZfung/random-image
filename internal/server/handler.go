package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/limiter"
	"github.com/FXAZfung/random-image/internal/picker"
)

type Handler struct {
	picker             *picker.Picker
	cache              *cache.Cache
	limiter            *limiter.Limiter
	relayMode          string
	version            string
	cacheControlMaxAge time.Duration
}

func NewHandler(p *picker.Picker, c *cache.Cache, l *limiter.Limiter, relayCfg config.RelayConfig, version string) *Handler {
	return &Handler{
		picker:             p,
		cache:              c,
		limiter:            l,
		relayMode:          relayCfg.Mode,
		version:            version,
		cacheControlMaxAge: relayCfg.CacheControlMaxAge,
	}
}

func (h *Handler) HandleRandom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	category := extractCategory(r.URL.Path)
	if category == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "category is required, use /api/{category}",
		})
		return
	}

	if !h.picker.HasCategory(category) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("category %q not found", category),
		})
		return
	}

	responseType := r.URL.Query().Get("type")
	if responseType == "" {
		responseType = h.relayMode
	}

	switch responseType {
	case "redirect":
		h.handleRedirect(w, r, category)
	case "json":
		h.handleJSON(w, r, category)
	default:
		h.handleProxy(w, r, category)
	}
}

func (h *Handler) handleProxy(w http.ResponseWriter, r *http.Request, category string) {
	result, err := h.picker.Pick(r.Context(), category)
	if err != nil {
		slog.Error("pick image failed", "category", category, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if h.writeConditionalNotModified(w, r, result) {
		return
	}

	h.applyImageHeaders(w, result)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(result.Data)))
	w.Header().Set("X-Image-Path", result.Path)
	w.Header().Set("X-Relay-Mode", "proxy")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Data)
}

func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request, category string) {
	store, ok := h.picker.GetStorage(category)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	if !store.SupportsRedirect() {
		slog.Debug("storage does not support redirect, falling back to proxy",
			"category", category,
			"storage", store.Name(),
		)
		h.handleProxy(w, r, category)
		return
	}

	imgPath, err := h.picker.PickPath(r.Context(), category)
	if err != nil {
		slog.Error("pick path failed", "category", category, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rawURL, err := store.GetImageURL(r.Context(), imgPath)
	if err != nil {
		slog.Error("get image url failed", "path", imgPath, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get image url"})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Image-Path", imgPath)
	w.Header().Set("X-Relay-Mode", "redirect")
	http.Redirect(w, r, rawURL, http.StatusFound)
}

func (h *Handler) handleJSON(w http.ResponseWriter, r *http.Request, category string) {
	store, ok := h.picker.GetStorage(category)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	imgPath, err := h.picker.PickPath(r.Context(), category)
	if err != nil {
		slog.Error("pick path failed", "category", category, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]string{
		"path":     imgPath,
		"category": category,
		"storage":  store.Name(),
	}

	if store.SupportsRedirect() {
		rawURL, err := store.GetImageURL(r.Context(), imgPath)
		if err == nil {
			resp["url"] = rawURL
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"categories": h.picker.Categories(),
	})
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	cacheItems, cacheBytes := h.cache.Stats()
	visitorTotal, visitorBanned := h.limiter.Stats()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"cache": map[string]interface{}{
			"items":     cacheItems,
			"memory_mb": float64(cacheBytes) / 1024 / 1024,
		},
		"limiter": map[string]interface{}{
			"visitors": visitorTotal,
			"banned":   visitorBanned,
		},
		"relay_mode": h.relayMode,
	})
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":    "Random Image API",
		"version": h.version,
		"endpoints": map[string]string{
			"random_image": "/api/{category}",
			"categories":   "/api/categories",
			"health":       "/health",
		},
		"relay_mode": h.relayMode,
		"categories": h.picker.Categories(),
	})
}

func (h *Handler) writeConditionalNotModified(w http.ResponseWriter, r *http.Request, result *picker.ImageResult) bool {
	if result.ETag != "" && r.Header.Get("If-None-Match") == result.ETag {
		h.applyImageHeaders(w, result)
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	if !result.LastModified.IsZero() {
		ifModifiedSince := r.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			if t, err := http.ParseTime(ifModifiedSince); err == nil && !result.LastModified.After(t) {
				h.applyImageHeaders(w, result)
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

func (h *Handler) applyImageHeaders(w http.ResponseWriter, result *picker.ImageResult) {
	w.Header().Set("Content-Type", result.ContentType)
	if result.ETag != "" {
		w.Header().Set("ETag", result.ETag)
	}
	if !result.LastModified.IsZero() {
		w.Header().Set("Last-Modified", result.LastModified.UTC().Format(http.TimeFormat))
	}
	if h.cacheControlMaxAge > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d, must-revalidate", int(h.cacheControlMaxAge.Seconds())))
	} else {
		w.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
	}
}

func extractCategory(urlPath string) string {
	p := strings.TrimPrefix(urlPath, "/api/")
	p = strings.Trim(p, "/")
	if strings.Contains(p, "/") {
		return ""
	}
	return p
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
