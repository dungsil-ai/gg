package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECIStatusTriggerArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab status",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "status"},
			want:   wantCall("glab", "ci", "status", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab status with branch",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "status", "--branch", "dev"},
			want:   wantCall("glab", "ci", "status", "--repo", "https://gitlab.com/o/r", "--branch", "dev"),
		},
		{
			name:   "glab trigger",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"ci", "trigger", "987"},
			want:   wantCall("glab", "ci", "trigger", "987", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab status repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "ci", "status"},
			want:   wantCall("glab", "ci", "status", "--repo", "https://gitlab.com/custom/repo"),
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

func TestE2ECIStatusTriggerUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh status",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "status"},
			want:     "ci does not support status",
		},
		{
			name:     "gh trigger",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "trigger", "987"},
			want:     "ci does not support trigger",
		},
		{
			name:     "tea status",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"ci", "status"},
			want:     "ci is not supported for tea",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeBin(t, fakeDir, "tea", logFile)
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

func TestE2ECIStatusTriggerUsageErrors(t *testing.T) {
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
			name: "status with positional arg",
			args: []string{"ci", "status", "extra"},
			want: "unexpected argument",
		},
		{
			name: "status unknown flag",
			args: []string{"ci", "status", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "trigger missing job id",
			args: []string{"ci", "trigger"},
			want: "usage: gg ci trigger <job-id>",
		},
		{
			name: "trigger too many positional args",
			args: []string{"ci", "trigger", "1", "2"},
			want: "usage: gg ci trigger <job-id>",
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

func TestE2ECIStatusTriggerExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "ci", "status"},
		{"ci", "trigger", "987", "--explain"},
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

func TestE2ECIStatusTriggerHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "ci", "status", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci status --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci status [--branch <branch>] [flags]", "--branch <branch>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci status help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "ci", "trigger", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci trigger --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci trigger <job-id> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci trigger help missing %q:\n%s", want, stdout)
		}
	}
}
