package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvOverrides(t *testing.T) {
	t.Setenv("RI_ALIST_TOKEN", "env-token")
	t.Setenv("RI_LOCAL_BASE_PATH", "./env-images")
	t.Setenv("RI_SELECTION_AVOID_REPEATS", "3")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
local:
  enabled: true
  base_path: "./images"
alist:
  enabled: false
categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Alist.Token != "env-token" {
		t.Fatalf("expected env token override, got %q", cfg.Alist.Token)
	}
	if cfg.Local.BasePath != "./env-images" {
		t.Fatalf("expected env base path override, got %q", cfg.Local.BasePath)
	}
	if cfg.Selection.AvoidRepeats != 3 {
		t.Fatalf("expected avoid repeats to be 3, got %d", cfg.Selection.AvoidRepeats)
	}
	if !cfg.Startup.RequireReadyCategory {
		t.Fatal("expected require_ready_category default to true")
	}
}

func TestLoadRejectsNegativeAvoidRepeats(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
local:
  enabled: true
  base_path: "./images"
selection:
  avoid_repeats: -1
categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("expected config validation error")
	}
}
