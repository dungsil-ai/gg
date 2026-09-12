package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRLockUnlockArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github lock",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "lock", "42"},
			want:     wantCall("gh", "pr", "lock", "42", "-R", "github.com/o/r"),
		},
		{
			name:     "github lock with reason",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "lock", "42", "--reason", "resolved"},
			want:     wantCall("gh", "pr", "lock", "42", "-R", "github.com/o/r", "--reason", "resolved"),
		},
		{
			name:     "github unlock",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "unlock", "42"},
			want:     wantCall("gh", "pr", "unlock", "42", "-R", "github.com/o/r"),
		},
		{
			name:     "github lock repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "pr", "lock", "42"},
			want:     wantCall("gh", "pr", "lock", "42", "-R", "github.com/custom/repo"),
		},
		{
			name:     "github unlock remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"pr", "unlock", "42", "--remote", "upstream"},
			want:     wantCall("gh", "pr", "unlock", "42", "-R", "github.com/o/upstream"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			var workDir string
			if tc.remote == "" {
				workDir = tempRepoWithUpstream(t)
			} else {
				workDir = tempRepo(t, tc.remote)
			}

			out, code := runGG(t, bin, fakeDir, workDir, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestE2EPRLockUnlockUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab lock",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "lock", "42"},
			want:     "pr lock is not supported for glab",
		},
		{
			name:     "glab unlock",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "unlock", "42"},
			want:     "pr unlock is not supported for glab",
		},
		{
			name:     "tea lock",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "lock", "42"},
			want:     "pr lock is not supported for tea",
		},
		{
			name:     "tea unlock",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "unlock", "42"},
			want:     "pr unlock is not supported for tea",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
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

func TestE2EPRLockUnlockUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "lock missing number",
			args: []string{"pr", "lock"},
			want: "usage: gg pr lock <number>",
		},
		{
			name: "lock too many positional args",
			args: []string{"pr", "lock", "1", "2"},
			want: "usage: gg pr lock <number>",
		},
		{
			name: "lock unknown flag",
			args: []string{"pr", "lock", "42", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "lock invalid reason",
			args: []string{"pr", "lock", "42", "--reason", "because"},
			want: "--reason must be off_topic, resolved, spam, or too_heated",
		},
		{
			name: "unlock missing number",
			args: []string{"pr", "unlock"},
			want: "usage: gg pr unlock <number>",
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

func TestE2EPRLockUnlockExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "lock", "42"},
		{"pr", "lock", "42", "--explain"},
		{"pr", "unlock", "42", "--explain"},
	} {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", args, code, out)
		}
		if !strings.Contains(out, "Provider: gh") || !strings.Contains(out, "CLI: gh") {
			t.Errorf("gg %v output unexpected:\n%s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child should not run, got: %q", args, got)
		}
	}
}

func TestE2EPRLockUnlockHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "lock", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr lock --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr lock <number> [flags]", "--reason <reason>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr lock help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "pr", "unlock", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr unlock --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr unlock <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr unlock help missing %q:\n%s", want, stdout)
		}
	}
}
