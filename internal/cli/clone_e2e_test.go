package cli

import (
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
