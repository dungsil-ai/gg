package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EIssueDeleteArgv(t *testing.T) {
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
			args:     []string{"issue", "delete", "42"},
			want:     wantCall("gh", "issue", "delete", "42", "-R", "github.com/o/r"),
		},
		{
			name:     "github delete with yes",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"issue", "delete", "42", "--yes"},
			want:     wantCall("gh", "issue", "delete", "42", "--yes", "-R", "github.com/o/r"),
		},
		{
			name:     "gitlab delete",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"issue", "delete", "42"},
			want:     wantCall("glab", "issue", "delete", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:     "github delete repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "issue", "delete", "42", "--yes"},
			want:     wantCall("gh", "issue", "delete", "42", "--yes", "-R", "github.com/custom/repo"),
		},
		{
			name:     "gitlab delete remote flag",
			remote:   "",
			fakeName: "glab",
			args:     []string{"issue", "delete", "42", "--repo", "https://gitlab.com/custom/repo"},
			want:     wantCall("glab", "issue", "delete", "42", "--repo", "https://gitlab.com/custom/repo"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, tc.fakeName, logFile, "", "", 0)
			workDir := t.TempDir()
			if tc.remote != "" {
				workDir = tempRepo(t, tc.remote)
			}

			out, code := runGG(t, bin, fakeDir, workDir, tc.args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("argv = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestE2EIssueDeleteTeaUnsupported(t *testing.T) {
	bin := buildGG(t)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	t.Run("tea login이 있어도 미지원 오류", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeTeaWithLogin(t, fakeDir, logFile)

		out, code := runGG(t, bin, fakeDir, repo, "issue", "delete", "42")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "issue delete is not supported for tea") {
			t.Errorf("output에 미지원 오류 없음: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("tea should not run, got %q", got)
		}
	})

	t.Run("tea login이 없어도 미지원 오류가 먼저 온다", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeBin(t, fakeDir, "tea", logFile)

		out, code := runGG(t, bin, fakeDir, repo, "issue", "delete", "42")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "issue delete is not supported for tea") {
			t.Errorf("output에 미지원 오류 없음: %s", out)
		}
		if strings.Contains(out, "no tea login") {
			t.Errorf("unsupported action은 tea login을 묻지 않아야 함: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("tea should not run, got %q", got)
		}
	})
}

func TestE2EIssueDeleteUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "delete missing number",
			args: []string{"issue", "delete"},
			want: "usage: gg issue delete <number>",
		},
		{
			name: "delete too many positional args",
			args: []string{"issue", "delete", "1", "2"},
			want: "usage: gg issue delete <number>",
		},
		{
			name: "delete unknown flag",
			args: []string{"issue", "delete", "42", "--invalid"},
			want: "unknown flag",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, t.TempDir(), tc.args...)
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

func TestE2EIssueDeleteHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "delete", "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{"gg issue delete <number> [flags]", "--yes", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
