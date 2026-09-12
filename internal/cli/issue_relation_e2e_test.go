package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeGHWithID는 gh 호출을 LOG 파일에 기록하고 모든 호출에 FIXED id를
// 표준 출력으로 응답하는 fake gh를 만든다. 번호→database id 조회에 답하면서
// 뒤이은 관계 API 호출도 기록한다.
func writeFakeGHWithID(t *testing.T, dir, logFile, id string) {
	t.Helper()
	writeFakeCLI(t, dir, "gh", fakeCLIConfig{LogFile: logFile, Stdout: id + "\n"})
}

// writeFakeGHFailing은 호출을 LOG에 기록하고 정해진 stderr와 종료 코드로
// 실패하는 fake gh를 만든다.
func writeFakeGHFailing(t *testing.T, dir, logFile, stderr string) {
	t.Helper()
	writeFakeCLI(t, dir, "gh", fakeCLIConfig{LogFile: logFile, Stderr: stderr + "\n", ExitCode: 1})
}

func TestE2EIssueSubIssue(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHWithID(t, fakeDir, logFile, "5277108047")
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "issue", "sub-issue", "42", "--parent", "7")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	calls := readLogLines(t, logFile)
	want := []string{
		wantCall("gh", "api", "--hostname", "github.com", "repos/o/r/issues/42", "--jq", ".id"),
		wantCall("gh", "api", "--method", "POST", "repos/o/r/issues/7/sub_issues", "--hostname", "github.com", "-F", "sub_issue_id=5277108047"),
	}
	if len(calls) != len(want) {
		t.Fatalf("gh calls = %v, want %v", calls, want)
	}
	for i, line := range want {
		if calls[i] != line {
			t.Errorf("gh call %d = %q, want %q", i, calls[i], line)
		}
	}
}

// readLogLines는 fake bin이 기록한 LOG를 줄 단위로 읽는다.
func readLogLines(t *testing.T, logFile string) []string {
	t.Helper()
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestE2EIssueBlockedBy(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHWithID(t, fakeDir, logFile, "5277108047")
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "issue", "blocked-by", "42", "--blocker", "7")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	want := wantCall("gh", "api", "--method", "POST", "repos/o/r/issues/42/dependencies/blocked_by", "--hostname", "github.com", "-F", "issue_id=5277108047")
	if !strings.Contains(got, want) {
		t.Errorf("gh argv = %q, want substring %q", got, want)
	}
}

func TestE2EIssueType(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHWithID(t, fakeDir, logFile, "5277108047")
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "issue", "type", "42", "--name", "Bug")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	want := wantCall("gh", "api", "--method", "PATCH", "repos/o/r/issues/42", "--hostname", "github.com", "-F", "type=Bug")
	if got != want {
		t.Errorf("gh argv = %q, want %q", got, want)
	}
}

func TestE2EIssueRelationLookupFailure(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHFailing(t, fakeDir, logFile, "HTTP 404: Not Found")
	repo := tempRepo(t, "https://github.com/o/r.git")

	stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, "issue", "sub-issue", "42", "--parent", "7")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "cannot resolve issue 42 id") || !strings.Contains(stderr, "HTTP 404") {
		t.Errorf("stderr = %q, want cannot resolve issue 42 id with gh detail", stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if strings.Contains(readLog(t, logFile), "sub_issues") {
		t.Errorf("lookup 실패에도 관계 API를 호출함: %q", readLog(t, logFile))
	}
}

func TestE2EIssueRelationUnsupportedProviders(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct {
		remote, provider string
	}{
		{"https://gitlab.com/o/r.git", "glab"},
		{"https://gitea.com/o/r.git", "tea"},
	} {
		for _, action := range []string{"sub-issue", "blocked-by", "type"} {
			repo := tempRepo(t, tc.remote)
			args := []string{"issue", action, "42"}
			switch action {
			case "sub-issue":
				args = append(args, "--parent", "7")
			case "blocked-by":
				args = append(args, "--blocker", "7")
			case "type":
				args = append(args, "--name", "Bug")
			}
			_, stderr, code := runGGStreams(t, bin, repo, args...)
			want := "issue " + action + " is not supported for " + tc.provider
			if code != 2 || !strings.Contains(stderr, want) {
				t.Errorf("%s gg %v = exit %d, stderr %q; want exit 2 with %q", tc.provider, args, code, stderr, want)
			}
		}
	}
}

func TestE2EIssueRelationHelp(t *testing.T) {
	bin := buildGG(t)
	assertGGHelp(t, bin, []string{"issue", "--help"}, []string{"sub-issue", "blocked-by", "type"})
	assertGGHelp(t, bin, []string{"issue", "sub-issue", "--help"}, []string{"sub-issue <number> --parent <parent>", "--parent <number>", "--explain"})
	assertGGHelp(t, bin, []string{"issue", "blocked-by", "--help"}, []string{"blocked-by <number> --blocker <blocker>", "--blocker <number>"})
	assertGGHelp(t, bin, []string{"issue", "type", "--help"}, []string{"type <number> --name <name>", "--name <name>"})
}
