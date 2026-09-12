package cli

import (
	"errors"
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

	teaMerge := []struct {
		name string
		req  Request
		want Invocation
	}{
		{name: "tea 기본 병합",
			req:  Request{Resource: "pr", Action: "merge", Number: "42"},
			want: Invocation{Bin: "tea", Args: []string{"pulls", "merge", "42", "--login", "pub", "--repo", "o/r"}}},
		{name: "tea squash 방식",
			req:  Request{Resource: "pr", Action: "merge", Number: "42", Squash: true},
			want: Invocation{Bin: "tea", Args: []string{"pulls", "merge", "42", "--style", "squash", "--login", "pub", "--repo", "o/r"}}},
		{name: "tea rebase 방식",
			req:  Request{Resource: "pr", Action: "merge", Number: "42", Rebase: true},
			want: Invocation{Bin: "tea", Args: []string{"pulls", "merge", "42", "--style", "rebase", "--login", "pub", "--repo", "o/r"}}},
		{name: "tea merge 방식",
			req:  Request{Resource: "pr", Action: "merge", Number: "42", Merge: true},
			want: Invocation{Bin: "tea", Args: []string{"pulls", "merge", "42", "--style", "merge", "--login", "pub", "--repo", "o/r"}}},
	}
	for _, c := range teaMerge {
		got, err := Translate(c.req, te, Tea, "pub")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}

	for _, tc := range []struct {
		name string
		req  Request
		want string
	}{
		{name: "tea 자동 병합", req: Request{Resource: "pr", Action: "merge", Number: "42", Auto: true},
			want: "pr merge --auto/--delete-branch is not supported for tea"},
		{name: "tea branch 삭제", req: Request{Resource: "pr", Action: "merge", Number: "42", DeleteBranch: true},
			want: "pr merge --auto/--delete-branch is not supported for tea"},
	} {
		_, err := Translate(tc.req, te, Tea, "pub")
		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("%s: UsageError 기대, got %v", tc.name, err)
			continue
		}
		if usage.Msg != tc.want {
			t.Errorf("%s 오류 = %q, want %q", tc.name, usage.Msg, tc.want)
		}
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
		{[]string{"pr", "merge", "42"}, wantCall("gh", "pr", "merge", "42", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--merge"}, wantCall("gh", "pr", "merge", "42", "--merge", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--squash"}, wantCall("gh", "pr", "merge", "42", "--squash", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--rebase"}, wantCall("gh", "pr", "merge", "42", "--rebase", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--delete-branch"}, wantCall("gh", "pr", "merge", "42", "--delete-branch", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--auto"}, wantCall("gh", "pr", "merge", "42", "--auto", "-R", "github.com/o/r")},
		{[]string{"pr", "merge", "42", "--squash", "--delete-branch", "--auto"}, wantCall("gh", "pr", "merge", "42", "--squash", "--delete-branch", "--auto", "-R", "github.com/o/r")},
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
		{[]string{"pr", "merge", "42"}, wantCall("glab", "mr", "merge", "42", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r")},
		{[]string{"pr", "merge", "42", "--merge"}, wantCall("glab", "mr", "merge", "42", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r")},
		{[]string{"pr", "merge", "42", "--squash"}, wantCall("glab", "mr", "merge", "42", "--squash", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r")},
		{[]string{"pr", "merge", "42", "--delete-branch"}, wantCall("glab", "mr", "merge", "42", "--remove-source-branch", "--when-pipeline-succeeds=false", "--repo", "https://gitlab.com/o/r")},
		{[]string{"pr", "merge", "42", "--auto"}, wantCall("glab", "mr", "merge", "42", "--auto-merge", "--repo", "https://gitlab.com/o/r")},
		{[]string{"pr", "merge", "42", "--squash", "--delete-branch", "--auto"}, wantCall("glab", "mr", "merge", "42", "--squash", "--remove-source-branch", "--auto-merge", "--repo", "https://gitlab.com/o/r")},
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

func TestE2EPRMergeTeaArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "tea 기본 병합",
			args: []string{"pr", "merge", "42"},
			want: wantTeaCall("pulls", "merge", "42", "--login", "pub", "--repo", "o/r"),
		},
		{
			name: "tea squash 병합",
			args: []string{"pr", "merge", "42", "--squash"},
			want: wantTeaCall("pulls", "merge", "42", "--style", "squash", "--login", "pub", "--repo", "o/r"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			repo := tempRepo(t, "https://gitea.com/o/r.git")

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

func TestE2EPRMergeTeaUnsupportedFlags(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "tea 자동 병합", args: []string{"pr", "merge", "42", "--auto"}},
		{name: "tea branch 삭제", args: []string{"pr", "merge", "42", "--delete-branch"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("gg %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if !strings.Contains(stderr, "pr merge --auto/--delete-branch is not supported for tea") {
				t.Errorf("gg %v: stderr = %q", tc.args, stderr)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("tea에게 명령이 실행되면 안 된다: %q", got)
			}
		})
	}
}

func TestE2EPRMergeHelp(t *testing.T) {
	bin := buildGG(t)
	assertGGHelp(t, bin, []string{"pr", "merge", "--help"}, []string{
		"Usage:", "pr merge <number>", "--merge", "--squash", "--rebase", "--delete-branch", "--auto", "--repo", "--remote", "--help",
	})
	assertGGHelp(t, bin, []string{"--help"}, []string{"Usage:", "pr"})
}
