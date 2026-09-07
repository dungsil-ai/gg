package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EIssueStatusArgv(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "github issue status",
			remote: "https://github.com/o/r.git",
			args:   []string{"issue", "status"},
			want:   "gh issue status -R github.com/o/r",
		},
		{
			name:   "github issue status repo flag",
			remote: "",
			args:   []string{"--repo", "https://github.com/custom/repo", "issue", "status"},
			want:   "gh issue status -R github.com/custom/repo",
		},
		{
			name:   "github issue status remote flag",
			remote: "",
			args:   []string{"issue", "status", "--remote", "upstream"},
			want:   "gh issue status -R github.com/o/upstream",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
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

func TestE2EIssueStatusUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab issue status",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "status"},
			want:     "issue status is not supported for glab",
		},
		{
			name:     "tea issue status",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "status"},
			want:     "issue status is not supported for tea",
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

func TestE2EIssueStatusUsageErrors(t *testing.T) {
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
			name: "status with positional arg",
			args: []string{"issue", "status", "42"},
			want: "unexpected argument",
		},
		{
			name: "status unknown flag",
			args: []string{"issue", "status", "--invalid"},
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

func TestE2EIssueStatusExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "issue", "status"},
		{"issue", "status", "--explain"},
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

func TestE2EIssueStatusHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "status", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue status --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue status [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue status help missing %q:\n%s", want, stdout)
		}
	}
}
