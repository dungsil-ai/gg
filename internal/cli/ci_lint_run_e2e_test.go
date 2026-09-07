package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECILintRunArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab lint",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "lint"},
			want:   "glab ci lint --repo https://gitlab.com/o/r",
		},
		{
			name:   "glab run",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "run"},
			want:   "glab ci run --repo https://gitlab.com/o/r",
		},
		{
			name:   "glab run with branch",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "run", "--branch", "dev"},
			want:   "glab ci run --repo https://gitlab.com/o/r --branch dev",
		},
		{
			name:   "glab lint repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "ci", "lint"},
			want:   "glab ci lint --repo https://gitlab.com/custom/repo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			repo := tempRepo(t, tc.remote)

			out, code := runGG(t, bin, fakeDir, repo, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestE2ECILintRunUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh lint",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "lint"},
			want:     "ci does not support lint",
		},
		{
			name:     "gh run",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "run"},
			want:     "ci does not support run",
		},
		{
			name:     "tea lint",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"ci", "lint"},
			want:     "ci is not supported for tea",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			repo := tempRepo(t, tc.remote)

			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("gg %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("gg %v: stderr = %q, want substring %q", tc.args, stderr, tc.want)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("fake provider should not be called, got: %q", got)
			}
		})
	}
}

func TestE2ECILintRunUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "lint with positional arg",
			args: []string{"ci", "lint", "extra"},
			want: "unexpected argument",
		},
		{
			name: "lint unknown flag",
			args: []string{"ci", "lint", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "run unknown flag",
			args: []string{"ci", "run", "--invalid"},
			want: "unknown flag",
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
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("args %v: stderr = %q, want substring %q", tc.args, stderr, tc.want)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("fake provider should not be called, got: %q", got)
			}
		})
	}
}

func TestE2ECILintRunExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "ci", "lint"},
		{"ci", "run", "--explain"},
	} {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", args, code, out)
		}
		if !strings.Contains(out, "Provider: glab") || !strings.Contains(out, "CLI: glab") {
			t.Errorf("gg %v output unexpected:\n%s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child should not run, got: %q", args, got)
		}
	}
}

func TestE2ECILintRunHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "ci", "lint", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci lint --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci lint [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci lint help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "ci", "run", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci run --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci run [--branch <branch>] [flags]", "--branch <branch>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci run help missing %q:\n%s", want, stdout)
		}
	}
}
