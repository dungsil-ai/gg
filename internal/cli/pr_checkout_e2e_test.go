package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRCheckoutArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github checkout",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "checkout", "42"},
			want:     "gh pr checkout 42 -R github.com/o/r",
		},
		{
			name:     "github checkout repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "pr", "checkout", "42"},
			want:     "gh pr checkout 42 -R github.com/custom/repo",
		},
		{
			name:     "gitlab checkout",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "checkout", "42"},
			want:     "glab mr checkout 42 --repo https://gitlab.com/o/r",
		},
		{
			name:     "gitlab checkout mr alias",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"mr", "checkout", "42"},
			want:     "glab mr checkout 42 --repo https://gitlab.com/o/r",
		},
		{
			name:     "gitea checkout",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "checkout", "42"},
			want:     "tea pulls checkout 42 --login pub --repo o/r",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			remote := tc.remote
			if remote == "" {
				remote = "https://github.com/o/unused.git"
			}
			workDir := tempRepo(t, remote)

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

func TestE2EPRCheckoutUsageErrors(t *testing.T) {
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
			name: "checkout missing number",
			args: []string{"pr", "checkout"},
			want: "usage: gg pr checkout <number>",
		},
		{
			name: "checkout too many positional args",
			args: []string{"pr", "checkout", "1", "2"},
			want: "usage: gg pr checkout <number>",
		},
		{
			name: "checkout unknown flag",
			args: []string{"pr", "checkout", "42", "--invalid"},
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

func TestE2EPRCheckoutExplain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		remote   string
		fakeName string
	}{
		{name: "gh", remote: "https://github.com/o/r.git", fakeName: "gh"},
		{name: "glab", remote: "https://gitlab.com/o/r.git", fakeName: "glab"},
		{name: "tea", remote: "https://gitea.com/o/r.git", fakeName: "tea"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			repo := tempRepo(t, tc.remote)

			for _, args := range [][]string{
				{"--explain", "pr", "checkout", "42"},
				{"pr", "checkout", "42", "--explain"},
			} {
				clearFile(t, logFile)
				out, code := runGG(t, bin, fakeDir, repo, args...)
				if code != 0 {
					t.Fatalf("gg %v: exit %d: %s", args, code, out)
				}
				if !strings.Contains(out, "Provider: "+tc.fakeName) || !strings.Contains(out, "CLI: "+tc.fakeName) {
					t.Errorf("gg %v output unexpected:\n%s", args, out)
				}
				if got := readLog(t, logFile); got != "" {
					t.Errorf("gg %v child should not run, got: %q", args, got)
				}
			}
		})
	}
}

func TestE2EPRCheckoutHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "checkout", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr checkout --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr checkout <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr checkout help missing %q:\n%s", want, stdout)
		}
	}
}
