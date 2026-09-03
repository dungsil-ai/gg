package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeReadyBin은 stdout, stderr, exitCode를 제어하고 호출 argv를 logFile에 남기는 fake provider CLI를 만든다.
func writeFakeReadyBin(t *testing.T, dir, name, logFile string, stdout, stderr string, exitCode int) {
	t.Helper()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name+".cmd")
		var b strings.Builder
		b.WriteString("@echo off\r\n")
		if logFile != "" {
			b.WriteString("echo " + name + " %* >> \"" + logFile + "\"\r\n")
		}
		if stdout != "" {
			for _, line := range strings.Split(stdout, "\n") {
				if line != "" {
					b.WriteString("(echo " + line + ")\r\n")
				}
			}
		}
		if stderr != "" {
			for _, line := range strings.Split(stderr, "\n") {
				if line != "" {
					b.WriteString("(echo " + line + ") 1>&2\r\n")
				}
			}
		}
		b.WriteString(fmt.Sprintf("exit /b %d\r\n", exitCode))
		body = b.String()
	} else {
		path = filepath.Join(dir, name)
		var b strings.Builder
		b.WriteString("#!/bin/sh\n")
		if logFile != "" {
			b.WriteString("echo \"" + name + " $@\" >> \"" + logFile + "\"\n")
		}
		if stdout != "" {
			for _, line := range strings.Split(stdout, "\n") {
				if line != "" {
					b.WriteString("echo \"" + line + "\"\n")
				}
			}
		}
		if stderr != "" {
			for _, line := range strings.Split(stderr, "\n") {
				if line != "" {
					b.WriteString("echo \"" + line + "\" >&2\n")
				}
			}
		}
		b.WriteString(fmt.Sprintf("exit %d\n", exitCode))
		body = b.String()
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// runGGStreamsWithFake는 fakeDir를 PATH 앞에 붙여 실행하고 stdout, stderr, exit code를 분리 반환한다.
func runGGStreamsWithFake(t *testing.T, bin, fakeDir, workDir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := ggCommand(t, bin, fakeDir, workDir, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := "stdout: " + stdout.String() + "\nstderr: " + stderr.String()
	return stdout.String(), stderr.String(), processExitCode(t, err, output)
}

func TestE2EPRReadyArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github ready",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "ready", "42"},
			want:     "gh pr ready 42 -R github.com/o/r",
		},
		{
			name:     "github draft",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "ready", "42", "--undo"},
			want:     "gh pr ready 42 --undo -R github.com/o/r",
		},
		{
			name:     "gitlab ready",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "ready", "42"},
			want:     "glab mr update 42 --ready --repo https://gitlab.com/o/r",
		},
		{
			name:     "gitlab draft",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "ready", "42", "--undo"},
			want:     "glab mr update 42 --draft --repo https://gitlab.com/o/r",
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

func TestE2EPRReadyRepositoryContext(t *testing.T) {
	bin := buildGG(t)

	t.Run("repo flag github ready", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)

		out, code := runGG(t, bin, fakeDir, t.TempDir(),
			"--repo", "https://github.com/custom/repo", "pr", "ready", "42")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh pr ready 42 -R github.com/custom/repo" {
			t.Errorf("argv = %q, want %q", got, "gh pr ready 42 -R github.com/custom/repo")
		}
	})

	t.Run("repo flag gitlab draft", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)

		out, code := runGG(t, bin, fakeDir, t.TempDir(),
			"pr", "ready", "42", "--undo", "--repo", "https://gitlab.com/custom/repo")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "glab mr update 42 --draft --repo https://gitlab.com/custom/repo" {
			t.Errorf("argv = %q, want %q", got, "glab mr update 42 --draft --repo https://gitlab.com/custom/repo")
		}
	})

	t.Run("remote flag github ready", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
		repo := tempRepoWithUpstream(t)

		out, code := runGG(t, bin, fakeDir, repo, "pr", "ready", "42", "--remote", "upstream")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh pr ready 42 -R github.com/o/upstream" {
			t.Errorf("argv = %q, want %q", got, "gh pr ready 42 -R github.com/o/upstream")
		}
	})

	t.Run("remote flag github draft", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
		repo := tempRepoWithUpstream(t)

		out, code := runGG(t, bin, fakeDir, repo, "--remote", "upstream", "pr", "ready", "42", "--undo")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh pr ready 42 --undo -R github.com/o/upstream" {
			t.Errorf("argv = %q, want %q", got, "gh pr ready 42 --undo -R github.com/o/upstream")
		}
	})
}

func TestE2EPRReadyChildPassthrough(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name       string
		fakeName   string
		remote     string
		args       []string
		stdoutText string
		stderrText string
		exitCode   int
	}{
		{
			name:       "success passthrough github",
			fakeName:   "gh",
			remote:     "https://github.com/o/r.git",
			args:       []string{"pr", "ready", "42"},
			stdoutText: "gh: marked ready",
			stderrText: "",
			exitCode:   0,
		},
		{
			name:       "error passthrough with non-zero exit code github",
			fakeName:   "gh",
			remote:     "https://github.com/o/r.git",
			args:       []string{"pr", "ready", "42"},
			stdoutText: "stdout message",
			stderrText: "gh: pull request already ready",
			exitCode:   7,
		},
		{
			name:       "error passthrough gitlab",
			fakeName:   "glab",
			remote:     "https://gitlab.com/o/r.git",
			args:       []string{"pr", "ready", "42", "--undo"},
			stdoutText: "mr draft marked",
			stderrText: "glab: warning note",
			exitCode:   4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			writeFakeReadyBin(t, fakeDir, tc.fakeName, "", tc.stdoutText, tc.stderrText, tc.exitCode)
			repo := tempRepo(t, tc.remote)

			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != tc.exitCode {
				t.Errorf("exit code = %d, want %d", code, tc.exitCode)
			}
			nl := "\n"
			if runtime.GOOS == "windows" {
				nl = "\r\n"
			}
			wantStdout := ""
			if tc.stdoutText != "" {
				wantStdout = tc.stdoutText + nl
			}
			wantStderr := ""
			if tc.stderrText != "" {
				wantStderr = tc.stderrText + nl
			}
			if stdout != wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, wantStdout)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, wantStderr)
			}
		})
	}
}

func TestE2EPRReadyUsageErrors(t *testing.T) {
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
			name: "missing number",
			args: []string{"pr", "ready"},
			want: "usage: gg pr ready <number>",
		},
		{
			name: "missing number with undo",
			args: []string{"pr", "ready", "--undo"},
			want: "usage: gg pr ready <number>",
		},
		{
			name: "too many positional args",
			args: []string{"pr", "ready", "1", "2"},
			want: "usage: gg pr ready <number>",
		},
		{
			name: "unknown flag",
			args: []string{"pr", "ready", "42", "--invalid"},
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

func TestE2EPRReadyTeaUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "tea", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "ready",
			args: []string{"pr", "ready", "42"},
		},
		{
			name: "draft undo",
			args: []string{"pr", "ready", "42", "--undo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("args %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if stdout != "" {
				t.Errorf("args %v: stdout = %q, want empty", tc.args, stdout)
			}
			if !strings.Contains(stderr, "pr ready is not supported for tea") {
				t.Errorf("args %v: stderr = %q, want substring 'pr ready is not supported for tea'", tc.args, stderr)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("tea should not be invoked, got %q", got)
			}
		})
	}
}

func TestE2EPRReadyHelp(t *testing.T) {
	bin := buildGG(t)
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "ready", "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	wants := []string{"<number>", "--undo", "--repo", "--remote"}
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
