package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECIDeleteDownloadArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github delete",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "delete", "123"},
			want:     "gh run delete 123 -R github.com/o/r",
		},
		{
			name:     "github delete repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "ci", "delete", "123"},
			want:     "gh run delete 123 -R github.com/custom/repo",
		},
		{
			name:     "glab delete",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"ci", "delete", "123"},
			want:     "glab ci delete 123 --repo https://gitlab.com/o/r",
		},
		{
			name:     "github download",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "download", "123"},
			want:     "gh run download 123 -R github.com/o/r",
		},
		{
			name:     "github download with pattern and dir",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"ci", "download", "123", "--pattern", "*.zip", "--dir", "dist"},
			want:     "gh run download 123 -R github.com/o/r --pattern *.zip --dir dist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, tc.fakeName, logFile, "", "", 0)
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

func TestE2ECIDeleteDownloadUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab download",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"ci", "download", "123"},
			want:     "ci does not support download",
		},
		{
			name:     "tea delete",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"ci", "delete", "123"},
			want:     "ci is not supported for tea",
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

func TestE2ECIDeleteDownloadUsageErrors(t *testing.T) {
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
			name: "delete missing id",
			args: []string{"ci", "delete"},
			want: "usage: gg ci delete <id>",
		},
		{
			name: "delete too many positional args",
			args: []string{"ci", "delete", "1", "2"},
			want: "usage: gg ci delete <id>",
		},
		{
			name: "download missing id",
			args: []string{"ci", "download"},
			want: "usage: gg ci download <id>",
		},
		{
			name: "download unknown flag",
			args: []string{"ci", "download", "123", "--invalid"},
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

func TestE2ECIDeleteDownloadExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "ci", "delete", "123"},
		{"ci", "download", "123", "--explain"},
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

func TestE2ECIDeleteDownloadHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "ci", "delete", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci delete --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci delete <id> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci delete help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "ci", "download", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg ci download --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg ci download <id> [flags]", "--pattern <glob>", "--dir <dir>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ci download help missing %q:\n%s", want, stdout)
		}
	}
}
