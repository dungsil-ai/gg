package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestParseRequestPRMerge(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "기본 병합", args: []string{"pr", "merge", "42"},
			want: Request{Resource: "pr", Action: "merge", Number: "42"}},
		{name: "squash", args: []string{"pr", "merge", "42", "--squash"},
			want: Request{Resource: "pr", Action: "merge", Number: "42", Squash: true}},
		{name: "rebase", args: []string{"pr", "merge", "42", "--rebase"},
			want: Request{Resource: "pr", Action: "merge", Number: "42", Rebase: true}},
		{name: "merge 방식", args: []string{"pr", "merge", "42", "--merge"},
			want: Request{Resource: "pr", Action: "merge", Number: "42", Merge: true}},
		{name: "branch 삭제와 자동 병합", args: []string{"pr", "merge", "42", "--delete-branch", "--auto"},
			want: Request{Resource: "pr", Action: "merge", Number: "42", DeleteBranch: true, Auto: true}},
		{name: "repo flag 뒤", args: []string{"pr", "merge", "42", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "pr", Action: "merge", Number: "42", RepoFlag: "https://github.com/o/r"}},
	}
	for _, c := range cases {
		got, err := ParseRequest(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestParseRequestPRMergeErrors(t *testing.T) {
	bad := [][]string{
		{"pr", "merge"},                                         // number 없음
		{"pr", "merge", "1", "2"},                               // 인자 초과
		{"pr", "merge", "1", "--merge", "--squash"},             // 방식 둘
		{"pr", "merge", "1", "--merge", "--rebase"},             // 방식 둘
		{"pr", "merge", "1", "--squash", "--rebase"},            // 방식 둘
		{"pr", "merge", "1", "--merge", "--squash", "--rebase"}, // 방식 셋
		{"pr", "merge", "1", "--wat"},                           // 알 수 없는 flag
		{"pr", "delete", "1"},                                   // 지원 안 하는 action
	}
	for _, args := range bad {
		_, err := ParseRequest(args)
		var ue UsageError
		if !isUsage(err) {
			_ = ue
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}

func isUsage(err error) bool {
	_, ok := err.(UsageError)
	return ok
}

func TestTranslatePRMerge(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	gl := RepoURL{Host: "gitlab.com", Owner: "o", Name: "r"}
	te := RepoURL{Host: "gitea.example.com", Owner: "o", Name: "r"}

	cases := []struct {
		name string
		req  Request
		repo RepoURL
		p    Provider
		tea  string
		want Invocation
	}{
		{name: "gh 기본 병합",
			req: Request{Resource: "pr", Action: "merge", Number: "42"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "-R", "github.com/o/r"}}},
		{name: "gh merge 방식",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Merge: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "--merge", "-R", "github.com/o/r"}}},
		{name: "gh squash 방식",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Squash: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "--squash", "-R", "github.com/o/r"}}},
		{name: "gh rebase 방식",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Rebase: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "--rebase", "-R", "github.com/o/r"}}},
		{name: "gh branch 삭제",
			req: Request{Resource: "pr", Action: "merge", Number: "42", DeleteBranch: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "--delete-branch", "-R", "github.com/o/r"}}},
		{name: "gh 자동 병합",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Auto: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "merge", "42", "--auto", "-R", "github.com/o/r"}}},
		{name: "glab 기본 병합",
			req: Request{Resource: "pr", Action: "merge", Number: "42"}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "merge", "42", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r"}}},
		{name: "glab merge 방식은 별도 flag 없음",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Merge: true}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "merge", "42", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r"}}},
		{name: "glab squash 방식",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Squash: true}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "merge", "42", "--squash", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r"}}},
		{name: "glab branch 삭제",
			req: Request{Resource: "pr", Action: "merge", Number: "42", DeleteBranch: true}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "merge", "42", "--remove-source-branch", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r"}}},
		{name: "glab 자동 병합",
			req: Request{Resource: "pr", Action: "merge", Number: "42", Auto: true}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "merge", "42", "--auto-merge", "--repo", "https://gitlab.com/o/r"}}},
	}
	for _, c := range cases {
		got, err := Translate(c.req, c.repo, c.p, c.tea)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}

	if _, err := Translate(Request{Resource: "pr", Action: "merge", Number: "42"}, te, Tea, "pub"); err == nil {
		t.Error("tea pr merge는 오류를 내야 한다")
	} else if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("tea pr merge 오류 = %v, want not supported", err)
	}
}

func clearFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestE2EPRMergeGitHubArgv(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pr", "merge", "42"}, "gh pr merge 42 -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--merge"}, "gh pr merge 42 --merge -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--squash"}, "gh pr merge 42 --squash -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--rebase"}, "gh pr merge 42 --rebase -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--delete-branch"}, "gh pr merge 42 --delete-branch -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--auto"}, "gh pr merge 42 --auto -R github.com/o/r"},
		{[]string{"pr", "merge", "42", "--squash", "--delete-branch", "--auto"}, "gh pr merge 42 --squash --delete-branch --auto -R github.com/o/r"},
	}
	for _, tc := range cases {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EPRMergeGitLabArgv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pr", "merge", "42"}, "glab mr merge 42 --when-pipeline-succeeds=false --repo https://gitlab.com/o/r"},
		{[]string{"pr", "merge", "42", "--merge"}, "glab mr merge 42 --when-pipeline-succeeds=false --repo https://gitlab.com/o/r"},
		{[]string{"pr", "merge", "42", "--squash"}, "glab mr merge 42 --squash --when-pipeline-succeeds=false --repo https://gitlab.com/o/r"},
		{[]string{"pr", "merge", "42", "--delete-branch"}, "glab mr merge 42 --remove-source-branch --when-pipeline-succeeds=false --repo https://gitlab.com/o/r"},
		{[]string{"pr", "merge", "42", "--auto"}, "glab mr merge 42 --auto-merge --repo https://gitlab.com/o/r"},
		{[]string{"pr", "merge", "42", "--squash", "--delete-branch", "--auto"}, "glab mr merge 42 --squash --remove-source-branch --auto-merge --repo https://gitlab.com/o/r"},
	}
	for _, tc := range cases {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EPRMergeUsageErrorsBeforeChild(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	cases := [][]string{
		{"pr", "merge"},                              // number 없음
		{"pr", "merge", "1", "2"},                    // 인자 초과
		{"pr", "merge", "1", "--merge", "--squash"},  // 방식 둘
		{"pr", "merge", "1", "--squash", "--rebase"}, // 방식 둘
		{"pr", "merge", "1", "--merge", "--rebase"},  // 방식 둘
		{"pr", "merge", "1", "--wat"},                // 알 수 없는 flag
	}
	for _, args := range cases {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, t.TempDir(), args...)
		if code != 2 {
			t.Errorf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v: provider CLI가 실행되면 안 된다: %q", args, got)
		}
	}
}

func TestE2EPRMergeChildPassthrough(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(fakeDir, "gh.cmd")
		body = "@echo off\r\necho out-line\r\necho err-line 1>&2\r\nexit /b 7\r\n"
	} else {
		path = filepath.Join(fakeDir, "gh")
		body = "#!/bin/sh\necho out-line\necho err-line 1>&2\nexit 7\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "merge", "42")
	if code != 7 {
		t.Errorf("exit = %d, want 7: %s", code, out)
	}
	for _, want := range []string{"out-line", "err-line"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestE2EPRMergeUnsupportedForTea(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	// tea login 응답을 내는 fake
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(fakeDir, "tea.cmd")
		body = "@echo off\r\nif \"%1\"==\"logins\" if \"%2\"==\"list\" (\r\n  echo [{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]\r\n  exit /b 0\r\n)\r\necho tea %* >> \"" + logFile + "\"\r\nexit /b 0\r\n"
	} else {
		path = filepath.Join(fakeDir, "tea")
		body = "#!/bin/sh\nif [ \"$1\" = \"logins\" ] && [ \"$2\" = \"list\" ]; then\n  echo '[{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]'\n  exit 0\nfi\necho \"tea $@\" >> \"" + logFile + "\"\nexit 0\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "merge", "42")
	if code == 0 {
		t.Fatalf("tea pr merge는 실패해야 한다: %s", out)
	}
	if !strings.Contains(out, "not supported") {
		t.Errorf("output missing \"not supported\":\n%s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea에게 명령이 실행되면 안 된다: %q", got)
	}
}

func TestE2EPRMergeHelp(t *testing.T) {
	bin := buildGG(t)
	assertGGHelp(t, bin, []string{"pr", "merge", "--help"}, []string{
		"Usage:", "pr merge <number>", "--merge", "--squash", "--rebase", "--delete-branch", "--auto", "--repo", "--remote", "--help",
	})
	assertGGHelp(t, bin, []string{"--help"}, []string{"comment on, or merge pull requests"})
}
