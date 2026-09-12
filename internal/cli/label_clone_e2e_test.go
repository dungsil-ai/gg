package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ELabelCloneArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github label clone",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "clone", "o/source"},
			want:     wantCall("gh", "label", "clone", "o/source", "-R", "github.com/o/r"),
		},
		{
			name:     "github label clone with force",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "clone", "o/source", "--force"},
			want:     wantCall("gh", "label", "clone", "o/source", "-R", "github.com/o/r", "--force"),
		},
		{
			name:     "github label clone repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "label", "clone", "o/source"},
			want:     wantCall("gh", "label", "clone", "o/source", "-R", "github.com/custom/repo"),
		},
		{
			name:     "github label clone remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"label", "clone", "o/source", "--remote", "upstream"},
			want:     wantCall("gh", "label", "clone", "o/source", "-R", "github.com/o/upstream"),
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

func TestE2ELabelCloneUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab label clone",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"label", "clone", "o/source"},
			want:     "label does not support clone",
		},
		{
			name:     "tea label clone",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"label", "clone", "o/source"},
			want:     "label clone is not supported for tea",
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

func TestE2ELabelCloneUsageErrors(t *testing.T) {
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
			name: "clone missing source",
			args: []string{"label", "clone"},
			want: "usage: gg label clone <source-repository>",
		},
		{
			name: "clone too many positional args",
			args: []string{"label", "clone", "o/source", "extra"},
			want: "usage: gg label clone <source-repository>",
		},
		{
			name: "clone unknown flag",
			args: []string{"label", "clone", "o/source", "--invalid"},
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

func TestE2ELabelCloneExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "label", "clone", "o/source"},
		{"label", "clone", "o/source", "--explain"},
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

func TestE2ELabelCloneHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "label", "clone", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg label clone --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg label clone <source-repository> [flags]", "--force", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("label clone help missing %q:\n%s", want, stdout)
		}
	}
}
