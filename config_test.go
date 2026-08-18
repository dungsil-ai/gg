package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDirFallback(t *testing.T) {
	t.Setenv("GG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	t.Setenv("GG_HOME", `C:\gghome`)
	if got := ConfigDir(); got != `C:\gghome` {
		t.Errorf("GG_HOME 우선이어야 함, got %q", got)
	}

	t.Setenv("GG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("x", "cfg"))
	if got := ConfigDir(); got != filepath.Join("x", "cfg", "gg") {
		t.Errorf("XDG fallback = %q", got)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	if got := ConfigDir(); got != filepath.Join(home, ".gg") {
		t.Errorf("home fallback = %q", got)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("빈 설정이어야 함: %v", err)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
}

func TestSaveAndReload(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	if err := SaveProvider("git.example.com", GLab); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts["git.example.com"] != "glab" {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
	data, _ := os.ReadFile(ConfigPath())
	if !strings.Contains(string(data), `"hosts"`) {
		t.Errorf("json 형식이 아님: %s", data)
	}
}

func TestBrokenConfigIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GG_HOME", dir)
	bad := []byte("{broken json")
	if err := os.WriteFile(filepath.Join(dir, "config.json"), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("손상 파일은 error여야 함")
	}
	if err := SaveProvider("h", GH); err == nil {
		t.Fatal("손상 파일 위 저장은 실패해야 함")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(after) != string(bad) {
		t.Error("손상 파일이 덮어써짐")
	}
}
