package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECloneShorthand(t *testing.T) {
	bin := buildGG(t)
	cases := []struct {
		provider string
		config   fakeCLIConfig
		lookup   []string
		clone    []string
	}{
		{
			provider: "gh",
			config:   fakeCLIConfig{Stdout: `{"hosts":{"github.com":[{"active":true}]},"id":1}`},
			lookup:   []string{"gh", "api", "--hostname", "github.com", "repos/o/r"},
			clone:    []string{"gh", "repo", "clone", "https://github.com/o/r", "local dir"},
		},
		{
			provider: "glab",
			config:   fakeCLIConfig{Stdout: `{"id":1}`},
			lookup:   []string{"glab", "api", "--hostname", "gitlab.com", "projects/o%2Fr"},
			clone:    []string{"glab", "repo", "clone", "https://gitlab.com/o/r", "local dir"},
		},
		{
			provider: "tea",
			config:   fakeCLIConfig{TeaLogin: true, Stdout: `{"id":1}`},
			lookup:   []string{"tea", "api", "repos/o/r", "--login", "pub", "--repo", "o/r"},
			clone:    []string{"tea", "clone", "--login", "pub", "o/r", "local dir"},
		},
	}
	for _, tc := range cases {
		for _, alias := range []bool{false, true} {
			for _, explain := range []bool{false, true} {
				name := tc.provider
				if alias {
					name += " alias"
				}
				if explain {
					name += " explain"
				}
				t.Run(name, func(t *testing.T) {
					isolateCloneConfig(t)
					fakeDir, workDir := t.TempDir(), t.TempDir()
					logFile := filepath.Join(t.TempDir(), "calls.log")
					config := tc.config
					config.LogFile = logFile
					writeFakeCLI(t, fakeDir, tc.provider, config)
					args := []string{"repo", "clone", "o/r", "local dir"}
					if alias {
						args = args[1:]
					}
					if explain {
						args = append(args, "--explain")
					}
					out, code := runGGWithPath(t, bin, workDir, t.TempDir(), fakeDir, args...)
					if code != 0 {
						t.Fatalf("exit %d: %s", code, out)
					}
					calls := readLog(t, logFile)
					if !strings.Contains(calls, wantCall(tc.lookup...)) {
						t.Fatalf("저장소 조회 누락: %s", calls)
					}
					if explain {
						if strings.Contains(calls, `"clone"`) || !strings.Contains(out, "Provider: "+tc.provider) {
							t.Fatalf("explain은 조회만 실행해야 한다: %s\n%s", calls, out)
						}
					} else if !strings.HasSuffix(calls, wantCall(tc.clone...)) {
						t.Fatalf("clone argv = %s", calls)
					}
				})
			}
		}
	}
}

func TestE2ECloneShorthandInvalidDoesNotDiscover(t *testing.T) {
	bin := buildGG(t)
	for _, raw := range []string{"repo", "", "o/", "/o/r", "./o/r", "../repo", "o/../r", "o//r", "o/-r", "o/.git", "o/r?x=1", "o/r#ref", "o/r%2Fother", "o/r\\other", "o/r\n", "o/r s", "git@host/o/r"} {
		t.Run(raw, func(t *testing.T) {
			isolateCloneConfig(t)
			dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			for _, name := range []string{"gh", "glab", "tea", "git"} {
				writeFakeBin(t, dir, name, log)
			}
			out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, "clone", raw)
			if code == 0 || !strings.Contains(out, "gg:") {
				t.Fatalf("invalid shorthand accepted: exit %d: %s", code, out)
			}
			if calls := readLog(t, log); calls != "" {
				t.Fatalf("invalid shorthand started children: %s", calls)
			}
		})
	}
}

