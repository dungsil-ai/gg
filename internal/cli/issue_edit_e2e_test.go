package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EIssueEditRelaysTitleAndBody는 이슈 제목·본문 수정이 gh issue edit와
// glab issue update로 1:1 대응되는지 본다. glab은 본문 flag로 --description을 쓴다.
func TestE2EIssueEditRelaysTitleAndBody(t *testing.T) {
	cases := []struct {
		name   string
		runner string
		repo   string
		args   []string
		want   string
	}{
		{"github both", "gh", "https://github.com/o/r.git",
			[]string{"issue", "edit", "18", "--title", "T", "--body", "B"},
			"gh issue edit 18 -R github.com/o/r --title T --body B"},
		{"github title only", "gh", "https://github.com/o/r.git",
			[]string{"issue", "edit", "18", "--title", "T"},
			"gh issue edit 18 -R github.com/o/r --title T"},
		{"gitlab both", "glab", "https://gitlab.com/o/r.git",
			[]string{"issue", "edit", "18", "--title", "T", "--body", "B"},
			"glab issue update 18 --repo https://gitlab.com/o/r --title T --description B"},
		{"gitlab repo flag", "glab", "https://gitlab.com/o/r.git",
			[]string{"--repo", "https://gitlab.com/custom/repo", "issue", "edit", "18", "--body", "B"},
			"glab issue update 18 --repo https://gitlab.com/custom/repo --description B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeBin(t, fakeDir, tc.runner, logFile)
			repo := tempRepo(t, tc.repo)

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

func TestE2EIssueEditTeaUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "issue", "edit", "18", "--title", "T")
	if code != 2 || !strings.Contains(out, "issue edit is not supported for tea") {
		t.Fatalf("exit %d, output %s", code, out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea should not run, got %q", got)
	}
}

func TestE2EIssueEditUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"issue", "edit"}, "usage: gg issue edit <number>"},
		{[]string{"issue", "edit", "18"}, "issue edit needs --title or --body"},
		{[]string{"issue", "edit", "18", "extra", "--title", "T"}, "usage: gg issue edit <number>"},
	}
	for _, tc := range cases {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), tc.args...)
		if code != 2 || !strings.Contains(out, tc.want) {
			t.Errorf("gg %v = exit %d, output %s; want exit 2 with %q", tc.args, code, out, tc.want)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("gh should not run, got %q", got)
	}
}
