package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ERepoMirrorArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab mirror",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"repo", "mirror", "--url", "https://example.com/o/r.git"},
			want:   wantCall("glab", "repo", "mirror", "--repo", "https://gitlab.com/o/r", "--url", "https://example.com/o/r.git"),
		},
		{
			name:   "glab mirror repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "repo", "mirror", "--url", "https://example.com/o/r.git"},
			want:   wantCall("glab", "repo", "mirror", "--repo", "https://gitlab.com/custom/repo", "--url", "https://example.com/o/r.git"),
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

func TestE2ERepoMirrorUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh mirror",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"repo", "mirror", "--url", "https://example.com/o/r.git"},
			want:     "repo does not support mirror",
		},
		{
			name:     "tea mirror",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"repo", "mirror", "--url", "https://example.com/o/r.git"},
			want:     "repo does not support mirror",
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

func TestE2ERepoMirrorUsageErrors(t *testing.T) {
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
			name: "mirror without --url",
			args: []string{"repo", "mirror"},
			want: "repo mirror needs --url <url>",
		},
		{
			name: "mirror with blank --url",
			args: []string{"repo", "mirror", "--url", "  "},
			want: "repo mirror needs --url <url>",
		},
		{
			name: "mirror unknown flag",
			args: []string{"repo", "mirror", "--url", "https://example.com/o/r.git", "--invalid"},
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

func TestE2ERepoMirrorExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "repo", "mirror", "--url", "https://example.com/o/r.git"},
		{"repo", "mirror", "--url", "https://example.com/o/r.git", "--explain"},
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

func TestE2ERepoMirrorHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "repo", "mirror", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg repo mirror --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg repo mirror --url <url> [flags]", "--url <url>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("repo mirror help missing %q:\n%s", want, stdout)
		}
	}
}
