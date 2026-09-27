package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// savedConfig describes the on-disk contract without importing product types.
type savedConfig struct {
	Hosts map[string]string `json:"hosts"`
}

func TestE2EConfigDirectoryPrecedence(t *testing.T) {
	bin := buildGG(t)
	for _, choice := range []string{"GG_HOME", "XDG_CONFIG_HOME", "home"} {
		t.Run(choice, func(t *testing.T) {
			root := t.TempDir()
			ggHome, xdg, home := filepath.Join(root, "explicit"), filepath.Join(root, "xdg"), filepath.Join(root, "home")
			want := ggHome
			if choice != "GG_HOME" {
				ggHome, want = "", filepath.Join(xdg, "gg")
			}
			if choice == "home" {
				xdg, want = "", filepath.Join(home, ".gg")
			}
			cmd := ggCommandWithHomeAndPath(bin, root, ggHome, t.TempDir(), "config", "set", "example.invalid", "gh")
			cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+xdg, "HOME="+home, "USERPROFILE="+home)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("config set: %v: %s", err, out)
			}
			data, err := os.ReadFile(filepath.Join(want, "config.json"))
			if err != nil || !strings.Contains(string(data), `"example.invalid": "gh"`) {
				t.Fatalf("config at %s: %s, %v", want, data, err)
			}
			cmd = ggCommandWithHomeAndPath(bin, root, ggHome, t.TempDir(), "config", "list")
			cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+xdg, "HOME="+home, "USERPROFILE="+home)
			if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "example.invalid") {
				t.Fatalf("config reload: %v: %s", err, out)
			}
		})
	}
}

func TestE2EConfigRejectsInvalidHostsAndProviders(t *testing.T) {
	bin, home := buildGG(t), t.TempDir()
	for _, host := range []string{"", "https://git.example.com", "git.example.com/group/repo", "git.example.com:", "git.example.com:not-a-port", "git.example.com:0", "git.example.com:70000", "user@git.example.com", "git.example.com?x=1", "git.example.com#part", "git.example.com:1:2", "github.com.", "--foo", "foo..bar", " git.example.com "} {
		if out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), home, "config", "set", host, "gh"); code != 2 {
			t.Errorf("host %q: exit %d: %s", host, code, out)
		}
	}
	for _, provider := range []string{"", "GH", "github", "tea "} {
		if out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), home, "config", "set", "example.invalid", provider); code != 2 {
			t.Errorf("provider %q: exit %d: %s", provider, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("invalid input created config: %v", err)
	}
}

func TestE2EConfigPreservesInvalidSchemasAndEntries(t *testing.T) {
	bin := buildGG(t)
	for _, content := range []string{
		"null", `{}`, `{"hosts":null}`, `[]`, `{"hosts":{},"token":"secret"}`,
		`{"hosts":{},"login":"user"}`, `{"hosts":{},"repository":"https://git.example.com/o/r"}`,
		`{"hosts":{"bad.example":"token"}}`, `{"hosts":{"Git.Example.com":"gh"}}`,
		`{"hosts":{"git.example.com:8443":"glab"}}`, `{"hosts":{"https://git.example.com/o/r":"tea"}}`,
		`{"hosts":{"github.com":"tea"}}`,
	} {
		t.Run(content, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, "config.json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"config", "list"}, {"config", "set", "other.example", "gh"}, {"config", "unset", "other.example"}} {
				if out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), home, args...); code != 1 || !strings.Contains(out, "broken config") {
					t.Fatalf("%v: exit %d: %s", args, code, out)
				}
				data, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(data, []byte(content)) {
					t.Fatalf("config changed: %s, %v", data, err)
				}
			}
		})
	}
}
