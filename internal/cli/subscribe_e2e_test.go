package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ESubscribeArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab issue subscribe",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"issue", "subscribe", "42"},
			want:   wantCall("glab", "issue", "subscribe", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab issue unsubscribe",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"issue", "unsubscribe", "42"},
			want:   wantCall("glab", "issue", "unsubscribe", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab pr subscribe",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"pr", "subscribe", "42"},
			want:   wantCall("glab", "mr", "subscribe", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab pr unsubscribe",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"pr", "unsubscribe", "42"},
			want:   wantCall("glab", "mr", "unsubscribe", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:   "glab pr subscribe repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "pr", "subscribe", "42"},
			want:   wantCall("glab", "mr", "subscribe", "42", "--repo", "https://gitlab.com/custom/repo"),
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

func TestE2ESubscribeUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh issue subscribe",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "subscribe", "42"},
			want:     "issue does not support subscribe",
		},
		{
			name:     "gh pr unsubscribe",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "unsubscribe", "42"},
			want:     "pr does not support unsubscribe",
		},
		{
			name:     "tea issue subscribe",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "subscribe", "42"},
			want:     "issue does not support subscribe",
		},
		{
			name:     "tea pr subscribe",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "subscribe", "42"},
			want:     "pr does not support subscribe",
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

func TestE2ESubscribeUsageErrors(t *testing.T) {
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
			name: "issue subscribe missing number",
			args: []string{"issue", "subscribe"},
			want: "usage: gg issue subscribe <number>",
		},
		{
			name: "issue unsubscribe too many positional args",
			args: []string{"issue", "unsubscribe", "1", "2"},
			want: "usage: gg issue unsubscribe <number>",
		},
		{
			name: "pr subscribe missing number",
			args: []string{"pr", "subscribe"},
			want: "usage: gg pr subscribe <number>",
		},
		{
			name: "pr unsubscribe unknown flag",
			args: []string{"pr", "unsubscribe", "42", "--invalid"},
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

func TestE2ESubscribeExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "issue", "subscribe", "42"},
		{"pr", "unsubscribe", "42", "--explain"},
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

func TestE2ESubscribeHelp(t *testing.T) {
	bin := buildGG(t)

	for _, action := range []string{"subscribe", "unsubscribe"} {
		stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", action, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("gg issue %s --help = stderr %q, exit %d", action, stderr, code)
		}
		for _, want := range []string{"gg issue " + action + " <number> [flags]", "--repo", "--remote", "--explain"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("issue %s help missing %q:\n%s", action, want, stdout)
			}
		}

		stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "pr", action, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("gg pr %s --help = stderr %q, exit %d", action, stderr, code)
		}
		for _, want := range []string{"gg pr " + action + " <number> [flags]", "--repo", "--remote", "--explain"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("pr %s help missing %q:\n%s", action, want, stdout)
			}
		}
	}
}