func TestE2ECloneShorthandRejectsNonRepositoryResponses(t *testing.T) {
	bin := buildGG(t)
	for _, response := range []string{"", "<html>Sign in</html>", `{}`, `null`, `{"message":"Not Found"}`, `{"id":0}`, `{"id":"bad"}`} {
		t.Run(response, func(t *testing.T) {
			isolateCloneConfig(t)
			dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			writeFakeCLI(t, dir, "gh", fakeCLIConfig{LogFile: log, Stdout: response, Rules: []fakeCLIRule{
				{Args: []string{"auth", "status", "--json", "hosts"}, Stdout: `{"hosts":{"github.com":[]}}`},
			}})
			out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, "clone", "o/r")
			if code != 1 || !strings.Contains(out, "cannot find an accessible repository") {
				t.Fatalf("exit %d: %s", code, out)
			}
			if calls := readLog(t, log); strings.Contains(calls, `"clone"`) {
				t.Fatalf("invalid response caused clone: %s", calls)
			}
		})
	}
}

func TestE2ECloneShorthandDiscoversConfiguredHosts(t *testing.T) {
	bin := buildGG(t)
	for _, source := range []string{"gg config", "GH_HOST", "GITLAB_HOST", "glab config path", "stale provider"} {
		t.Run(source, func(t *testing.T) {
			isolateCloneConfig(t)
			dir, home := t.TempDir(), t.TempDir()
			log := filepath.Join(t.TempDir(), "calls.log")
			provider := "gh"
			rules := []fakeCLIRule{}
			switch source {
			case "GH_HOST":
				t.Setenv("GH_HOST", "git.example.com")
			case "GITLAB_HOST":
				provider = "glab"
				t.Setenv("GITLAB_HOST", "https://git.example.com")
			case "glab config path":
				provider = "glab"
				path := filepath.Join(t.TempDir(), "config.yml")
				if err := os.WriteFile(path, []byte("hosts:\n  git.example.com:\n    token: fixture-secret\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				rules = append(rules, fakeCLIRule{Args: []string{"config", "path"}, Stdout: path})
			case "gg config", "stale provider":
				saved := "gh"
				if source == "stale provider" {
					saved = "glab"
					rules = append(rules, fakeCLIRule{Args: []string{"auth", "status", "--json", "hosts"}, Stdout: `{"hosts":{"git.example.com":[]}}`})
				}
				if out, code := runGGWithPath(t, bin, t.TempDir(), home, dir, "config", "set", "git.example.com", saved); code != 0 {
					t.Fatalf("config: %d: %s", code, out)
				}
			}
			if provider == "gh" {
				rules = append(rules, fakeCLIRule{Args: []string{"api", "--hostname", "git.example.com", "repos/o/r"}, Stdout: `{"id":1}`})
			} else {
				rules = append(rules, fakeCLIRule{Args: []string{"api", "--hostname", "git.example.com", "projects/o%2Fr"}, Stdout: `{"id":1}`})
			}
			rules = append(rules, fakeCLIRule{Args: []string{"repo", "clone", "https://git.example.com/o/r"}})
			writeFakeCLI(t, dir, provider, fakeCLIConfig{LogFile: log, ExitCode: 1, Rules: rules})
			out, code := runGGWithPath(t, bin, t.TempDir(), home, dir, "clone", "o/r")
			if code != 0 || strings.Contains(out, "fixture-secret") {
				t.Fatalf("exit %d: %s", code, out)
			}
			if got := readLog(t, log); !strings.HasSuffix(got, wantCall(provider, "repo", "clone", "https://git.example.com/o/r")) {
				t.Fatalf("clone calls: %s", got)
			}
		})
	}
}

func TestE2ECloneShorthandHTTPAndMultipleTeaAccounts(t *testing.T) {
	bin := buildGG(t)
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "blocked HTTP", true: "allowed HTTP"}[allow], func(t *testing.T) {
			isolateCloneConfig(t)
			dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			writeFakeCLI(t, dir, "tea", fakeCLIConfig{LogFile: log, Stdout: `{"id":1}`, Rules: []fakeCLIRule{
				{Args: []string{"logins", "list", "--output", "csv"}, Stdout: "Name,URL\nlocal,http://git.example.com:3000\n"},
			}})
			args := []string{"clone", "o/r"}
			if allow {
				args = append(args, "--allow-insecure-http")
			}
			out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, args...)
			calls := readLog(t, log)
			if allow {
				if code != 0 || !strings.HasSuffix(calls, wantCall("tea", "clone", "--login", "local", "o/r")) {
					t.Fatalf("exit %d: %s; calls %s", code, out, calls)
				}
			} else if code != 1 || !strings.Contains(out, "HTTP clone is blocked") || strings.Contains(calls, `"api"`) || strings.Contains(calls, `"clone"`) {
				t.Fatalf("blocked HTTP: %d: %s; calls %s", code, out, calls)
			}
		})
	}
	t.Run("second account", func(t *testing.T) {
		isolateCloneConfig(t)
		dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
		writeFakeCLI(t, dir, "tea", fakeCLIConfig{LogFile: log, ExitCode: 1, Rules: []fakeCLIRule{
			{Args: []string{"logins", "list", "--output", "csv"}, Stdout: "Name,URL\na,https://git.example.com\nz,https://git.example.com\n"},
			{Args: []string{"api", "repos/o/r", "--login", "z", "--repo", "o/r"}, Stdout: `{"id":1}`},
			{Args: []string{"clone", "--login", "z", "o/r"}},
		}})
		out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, "clone", "o/r")
		if code != 0 || !strings.HasSuffix(readLog(t, log), wantCall("tea", "clone", "--login", "z", "o/r")) {
			t.Fatalf("exit %d: %s; calls %s", code, out, readLog(t, log))
		}
	})
}

