package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EIssueTransferArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github transfer",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "transfer", "42", "o/other"},
			want:     wantCall("gh", "issue", "transfer", "42", "o/other", "-R", "github.com/o/r"),
		},
		{
			name:     "github transfer repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "issue", "transfer", "42", "o/other"},
			want:     wantCall("gh", "issue", "transfer", "42", "o/other", "-R", "github.com/custom/repo"),
		},
		{
			name:     "github transfer remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"issue", "transfer", "42", "o/other", "--remote", "upstream"},
			want:     wantCall("gh", "issue", "transfer", "42", "o/other", "-R", "github.com/o/upstream"),
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

func TestE2EIssueTransferUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab transfer",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "transfer", "42", "o/other"},
			want:     "issue transfer is not supported for glab",
		},
		{
			name:     "tea transfer",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"issue", "transfer", "42", "o/other"},
			want:     "issue transfer is not supported for tea",
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

func TestE2EIssueTransferUsageErrors(t *testing.T) {
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
			name: "transfer missing destination",
			args: []string{"issue", "transfer", "42"},
			want: "usage: gg issue transfer <number> <destination-repository>",
		},
		{
			name: "transfer missing number",
			args: []string{"issue", "transfer"},
			want: "usage: gg issue transfer <number> <destination-repository>",
		},
		{
			name: "transfer too many positional args",
			args: []string{"issue", "transfer", "42", "o/other", "extra"},
			want: "usage: gg issue transfer <number> <destination-repository>",
		},
		{
			name: "transfer unknown flag",
			args: []string{"issue", "transfer", "42", "o/other", "--invalid"},
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

func TestE2EIssueTransferExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "issue", "transfer", "42", "o/other"},
		{"issue", "transfer", "42", "o/other", "--explain"},
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

func TestE2EIssueTransferHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "transfer", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg issue transfer --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg issue transfer <number> <destination-repository> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("issue transfer help missing %q:\n%s", want, stdout)
		}
	}
}
