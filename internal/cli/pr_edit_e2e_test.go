package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPREditArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		origin   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "github edit title and body",
			origin:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "edit", "42", "--title", "t2", "--body", "b2"},
			want:     wantCall("gh", "pr", "edit", "42", "-R", "github.com/o/r", "--title", "t2", "--body", "b2"),
		},
		{
			name:     "github edit body only",
			origin:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "edit", "42", "--body", "b2"},
			want:     wantCall("gh", "pr", "edit", "42", "-R", "github.com/o/r", "--body", "b2"),
		},
		{
			name:     "github edit repo flag",
			origin:   "https://github.com/o/unused.git",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "pr", "edit", "42", "--title", "t"},
			want:     wantCall("gh", "pr", "edit", "42", "-R", "github.com/custom/repo", "--title", "t"),
		},
		{
			name:     "gitlab edit title and body",
			origin:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "edit", "42", "--title", "t", "--body", "b"},
			want:     wantCall("glab", "mr", "update", "42", "--repo", "https://gitlab.com/o/r", "--title", "t", "--description", "b"),
		},
		{
			name:     "gitlab edit mr alias",
			origin:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"mr", "edit", "42", "--title", "t"},
			want:     wantCall("glab", "mr", "update", "42", "--repo", "https://gitlab.com/o/r", "--title", "t"),
		},
		{
			name:     "gitea edit title and body",
			origin:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "edit", "42", "--title", "t", "--body", "b"},
			want:     wantTeaCall("pulls", "edit", "42", "--login", "pub", "--repo", "o/r", "--title", "t", "--description", "b"),
		},
		{
			name:   "github preserves title and body argument boundaries",
			origin: "https://github.com/o/r.git",
			args:   []string{"pr", "edit", "42", "--title", "hello \"world\"", "--body", "line one\nline two\t끝\\"},
			want:   wantCall("gh", "pr", "edit", "42", "-R", "github.com/o/r", "--title", "hello \"world\"", "--body", "line one\nline two\t끝\\"),
		},
		{
			name:   "gitlab preserves title and body argument boundaries",
			origin: "https://gitlab.com/o/r.git",
			args:   []string{"pr", "edit", "42", "--title", "hello \"world\"", "--body", "line one\nline two\t끝\\"},
			want:   wantCall("glab", "mr", "update", "42", "--repo", "https://gitlab.com/o/r", "--title", "hello \"world\"", "--description", "line one\nline two\t끝\\"),
		},
		{
			name:   "gitea preserves title and body argument boundaries",
			origin: "https://gitea.com/o/r.git",
			args:   []string{"pr", "edit", "42", "--title", "hello \"world\"", "--body", "line one\nline two\t끝\\"},
			want:   wantTeaCall("pulls", "edit", "42", "--login", "pub", "--repo", "o/r", "--title", "hello \"world\"", "--description", "line one\nline two\t끝\\"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			repo := tempRepo(t, tc.origin)

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

func TestE2EPREditUsageErrors(t *testing.T) {
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
			name: "edit missing number",
			args: []string{"pr", "edit"},
			want: "usage: gg pr edit <number>",
		},
		{
			name: "edit without flags",
			args: []string{"pr", "edit", "42"},
			want: "pr edit needs --title or --body",
		},
		{
			name: "edit with blank flags",
			args: []string{"pr", "edit", "42", "--title", "  ", "--body", " "},
			want: "pr edit needs --title or --body",
		},
		{
			name: "edit unknown flag",
			args: []string{"pr", "edit", "42", "--title", "t", "--invalid"},
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

func TestE2EPREditExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "edit", "42", "--title", "t"},
		{"pr", "edit", "42", "--body", "b", "--explain"},
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

func TestE2EPREditHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "edit", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr edit --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg pr edit <number> [flags]", "--title <text>", "--body <text>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr edit help missing %q:\n%s", want, stdout)
		}
	}
}
