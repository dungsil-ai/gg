package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EIssueLockUnlockArgv(t *testing.T) {
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
			args:     []string{"issue", "lock", "42"},
			want:     wantCall("gh", "issue", "lock", "42", "-R", "github.com/o/r"),
		},
		{
			name:     "github lock with reason",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "lock", "42", "--reason", "resolved"},
			want:     wantCall("gh", "issue", "lock", "42", "-R", "github.com/o/r", "--reason", "resolved"),
		},
		{
			name:     "github unlock",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "unlock", "42"},
			want:     wantCall("gh", "issue", "unlock", "42", "-R", "github.com/o/r"),
		},
		{
			name:     "github lock repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "issue", "lock", "42"},
			want:     wantCall("gh", "issue", "lock", "42", "-R", "github.com/custom/repo"),
		},
		{
			name:     "github unlock remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"issue", "unlock", "42", "--remote", "upstream"},
			want:     wantCall("gh", "issue", "unlock", "42", "-R", "github.com/o/upstream"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			workDir := tempRepo(t, tc.remote)
			if tc.remote == "" {
				workDir = tempRepoWithUpstream(t)
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

func TestE2EIssueLockUnlockUnsupported(t *testing.T) {
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
			args:     []string{"issue", "lock", "42"},
			want:     "issue lock is not supported for glab",
		},
		{
			name:     "glab unlock",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "unlock", "42"},
			want:     "issue unlock is not supported for glab",
		},
		{
			name:     "tea lock",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "lock", "42"},
			want:     "issue lock is not supported for tea",
		},
		{
			name:     "tea unlock",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "unlock", "42"},
			want:     "issue unlock is not supported for tea",
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

func TestE2EIssueLockUnlockUsageErrors(t *testing.T) {
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
			args: []string{"issue", "lock"},
			want: "usage: gg issue lock <number>",
		},
		{
			name: "lock too many positional args",
			args: []string{"issue", "lock", "1", "2"},
			want: "usage: gg issue lock <number>",
		},
		{
			name: "lock unknown flag",
			args: []string{"issue", "lock", "42", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "lock invalid reason",
			args: []string{"issue", "lock", "42", "--reason", "because"},
			want: "--reason must be off_topic, resolved, spam, or too_heated",
		},
		{
			name: "unlock missing number",
			args: []string{"issue", "unlock"},
			want: "usage: gg issue unlock <number>",
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

func TestE2EIssueLockUnlockExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "issue", "lock", "42"},
		{"issue", "lock", "42", "--explain"},
		{"issue", "unlock", "42", "--explain"},
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

func TestE2EIssueLockUnlockHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "lock", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue lock --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue lock <number> [flags]", "--reason <reason>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue lock help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "issue", "unlock", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue unlock --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue unlock <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue unlock help missing %q:\n%s", want, stdout)
		}
	}
}
