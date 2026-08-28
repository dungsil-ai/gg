package main

import (
	"errors"
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

func TestSaveProviderReplacesExistingConfig(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	if err := SaveProvider("github.com", GH); err != nil {
		t.Fatal(err)
	}
	if err := SaveProvider("git.example.com", GLab); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Hosts) != 2 || cfg.Hosts["github.com"] != "gh" || cfg.Hosts["git.example.com"] != "glab" {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
}

func TestSaveProviderRenameFailurePreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GG_HOME", dir)
	if err := SaveProvider("github.com", GH); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}

	originalRename := renameConfigFile
	t.Cleanup(func() { renameConfigFile = originalRename })
	wantErr := errors.New("rename failed")
	renameCalls := 0
	renameConfigFile = func(tmpPath, dstPath string) error {
		renameCalls++
		if filepath.Dir(tmpPath) != dir {
			t.Errorf("temp dir = %q, want %q", filepath.Dir(tmpPath), dir)
		}
		if dstPath != ConfigPath() {
			t.Errorf("destination = %q, want %q", dstPath, ConfigPath())
		}
		return wantErr
	}

	if err := SaveProvider("git.example.com", GLab); !errors.Is(err, wantErr) {
		t.Fatalf("SaveProvider error = %v, want %v", err, wantErr)
	}
	if renameCalls != 1 {
		t.Errorf("rename calls = %d, want 1", renameCalls)
	}
	after, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("config changed after rename failure:\n%s", after)
	}
	tempFiles, err := filepath.Glob(filepath.Join(dir, "config-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tempFiles) != 0 {
		t.Errorf("temporary files were not removed: %v", tempFiles)
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
