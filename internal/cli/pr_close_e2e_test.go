package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeTeaWithLogin은 tea logins list를 응답하고 나머지 호출을 logFile에
// 남기는 fake tea CLI를 만든다.
func writeFakeTeaWithLogin(t *testing.T, dir, logFile string) {
	t.Helper()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "tea.cmd")
		body = "@echo off\r\nif \"%1\"==\"logins\" if \"%2\"==\"list\" (\r\n  echo [{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]\r\n  exit /b 0\r\n)\r\necho tea %* >> \"" + logFile + "\"\r\nexit /b 0\r\n"
	} else {
		path = filepath.Join(dir, "tea")
		body = "#!/bin/sh\nif [ \"$1\" = \"logins\" ] && [ \"$2\" = \"list\" ]; then\n  echo '[{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]'\n  exit 0\nfi\necho \"tea $@\" >> \"" + logFile + "\"\nexit 0\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestE2EPRCloseReopenArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github close",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "close", "42"},
			want:     "gh pr close 42 -R github.com/o/r",
		},
		{
			name:     "github reopen",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "reopen", "42"},
			want:     "gh pr reopen 42 -R github.com/o/r",
		},
		{
			name:     "gitlab close",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "close", "42"},
			want:     "glab mr close 42 --repo https://gitlab.com/o/r",
		},
		{
			name:     "gitlab reopen",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "reopen", "42"},
			want:     "glab mr reopen 42 --repo https://gitlab.com/o/r",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, tc.fakeName, logFile, "", "", 0)
			repo := tempRepo(t, tc.remote)

			out, code := runGG(t, bin, fakeDir, repo, tc.args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("argv = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestE2EPRCloseReopenTeaArgv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pr", "close", "42"}, "tea pulls close 42 --login pub --repo o/r"},
		{[]string{"pr", "reopen", "42"}, "tea pulls reopen 42 --login pub --repo o/r"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EPRCloseReopenRepositoryContext(t *testing.T) {
	bin := buildGG(t)

	t.Run("repo flag github close", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)

		out, code := runGG(t, bin, fakeDir, t.TempDir(),
			"--repo", "https://github.com/custom/repo", "pr", "close", "42")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh pr close 42 -R github.com/custom/repo" {
			t.Errorf("argv = %q, want %q", got, "gh pr close 42 -R github.com/custom/repo")
		}
	})

	t.Run("repo flag after command gitlab reopen", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)

		out, code := runGG(t, bin, fakeDir, t.TempDir(),
			"pr", "reopen", "42", "--repo", "https://gitlab.com/custom/repo")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "glab mr reopen 42 --repo https://gitlab.com/custom/repo" {
			t.Errorf("argv = %q, want %q", got, "glab mr reopen 42 --repo https://gitlab.com/custom/repo")
		}
	})

	t.Run("remote flag github close", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
		repo := tempRepoWithUpstream(t)

		out, code := runGG(t, bin, fakeDir, repo, "pr", "close", "42", "--remote", "upstream")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh pr close 42 -R github.com/o/upstream" {
			t.Errorf("argv = %q, want %q", got, "gh pr close 42 -R github.com/o/upstream")
		}
	})
}

func TestE2EPRCloseReopenChildPassthrough(t *testing.T) {
	bin := buildGG(t)

	t.Run("error passthrough keeps exit code and streams", func(t *testing.T) {
		fakeDir := t.TempDir()
		writeFakeReadyBin(t, fakeDir, "gh", "", "stdout message", "gh: pull request is already closed", 7)
		repo := tempRepo(t, "https://github.com/o/r.git")

		stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, "pr", "close", "42")
		if code != 7 {
			t.Errorf("exit code = %d, want 7", code)
		}
		nl := "\n"
		if runtime.GOOS == "windows" {
			nl = "\r\n"
		}
		if stdout != "stdout message"+nl {
			t.Errorf("stdout = %q", stdout)
		}
		if stderr != "gh: pull request is already closed"+nl {
			t.Errorf("stderr = %q", stderr)
		}
	})
}

func TestE2EPRCloseReopenUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "close missing number",
			args: []string{"pr", "close"},
			want: "usage: gg pr close <number>",
		},
		{
			name: "close too many positional args",
			args: []string{"pr", "close", "1", "2"},
			want: "usage: gg pr close <number>",
		},
		{
			name: "reopen missing number",
			args: []string{"pr", "reopen"},
			want: "usage: gg pr reopen <number>",
		},
		{
			name: "close unknown flag",
			args: []string{"pr", "close", "42", "--invalid"},
			want: "unknown flag",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, t.TempDir(), tc.args...)
			if code != 2 {
				t.Errorf("args %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if stdout != "" {
				t.Errorf("args %v: stdout = %q, want empty", tc.args, stdout)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("args %v: stderr = %q, want substring %q", tc.args, stderr, tc.want)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("fake provider should not be called, got: %q", got)
			}
		})
	}
}

func TestE2EPRCloseReopenHelp(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name  string
		args  []string
		usage string
	}{
		{name: "close", args: []string{"pr", "close", "--help"}, usage: "gg pr close <number> [flags]"},
		{name: "reopen", args: []string{"pr", "reopen", "--help"}, usage: "gg pr reopen <number> [flags]"},
		{name: "mr alias close", args: []string{"mr", "close", "--help"}, usage: "gg pr close <number> [flags]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), tc.args...)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}
			wants := []string{tc.usage, "<number>", "--repo", "--remote", "--explain"}
			for _, want := range wants {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout missing %q:\n%s", want, stdout)
				}
			}
		})
	}
}
