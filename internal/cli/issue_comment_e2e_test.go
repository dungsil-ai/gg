package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestE2EGitHubIssueCommentCRUD는 issue comment 입력/조회/수정/삭제가 gh argv로
// 1:1 대응되는지 본다. GitHub 이슈 댓글은 PR 대화 댓글과 같은 endpoint를 공유하므로
// 조회/수정/삭제 argv는 pr용과 동일하고, 호스트는 GH_HOST 환경변수로 전달한다.
func TestE2EGitHubIssueCommentCRUD(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"issue", "comment", "18", "--body", "fixed"}, "gh issue comment 18 --body fixed -R github.com/o/origin"},
		{[]string{"issue", "comment", "list", "18"}, "gh api repos/o/origin/issues/18/comments"},
		{[]string{"issue", "comment", "edit", "18", "77", "--body", "edited"}, "gh api -X PATCH repos/o/origin/issues/comments/77 -f body=edited"},
		{[]string{"issue", "comment", "delete", "18", "77"}, "gh api -X DELETE repos/o/origin/issues/comments/77"},
		{[]string{"--repo", "https://github.com/custom/repo", "issue", "comment", "list", "18"}, "gh api repos/custom/repo/issues/18/comments"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestE2EGitLabIssueCommentCRUD는 GitLab에서 issue comment 입력은 glab issue
// note로, 조회/수정/삭제는 glab api의 issue note endpoint로 중계되는지 본다.
// 프로젝트 경로는 하위 그룹을 대비해 URL 인코딩한다.
func TestE2EGitLabIssueCommentCRUD(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"issue", "comment", "18", "--body", "fixed"}, "glab issue note 18 --message fixed --repo https://gitlab.com/o/r"},
		{[]string{"issue", "comment", "list", "18"}, "glab api projects/o%2Fr/issues/18/notes"},
		{[]string{"issue", "comment", "edit", "18", "77", "--body", "edited"}, "glab api -X PUT projects/o%2Fr/issues/18/notes/77 -f body=edited"},
		{[]string{"issue", "comment", "delete", "18", "77"}, "glab api -X DELETE projects/o%2Fr/issues/18/notes/77"},
		{[]string{"issue", "comment", "list", "18", "--repo", "https://gitlab.com/grp/sub/repo"}, "glab api projects/grp%2Fsub%2Frepo/issues/18/notes"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestE2EGiteaIssueCommentCreateAndUnsupported는 Gitea에서 이슈 댓글 추가는
// tea comment로 중계되고 목록/수정/삭제는 tea login 조회 없이 미지원 오류가
// 나는지 본다.
func TestE2EGiteaIssueCommentCreateAndUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	var scriptPath, body string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(fakeDir, "tea.cmd")
		body = "@echo off\r\nif \"%1\"==\"logins\" if \"%2\"==\"list\" (\r\n  echo [{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]\r\n  exit /b 0\r\n)\r\necho tea %* >> \"" + logFile + "\"\r\nexit /b 0\r\n"
	} else {
		scriptPath = filepath.Join(fakeDir, "tea")
		body = "#!/bin/sh\nif [ \"$1\" = \"logins\" ] && [ \"$2\" = \"list\" ]; then\n  echo '[{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]'\n  exit 0\nfi\necho \"tea $@\" >> \"" + logFile + "\"\nexit 0\n"
	}
	if err := os.WriteFile(scriptPath, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	if err := os.WriteFile(logFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	out, code := runGG(t, bin, fakeDir, repo, "issue", "comment", "18", "--body", "fixed")
	if code != 0 {
		t.Fatalf("gg issue comment: exit %d: %s", code, out)
	}
	if got, want := readLog(t, logFile), "tea comment 18 fixed --login pub --repo o/r"; got != want {
		t.Errorf("tea argv = %q, want %q", got, want)
	}

	for _, args := range [][]string{
		{"issue", "comment", "list", "18"},
		{"issue", "comment", "edit", "18", "77", "--body", "text"},
		{"issue", "comment", "delete", "18", "77"},
	} {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "is not supported for tea") {
			t.Errorf("gg %v output에 미지원 오류 없음: %s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v tea should not run, got %q", args, got)
		}
	}
}

// TestE2EIssueCommentUsageErrors는 issue comment 계열 사용법 오류가 exit 2와
// usage 안내로 나오고 자식 CLI를 실행하지 않는지 본다.
func TestE2EIssueCommentUsageErrors(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	for _, args := range [][]string{
		{"issue", "comment"},
		{"issue", "comment", "18"},
		{"issue", "comment", "list"},
		{"issue", "comment", "list", "1", "2"},
		{"issue", "comment", "edit"},
		{"issue", "comment", "edit", "18"},
		{"issue", "comment", "edit", "18", "77"},
		{"issue", "comment", "delete"},
		{"issue", "comment", "delete", "18"},
	} {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), args...)
		if code != 2 {
			t.Errorf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "usage: gg issue comment") {
			t.Errorf("gg %v output에 usage 안내 없음: %s", args, out)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("child command should not run, got %q", got)
	}
}
