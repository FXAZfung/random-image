package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server        ServerConfig     `yaml:"server"`
	Alist         AlistConfig      `yaml:"alist"`
	Local         LocalConfig      `yaml:"local"`
	OutboundProxy ProxyConfig      `yaml:"outbound_proxy"`
	Relay         RelayConfig      `yaml:"relay"`
	Cache         CacheConfig      `yaml:"cache"`
	Limiter       LimiterConfig    `yaml:"limiter"`
	Startup       StartupConfig    `yaml:"startup"`
	Selection     SelectionConfig  `yaml:"selection"`
	Categories    []CategoryConfig `yaml:"categories"`
}

type ServerConfig struct {
	Address      string        `yaml:"address"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type AlistConfig struct {
	Enabled  bool          `yaml:"enabled"`
	URL      string        `yaml:"url"`
	Token    string        `yaml:"token"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
}

type LocalConfig struct {
	Enabled  bool   `yaml:"enabled"`
	BasePath string `yaml:"base_path"`
}

type ProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

type RelayConfig struct {
	Mode               string        `yaml:"mode"`
	MaxBodySizeMB      int           `yaml:"max_body_size_mb"`
	UserAgent          string        `yaml:"user_agent"`
	CacheControlMaxAge time.Duration `yaml:"cache_control_max_age"`
}

type CacheConfig struct {
	MaxSize          int           `yaml:"max_size"`
	MaxMemoryMB      int           `yaml:"max_memory_mb"`
	PrefetchCount    int           `yaml:"prefetch_count"`
	PrefetchInterval time.Duration `yaml:"prefetch_interval"`
	TTL              time.Duration `yaml:"ttl"`
}

type LimiterConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Rate            int           `yaml:"rate"`
	Burst           int           `yaml:"burst"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
	BanThreshold    int           `yaml:"ban_threshold"`
	BanDuration     time.Duration `yaml:"ban_duration"`
}

type StartupConfig struct {
	RequireReadyCategory bool `yaml:"require_ready_category"`
}

type SelectionConfig struct {
	AvoidRepeats int `yaml:"avoid_repeats"`
}

type CategoryConfig struct {
	Name        string `yaml:"name"`
	Storage     string `yaml:"storage"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{
		Startup: StartupConfig{
			RequireReadyCategory: true,
		},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()
	if err := cfg.applyEnvOverrides(); err != nil {
		return nil, fmt.Errorf("apply env overrides: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.Server.Address == "" {
		c.Server.Address = ":8080"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 10 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Alist.Timeout == 0 {
		c.Alist.Timeout = 15 * time.Second
	}
	if c.Relay.Mode == "" {
		c.Relay.Mode = "proxy"
	}
	if c.Relay.MaxBodySizeMB == 0 {
		c.Relay.MaxBodySizeMB = 20
	}
	if c.Relay.UserAgent == "" {
		c.Relay.UserAgent = "RandomImage/dev"
	}
	if c.Relay.CacheControlMaxAge == 0 {
		c.Relay.CacheControlMaxAge = 0
	}
	if c.Cache.MaxSize == 0 {
		c.Cache.MaxSize = 256
	}
	if c.Cache.MaxMemoryMB == 0 {
		c.Cache.MaxMemoryMB = 512
	}
	if c.Cache.PrefetchCount == 0 {
		c.Cache.PrefetchCount = 5
	}
	if c.Cache.PrefetchInterval == 0 {
		c.Cache.PrefetchInterval = 60 * time.Second
	}
	if c.Cache.TTL == 0 {
		c.Cache.TTL = 30 * time.Minute
	}
	if c.Limiter.Rate == 0 {
		c.Limiter.Rate = 30
	}
	if c.Limiter.Burst == 0 {
		c.Limiter.Burst = 10
	}
	if c.Limiter.CleanupInterval == 0 {
		c.Limiter.CleanupInterval = 5 * time.Minute
	}
	if c.Limiter.BanThreshold == 0 {
		c.Limiter.BanThreshold = 100
	}
	if c.Limiter.BanDuration == 0 {
		c.Limiter.BanDuration = 30 * time.Minute
	}
	c.inferCategoryStorage()
}

func (c *Config) inferCategoryStorage() {
	for i := range c.Categories {
		if c.Categories[i].Storage == "" {
			if c.Local.Enabled {
				c.Categories[i].Storage = "local"
			} else if c.Alist.Enabled {
				c.Categories[i].Storage = "alist"
			}
		}
	}
}

func (c *Config) applyEnvOverrides() error {
	overrideString("RI_SERVER_ADDRESS", &c.Server.Address)
	overrideBool("RI_ALIST_ENABLED", &c.Alist.Enabled)
	overrideString("RI_ALIST_URL", &c.Alist.URL)
	overrideString("RI_ALIST_TOKEN", &c.Alist.Token)
	overrideString("RI_ALIST_USERNAME", &c.Alist.Username)
	overrideString("RI_ALIST_PASSWORD", &c.Alist.Password)
	if err := overrideDuration("RI_ALIST_TIMEOUT", &c.Alist.Timeout); err != nil {
		return err
	}
	overrideBool("RI_LOCAL_ENABLED", &c.Local.Enabled)
	overrideString("RI_LOCAL_BASE_PATH", &c.Local.BasePath)
	overrideBool("RI_OUTBOUND_PROXY_ENABLED", &c.OutboundProxy.Enabled)
	overrideString("RI_OUTBOUND_PROXY_URL", &c.OutboundProxy.URL)
	overrideString("RI_RELAY_MODE", &c.Relay.Mode)
	overrideString("RI_RELAY_USER_AGENT", &c.Relay.UserAgent)
	if err := overrideInt("RI_RELAY_MAX_BODY_SIZE_MB", &c.Relay.MaxBodySizeMB); err != nil {
		return err
	}
	if err := overrideDuration("RI_RELAY_CACHE_CONTROL_MAX_AGE", &c.Relay.CacheControlMaxAge); err != nil {
		return err
	}
	if err := overrideInt("RI_CACHE_MAX_SIZE", &c.Cache.MaxSize); err != nil {
		return err
	}
	if err := overrideInt("RI_CACHE_MAX_MEMORY_MB", &c.Cache.MaxMemoryMB); err != nil {
		return err
	}
	if err := overrideInt("RI_CACHE_PREFETCH_COUNT", &c.Cache.PrefetchCount); err != nil {
		return err
	}
	if err := overrideDuration("RI_CACHE_PREFETCH_INTERVAL", &c.Cache.PrefetchInterval); err != nil {
		return err
	}
	if err := overrideDuration("RI_CACHE_TTL", &c.Cache.TTL); err != nil {
		return err
	}
	overrideBool("RI_LIMITER_ENABLED", &c.Limiter.Enabled)
	if err := overrideInt("RI_LIMITER_RATE", &c.Limiter.Rate); err != nil {
		return err
	}
	if err := overrideInt("RI_LIMITER_BURST", &c.Limiter.Burst); err != nil {
		return err
	}
	if err := overrideDuration("RI_LIMITER_CLEANUP_INTERVAL", &c.Limiter.CleanupInterval); err != nil {
		return err
	}
	if err := overrideInt("RI_LIMITER_BAN_THRESHOLD", &c.Limiter.BanThreshold); err != nil {
		return err
	}
	if err := overrideDuration("RI_LIMITER_BAN_DURATION", &c.Limiter.BanDuration); err != nil {
		return err
	}
	overrideBool("RI_STARTUP_REQUIRE_READY_CATEGORY", &c.Startup.RequireReadyCategory)
	if err := overrideInt("RI_SELECTION_AVOID_REPEATS", &c.Selection.AvoidRepeats); err != nil {
		return err
	}

	c.inferCategoryStorage()
	return nil
}

func (c *Config) validate() error {
	if !c.Alist.Enabled && !c.Local.Enabled {
		return fmt.Errorf("at least one storage backend (alist or local) must be enabled")
	}

	if len(c.Categories) == 0 {
		return fmt.Errorf("at least one category must be configured")
	}

	for _, cat := range c.Categories {
		if cat.Name == "" {
			return fmt.Errorf("category name cannot be empty")
		}
		if cat.Path == "" {
			return fmt.Errorf("category %q: path cannot be empty", cat.Name)
		}
		switch cat.Storage {
		case "alist":
			if !c.Alist.Enabled {
				return fmt.Errorf("category %q uses alist storage but alist is not enabled", cat.Name)
			}
		case "local":
			if !c.Local.Enabled {
				return fmt.Errorf("category %q uses local storage but local is not enabled", cat.Name)
			}
		default:
			return fmt.Errorf("category %q: unknown storage type %q (must be 'alist' or 'local')", cat.Name, cat.Storage)
		}
	}

	if c.Local.Enabled && c.Local.BasePath == "" {
		return fmt.Errorf("local storage enabled but base_path is empty")
	}
	if c.Alist.Enabled && c.Alist.URL == "" {
		return fmt.Errorf("alist storage enabled but url is empty")
	}
	if c.Selection.AvoidRepeats < 0 {
		return fmt.Errorf("selection.avoid_repeats cannot be negative")
	}

	return nil
}

func overrideString(key string, target *string) {
	if value, ok := os.LookupEnv(key); ok {
		*target = value
	}
}

func overrideBool(key string, target *bool) {
	if value, ok := os.LookupEnv(key); ok {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			*target = parsed
		}
	}
}

func overrideInt(key string, target *int) error {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	*target = parsed
	return nil
}

func overrideDuration(key string, target *time.Duration) error {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	*target = parsed
	return nil
}
