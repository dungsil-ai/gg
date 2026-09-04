package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestE2EGitHubPRCommentCRUD는 pr comment 입력/조회/수정/삭제가 gh argv로
// 1:1 대응되는지 본다. 조회/수정/삭제는 gh api로 중계하고, 저장소 문맥의
// 호스트는 GH_HOST 환경변수로 전달한다.
func TestE2EGitHubPRCommentCRUD(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pr", "comment", "18", "--body", "fixed"}, "gh pr comment 18 --body fixed -R github.com/o/origin"},
		{[]string{"pr", "comment", "list", "18"}, "gh api repos/o/origin/issues/18/comments"},
		{[]string{"pr", "comment", "edit", "18", "77", "--body", "edited"}, "gh api -X PATCH repos/o/origin/issues/comments/77 -f body=edited"},
		{[]string{"pr", "comment", "delete", "18", "77"}, "gh api -X DELETE repos/o/origin/issues/comments/77"},
		{[]string{"mr", "comment", "list", "18"}, "gh api repos/o/origin/issues/18/comments"},
		{[]string{"pr", "comment", "18", "--body", "upstream-note", "--remote", "upstream"}, "gh pr comment 18 --body upstream-note -R github.com/o/upstream"},
		{[]string{"--repo", "https://github.com/custom/repo", "pr", "comment", "list", "18"}, "gh api repos/custom/repo/issues/18/comments"},
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

// TestE2EGitHubPRCommentListPassesHostEnv는 gh api 중계 시 저장소 문맥의
// 호스트가 GH_HOST로 자식 프로세스에 전달되는지 본다. 기본 도메인이 아닌
// 호스트를 쓰므로 fake gh가 auth status에도 응답한다.
func TestE2EGitHubPRCommentListPassesHostEnv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	var scriptPath, body string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(fakeDir, "gh.cmd")
		body = "@echo off\r\n" +
			"if \"%1\"==\"auth\" if \"%2\"==\"status\" (\r\n" +
			"  echo {\"hosts\":{\"ghe.corp.example\":{}}}\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"echo gh %* GH_HOST=%GH_HOST% >> \"" + logFile + "\"\r\n" +
			"exit /b 0\r\n"
	} else {
		scriptPath = filepath.Join(fakeDir, "gh")
		body = "#!/bin/sh\n" +
			"if [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n" +
			"  echo '{\"hosts\":{\"ghe.corp.example\":{}}}'\n" +
			"  exit 0\n" +
			"fi\n" +
			"echo \"gh $@ GH_HOST=$GH_HOST\" >> \"" + logFile + "\"\n" +
			"exit 0\n"
	}
	if err := os.WriteFile(scriptPath, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://ghe.corp.example/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "comment", "list", "18")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := "gh api repos/o/r/issues/18/comments GH_HOST=ghe.corp.example"
	if got := readLog(t, logFile); got != want {
		t.Errorf("gh argv = %q, want %q", got, want)
	}
}

// TestE2EGitLabPRCommentCRUD는 GitLab에서 pr comment 입력은 glab mr note로,
// 조회/수정/삭제는 glab api로 중계되는지 본다. 프로젝트 경로는 하위 그룹을
// 대비해 URL 인코딩한다.
func TestE2EGitLabPRCommentCRUD(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pr", "comment", "18", "--body", "fixed"}, "glab mr note 18 --message fixed --repo https://gitlab.com/o/r"},
		{[]string{"mr", "comment", "18", "--body", "fixed"}, "glab mr note 18 --message fixed --repo https://gitlab.com/o/r"},
		{[]string{"pr", "comment", "list", "18"}, "glab api projects/o%2Fr/merge_requests/18/notes"},
		{[]string{"pr", "comment", "edit", "18", "77", "--body", "edited"}, "glab api -X PUT projects/o%2Fr/merge_requests/18/notes/77 -f body=edited"},
		{[]string{"pr", "comment", "delete", "18", "77"}, "glab api -X DELETE projects/o%2Fr/merge_requests/18/notes/77"},
		{[]string{"pr", "comment", "list", "18", "--repo", "https://gitlab.com/grp/sub/repo"}, "glab api projects/grp%2Fsub%2Frepo/merge_requests/18/notes"},
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

// TestE2EGiteaPRCommentCreateAndUnsupported는 Gitea에서 PR 댓글 추가는
// tea comment로 중계되고 목록 조회는 미지원 오류가 나는지 본다.
func TestE2EGiteaPRCommentCreateAndUnsupported(t *testing.T) {
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
	out, code := runGG(t, bin, fakeDir, repo, "pr", "comment", "18", "--body", "fixed")
	if code != 0 {
		t.Fatalf("gg pr comment: exit %d: %s", code, out)
	}
	if got, want := readLog(t, logFile), "tea comment 18 fixed --login pub --repo o/r"; got != want {
		t.Errorf("tea argv = %q, want %q", got, want)
	}

	for _, args := range [][]string{
		{"pr", "comment", "list", "18"},
		{"pr", "comment", "edit", "18", "77", "--body", "text"},
		{"pr", "comment", "delete", "18", "77"},
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

// TestE2EPRCommentUsageErrors는 pr comment 계열 사용법 오류가 exit 2와
// usage 안내로 나오고 자식 CLI를 실행하지 않는지 본다.
func TestE2EPRCommentUsageErrors(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	for _, args := range [][]string{
		{"pr", "comment"},
		{"pr", "comment", "18"},
		{"pr", "comment", "list"},
		{"pr", "comment", "list", "1", "2"},
		{"pr", "comment", "edit"},
		{"pr", "comment", "edit", "18"},
		{"pr", "comment", "delete"},
		{"pr", "comment", "delete", "18"},
	} {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), args...)
		if code != 2 {
			t.Errorf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "usage: gg pr comment") {
			t.Errorf("gg %v output에 usage 안내 없음: %s", args, out)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("child command should not run, got %q", got)
	}
}
