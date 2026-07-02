package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error loading non-existent config, got %v", err)
	}
	if cfg.LastTab != 0 {
		t.Fatalf("expected LastTab to be 0, got %d", cfg.LastTab)
	}

	cfg.LastTab = 2
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("expected no error saving config, got %v", err)
	}

	path := filepath.Join(tmpDir, ".config", "nmtui", "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected config file to be created at %s", path)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}
	if loadedCfg.LastTab != 2 {
		t.Fatalf("expected LastTab to be 2, got %d", loadedCfg.LastTab)
	}
}