func TestE2ECloneMalformedDiscoveryDoesNotLeakSecrets(t *testing.T) {
	isolateCloneConfig(t)
	bin, dir, home := buildGG(t), t.TempDir(), t.TempDir()
	writeFakeCLI(t, dir, "gh", fakeCLIConfig{Stdout: "secret-gh"})
	writeFakeCLI(t, dir, "tea", fakeCLIConfig{Stdout: "Name,URL\n\"secret-tea"})
	writeFakeCLI(t, dir, "glab", fakeCLIConfig{ExitCode: 1})
	if err := os.WriteFile(filepath.Join(os.Getenv("GLAB_CONFIG_DIR"), "config.yml"), []byte("hosts: [secret-glab"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, code := runGGWithPath(t, bin, t.TempDir(), home, dir, "clone", "o/r")
	if code != 1 || strings.Contains(out, "secret-") {
		t.Fatalf("discovery leaked credentials: exit %d: %s", code, out)
	}
	for _, want := range []string{"gh hosts", "glab hosts", "tea logins"} {
		if !strings.Contains(out, want) {
			t.Errorf("diagnostic %q missing: %s", want, out)
		}
	}
}

func TestE2ECloneShorthandAmbiguityDoesNotClone(t *testing.T) {
	isolateCloneConfig(t)
	bin, fakeDir := buildGG(t), t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeCLI(t, fakeDir, "gh", fakeCLIConfig{LogFile: logFile, Stdout: `{"hosts":{"github.com":[]},"id":1}`})
	writeFakeCLI(t, fakeDir, "glab", fakeCLIConfig{LogFile: logFile, Stdout: `{"id":1}`})
	out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), fakeDir, "clone", "o/r")
	if code != 1 || !strings.Contains(out, "multiple repositories match") ||
		!strings.Contains(out, "https://github.com/o/r") || !strings.Contains(out, "https://gitlab.com/o/r") {
		t.Fatalf("exit %d: %s", code, out)
	}
	if calls := readLog(t, logFile); strings.Contains(calls, `"clone"`) {
		t.Fatalf("중복 후보에서 clone을 실행하면 안 된다: %s", calls)
	}
}

func TestE2ECloneShorthandMissingDoesNotClone(t *testing.T) {
	isolateCloneConfig(t)
	bin, fakeDir := buildGG(t), t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeCLI(t, fakeDir, "gh", fakeCLIConfig{LogFile: logFile, ExitCode: 1})
	out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), fakeDir, "clone", "o/missing")
	if code != 1 || !strings.Contains(out, "cannot find an accessible repository") {
		t.Fatalf("exit %d: %s", code, out)
	}
	if calls := readLog(t, logFile); strings.Contains(calls, `"clone"`) {
		t.Fatalf("저장소를 찾지 못하면 clone을 실행하면 안 된다: %s", calls)
	}
}
