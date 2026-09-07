package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EWorkflowArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "workflow list",
			remote: "https://github.com/o/r.git",
			args:   []string{"workflow", "list"},
			want:   "gh workflow list -R github.com/o/r",
		},
		{
			name:   "workflow view",
			remote: "https://github.com/o/r.git",
			args:   []string{"workflow", "view", "ci.yml"},
			want:   "gh workflow view ci.yml -R github.com/o/r",
		},
		{
			name:   "workflow view with ref",
			remote: "https://github.com/o/r.git",
			args:   []string{"workflow", "view", "ci.yml", "--ref", "dev"},
			want:   "gh workflow view ci.yml -R github.com/o/r --ref dev",
		},
		{
			name:   "workflow run",
			remote: "https://github.com/o/r.git",
			args:   []string{"workflow", "run", "release.yml", "--ref", "main"},
			want:   "gh workflow run release.yml -R github.com/o/r --ref main",
		},
		{
			name:   "workflow enable",
			remote: "https://github.com/o/r.git",
			args:   []string{"workflow", "enable", "ci.yml"},
			want:   "gh workflow enable ci.yml -R github.com/o/r",
		},
		{
			name:   "workflow disable repo flag",
			remote: "",
			args:   []string{"--repo", "https://github.com/custom/repo", "workflow", "disable", "ci.yml"},
			want:   "gh workflow disable ci.yml -R github.com/custom/repo",
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

func TestE2EWorkflowUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab workflow list",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"workflow", "list"},
			want:     "workflow does not support list",
		},
		{
			name:     "tea workflow list",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"workflow", "list"},
			want:     "workflow is not supported for tea",
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

func TestE2EWorkflowUsageErrors(t *testing.T) {
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
			name: "view missing workflow",
			args: []string{"workflow", "view"},
			want: "usage: gg workflow view <name-or-id>",
		},
		{
			name: "run missing workflow",
			args: []string{"workflow", "run"},
			want: "usage: gg workflow run <name-or-id>",
		},
		{
			name: "enable too many positional args",
			args: []string{"workflow", "enable", "a", "b"},
			want: "usage: gg workflow enable <name-or-id>",
		},
		{
			name: "disable unknown flag",
			args: []string{"workflow", "disable", "ci.yml", "--invalid"},
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

func TestE2EWorkflowExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "workflow", "list"},
		{"workflow", "view", "ci.yml", "--explain"},
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

func TestE2EWorkflowHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "workflow", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg workflow --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"list", "view", "run", "enable", "disable", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("workflow help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "workflow", "run", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg workflow run --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg workflow run <name-or-id> [flags]", "--ref <ref>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("workflow run help missing %q:\n%s", want, stdout)
		}
	}
}
