package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRDeleteArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab delete",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "delete", "42"},
			want:     wantCall("glab", "mr", "delete", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:     "glab delete repo flag",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"--repo", "https://gitlab.com/custom/repo", "pr", "delete", "42"},
			want:     wantCall("glab", "mr", "delete", "42", "--repo", "https://gitlab.com/custom/repo"),
		},
		{
			name:     "glab delete remote flag",
			remote:   "upstream",
			fakeName: "glab",
			args:     []string{"pr", "delete", "42", "--remote", "upstream"},
			want:     wantCall("glab", "mr", "delete", "42", "--repo", "https://gitlab.com/o/upstream"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			workDir := tempRepo(t, "https://gitlab.com/o/r.git")
			if tc.remote == "upstream" {
				up := exec.Command("git", "-C", workDir, "remote", "add", "upstream", "https://gitlab.com/o/upstream.git")
				if out2, err := up.CombinedOutput(); err != nil {
					t.Fatalf("upstream remote 추가 실패: %v: %s", err, out2)
				}
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

func TestE2EPRDeleteUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh delete",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "delete", "42"},
			want:     "pr does not support delete",
		},
		{
			name:     "tea delete",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "delete", "42"},
			want:     "pr does not support delete",
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

func TestE2EPRDeleteUsageErrors(t *testing.T) {
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
			name: "delete missing number",
			args: []string{"pr", "delete"},
			want: "usage: gg pr delete <number>",
		},
		{
			name: "delete too many positional args",
			args: []string{"pr", "delete", "1", "2"},
			want: "usage: gg pr delete <number>",
		},
		{
			name: "delete unknown flag",
			args: []string{"pr", "delete", "42", "--invalid"},
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

func TestE2EPRDeleteExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "delete", "42"},
		{"pr", "delete", "42", "--explain"},
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

func TestE2EPRDeleteHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "delete", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr delete --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr delete <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr delete help missing %q:\n%s", want, stdout)
		}
	}
}
