package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRDiffArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name        string
		originURL   string
		upstreamURL string
		fakeName    string
		args        []string
		want        string
	}{
		{
			name:      "github diff",
			originURL: "https://github.com/o/r.git",
			fakeName:  "gh",
			args:      []string{"pr", "diff", "42"},
			want:      "gh pr diff 42 -R github.com/o/r",
		},
		{
			name:      "github diff repo flag",
			originURL: "https://github.com/o/unused.git",
			fakeName:  "gh",
			args:      []string{"--repo", "https://github.com/custom/repo", "pr", "diff", "42"},
			want:      "gh pr diff 42 -R github.com/custom/repo",
		},
		{
			name:      "gitlab diff",
			originURL: "https://gitlab.com/o/r.git",
			fakeName:  "glab",
			args:      []string{"pr", "diff", "42"},
			want:      "glab mr diff 42 --repo https://gitlab.com/o/r",
		},
		{
			name:      "gitlab diff mr alias",
			originURL: "https://gitlab.com/o/r.git",
			fakeName:  "glab",
			args:      []string{"mr", "diff", "42"},
			want:      "glab mr diff 42 --repo https://gitlab.com/o/r",
		},
		{
			name:        "gitlab diff remote flag",
			originURL:   "https://gitlab.com/o/origin.git",
			upstreamURL: "https://gitlab.com/o/upstream.git",
			fakeName:    "glab",
			args:        []string{"pr", "diff", "42", "--remote", "upstream"},
			want:        "glab mr diff 42 --repo https://gitlab.com/o/upstream",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			workDir := tempRepo(t, tc.originURL)
			if tc.upstreamURL != "" {
				gitIn(t, workDir, "remote", "add", "upstream", tc.upstreamURL)
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

func TestE2EPRDiffTeaUnsupported(t *testing.T) {
	bin := buildGG(t)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	t.Run("tea login이 있어도 미지원 오류", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeTeaWithLogin(t, fakeDir, logFile)

		out, code := runGG(t, bin, fakeDir, repo, "pr", "diff", "42")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "pr diff is not supported for tea") {
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

		out, code := runGG(t, bin, fakeDir, repo, "pr", "diff", "42")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "pr diff is not supported for tea") {
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

func TestE2EPRDiffUsageErrors(t *testing.T) {
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
			name: "diff missing number",
			args: []string{"pr", "diff"},
			want: "usage: gg pr diff <number>",
		},
		{
			name: "diff too many positional args",
			args: []string{"pr", "diff", "1", "2"},
			want: "usage: gg pr diff <number>",
		},
		{
			name: "diff unknown flag",
			args: []string{"pr", "diff", "42", "--invalid"},
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

func TestE2EPRDiffExplain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		remote   string
		fakeName string
	}{
		{name: "gh", remote: "https://github.com/o/r.git", fakeName: "gh"},
		{name: "glab", remote: "https://gitlab.com/o/r.git", fakeName: "glab"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			repo := tempRepo(t, tc.remote)

			for _, args := range [][]string{
				{"--explain", "pr", "diff", "42"},
				{"pr", "diff", "42", "--explain"},
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

func TestE2EPRDiffHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "diff", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr diff --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr diff <number> [flags]", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr diff help missing %q:\n%s", want, stdout)
		}
	}
}
