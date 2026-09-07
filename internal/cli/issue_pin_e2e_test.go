package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EIssuePinUnpinArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github pin",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "pin", "42"},
			want:     "gh issue pin 42 -R github.com/o/r",
		},
		{
			name:     "github unpin",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "unpin", "42"},
			want:     "gh issue unpin 42 -R github.com/o/r",
		},
		{
			name:     "github pin repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "issue", "pin", "42"},
			want:     "gh issue pin 42 -R github.com/custom/repo",
		},
		{
			name:     "github unpin remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"issue", "unpin", "42", "--remote", "upstream"},
			want:     "gh issue unpin 42 -R github.com/o/upstream",
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

func TestE2EIssuePinUnpinUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab pin",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "pin", "42"},
			want:     "issue pin is not supported for glab",
		},
		{
			name:     "glab unpin",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "unpin", "42"},
			want:     "issue unpin is not supported for glab",
		},
		{
			name:     "tea pin",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "pin", "42"},
			want:     "issue pin is not supported for tea",
		},
		{
			name:     "tea unpin",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "unpin", "42"},
			want:     "issue unpin is not supported for tea",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
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

func TestE2EIssuePinUnpinUsageErrors(t *testing.T) {
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
			name: "pin missing number",
			args: []string{"issue", "pin"},
			want: "usage: gg issue pin <number>",
		},
		{
			name: "pin too many positional args",
			args: []string{"issue", "pin", "1", "2"},
			want: "usage: gg issue pin <number>",
		},
		{
			name: "pin unknown flag",
			args: []string{"issue", "pin", "42", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "unpin missing number",
			args: []string{"issue", "unpin"},
			want: "usage: gg issue unpin <number>",
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

func TestE2EIssuePinUnpinExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "issue", "pin", "42"},
		{"issue", "pin", "42", "--explain"},
		{"issue", "unpin", "42", "--explain"},
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

func TestE2EIssuePinUnpinHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "pin", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue pin --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue pin <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue pin help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "issue", "unpin", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue unpin --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue unpin <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue unpin help missing %q:\n%s", want, stdout)
		}
	}
}
