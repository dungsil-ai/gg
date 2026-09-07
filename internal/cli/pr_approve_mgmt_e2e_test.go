package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRApproveMgmtArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab approvers",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"pr", "approvers", "42"},
			want:   "glab mr approvers 42 --repo https://gitlab.com/o/r",
		},
		{
			name:   "glab revoke",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"pr", "revoke", "42"},
			want:   "glab mr revoke 42 --repo https://gitlab.com/o/r",
		},
		{
			name:   "glab approvers repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "pr", "approvers", "42"},
			want:   "glab mr approvers 42 --repo https://gitlab.com/custom/repo",
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

func TestE2EPRApproveMgmtUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh approvers",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "approvers", "42"},
			want:     "pr does not support approvers",
		},
		{
			name:     "gh revoke",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "revoke", "42"},
			want:     "pr does not support revoke",
		},
		{
			name:     "tea approvers",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "approvers", "42"},
			want:     "pr does not support approvers",
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

func TestE2EPRApproveMgmtUsageErrors(t *testing.T) {
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
			name: "approvers missing number",
			args: []string{"pr", "approvers"},
			want: "usage: gg pr approvers <number>",
		},
		{
			name: "approvers too many positional args",
			args: []string{"pr", "approvers", "1", "2"},
			want: "usage: gg pr approvers <number>",
		},
		{
			name: "revoke missing number",
			args: []string{"pr", "revoke"},
			want: "usage: gg pr revoke <number>",
		},
		{
			name: "revoke unknown flag",
			args: []string{"pr", "revoke", "42", "--invalid"},
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

func TestE2EPRApproveMgmtExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "approvers", "42"},
		{"pr", "revoke", "42", "--explain"},
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

func TestE2EPRApproveMgmtHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "approvers", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr approvers --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr approvers <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr approvers help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "pr", "revoke", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr revoke --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr revoke <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr revoke help missing %q:\n%s", want, stdout)
		}
	}
}
