package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRCleanArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "tea clean",
			remote: "https://gitea.com/o/r.git",
			args:   []string{"pr", "clean", "42"},
			want:   "tea pulls clean 42 --login pub --repo o/r",
		},
		{
			name:   "tea clean repo flag",
			remote: "https://gitea.com/o/r.git",
			args:   []string{"--repo", "https://gitea.com/custom/repo", "pr", "clean", "42"},
			want:   "tea pulls clean 42 --login pub --repo custom/repo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeTeaWithLogin(t, fakeDir, logFile)
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

func TestE2EPRCleanUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh clean",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "clean", "42"},
			want:     "pr does not support clean",
		},
		{
			name:     "glab clean",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "clean", "42"},
			want:     "pr does not support clean",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
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

func TestE2EPRCleanUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "clean missing number",
			args: []string{"pr", "clean"},
			want: "usage: gg pr clean <number>",
		},
		{
			name: "clean too many positional args",
			args: []string{"pr", "clean", "1", "2"},
			want: "usage: gg pr clean <number>",
		},
		{
			name: "clean unknown flag",
			args: []string{"pr", "clean", "42", "--invalid"},
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

func TestE2EPRCleanExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "clean", "42"},
		{"pr", "clean", "42", "--explain"},
	} {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", args, code, out)
		}
		if !strings.Contains(out, "Provider: tea") || !strings.Contains(out, "CLI: tea") {
			t.Errorf("gg %v output unexpected:\n%s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child should not run, got: %q", args, got)
		}
	}
}

func TestE2EPRCleanHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "clean", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr clean --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr clean <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr clean help missing %q:\n%s", want, stdout)
		}
	}
}
