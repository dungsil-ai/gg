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
	if err := SaveProvider("one.example", GH); err != nil {
		t.Fatal(err)
	}
	if err := SaveProvider("git.example.com", GLab); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Hosts) != 2 || cfg.Hosts["one.example"] != "gh" || cfg.Hosts["git.example.com"] != "glab" {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
}

func TestSaveProviderRenameFailurePreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GG_HOME", dir)
	if err := SaveProvider("one.example", GH); err != nil {
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

func TestBrokenProviderSettingSchemaIsNotOverwritten(t *testing.T) {
	for _, broken := range []string{
		"null",
		`{}`,
		`{"hosts":null}`,
		`[]`,
		`{"hosts":{},"token":"secret"}`,
		`{"hosts":{},"login":"user"}`,
		`{"hosts":{},"repository":"https://git.example.com/o/r"}`,
	} {
		t.Run(broken, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GG_HOME", dir)
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfig(); err == nil {
				t.Fatal("broken Provider Setting schema should fail")
			}
			if err := SaveProvider("git.example.com", GH); err == nil {
				t.Fatal("broken Provider Setting schema should not be overwritten")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != broken {
				t.Fatalf("broken config changed to %q", got)
			}
		})
	}
}

func TestBrokenProviderSettingEntriesAreNotOverwritten(t *testing.T) {
	brokenConfigs := []string{
		`{"hosts":{"bad.example":"token"}}`,
		`{"hosts":{"Git.Example.com":"gh"}}`,
		`{"hosts":{"git.example.com:8443":"glab"}}`,
		`{"hosts":{"https://git.example.com/o/r":"tea"}}`,
		`{"hosts":{"github.com":"tea"}}`,
	}
	for _, broken := range brokenConfigs {
		t.Run(broken, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GG_HOME", dir)
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfig(); err == nil {
				t.Fatal("broken Provider Setting entry should fail")
			}
			if err := SaveProvider("good.example", GH); err == nil {
				t.Fatal("broken Provider Setting entry should not be overwritten")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != broken {
				t.Fatalf("broken config changed to %q", got)
			}
		})
	}
}

func TestProviderSettingHostNormalization(t *testing.T) {
	cases := map[string]string{
		"Git.Example.COM":      "git.example.com",
		"Git.Example.COM:8443": "git.example.com",
		"localhost:3000":       "localhost",
		"[2001:DB8::1]:2222":   "2001:db8::1",
	}
	for input, want := range cases {
		got, err := NormalizeConfigHost(input)
		if err != nil || got != want {
			t.Errorf("NormalizeConfigHost(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	host, err := NormalizeConfigHost("[2001:DB8::1]:2222")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded, err := NormalizeConfigHost(host); err != nil || reloaded != host {
		t.Errorf("stored IPv6 host %q reload = %q, %v", host, reloaded, err)
	}
}

func TestProviderSettingHostNormalizationRejectsInvalidInput(t *testing.T) {
	inputs := []string{
		"", "https://git.example.com", "git.example.com/group/repo",
		"git.example.com:", "git.example.com:not-a-port", "git.example.com:0", "git.example.com:70000",
		"user@git.example.com", "git.example.com?x=1", "git.example.com#part",
		"git.example.com:1:2", "github.com.", "--foo", "foo..bar", " git.example.com ",
	}
	for _, input := range inputs {
		if got, err := NormalizeConfigHost(input); err == nil {
			t.Errorf("NormalizeConfigHost(%q) = %q; want error", input, got)
		}
	}
}

func TestParseProvider(t *testing.T) {
	for _, input := range []string{"gh", "glab", "tea"} {
		got, err := ParseProvider(input)
		if err != nil || string(got) != input {
			t.Errorf("ParseProvider(%q) = %q, %v", input, got, err)
		}
	}
	for _, input := range []string{"", "GH", "github", "tea "} {
		if got, err := ParseProvider(input); err == nil {
			t.Errorf("ParseProvider(%q) = %q; want error", input, got)
		}
	}
}
