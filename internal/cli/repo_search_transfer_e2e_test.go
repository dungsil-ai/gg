package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ERepoSearchTransferArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "glab search",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"repo", "search", "--search", "gg"},
			want:   "glab repo search --repo https://gitlab.com/o/r --search gg",
		},
		{
			name:   "glab transfer",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"repo", "transfer", "--target-namespace", "newgrp"},
			want:   "glab repo transfer --repo https://gitlab.com/o/r --target-namespace newgrp",
		},
		{
			name:   "glab transfer with yes",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"repo", "transfer", "--target-namespace", "newgrp", "--yes"},
			want:   "glab repo transfer --repo https://gitlab.com/o/r --target-namespace newgrp --yes",
		},
		{
			name:   "glab search repo flag",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"--repo", "https://gitlab.com/custom/repo", "repo", "search", "--search", "gg"},
			want:   "glab repo search --repo https://gitlab.com/custom/repo --search gg",
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

func TestE2ERepoSearchTransferUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh search",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"repo", "search", "--search", "gg"},
			want:     "repo does not support search",
		},
		{
			name:     "gh transfer",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"repo", "transfer", "--target-namespace", "newgrp"},
			want:     "repo does not support transfer",
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

func TestE2ERepoSearchTransferUsageErrors(t *testing.T) {
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
			name: "search without --search",
			args: []string{"repo", "search"},
			want: "repo search needs --search <text>",
		},
		{
			name: "search with blank --search",
			args: []string{"repo", "search", "--search", "  "},
			want: "repo search needs --search <text>",
		},
		{
			name: "transfer without --target-namespace",
			args: []string{"repo", "transfer"},
			want: "repo transfer needs --target-namespace <namespace>",
		},
		{
			name: "search unknown flag",
			args: []string{"repo", "search", "--search", "gg", "--invalid"},
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

func TestE2ETeaRepoSearchArgv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "repo", "search", "--search", "gg")
	if code != 0 {
		t.Fatalf("gg repo search: exit %d: %s", code, out)
	}
	if got, want := readLog(t, logFile), "tea repos search gg --login pub --repo o/r"; got != want {
		t.Errorf("tea repos search argv = %q, want %q", got, want)
	}
}

func TestE2ERepoSearchTransferExplain(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "repo", "search", "--search", "gg"},
		{"repo", "transfer", "--target-namespace", "newgrp", "--explain"},
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

func TestE2ERepoSearchTransferHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "repo", "search", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg repo search --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg repo search --search <text> [flags]", "--search <text>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("repo search help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "repo", "transfer", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg repo transfer --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg repo transfer --target-namespace <namespace> [flags]", "--target-namespace <namespace>", "--yes", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("repo transfer help missing %q:\n%s", want, stdout)
		}
	}
}
