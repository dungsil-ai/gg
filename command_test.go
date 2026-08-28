package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseRequest(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "repo 생략 list", args: []string{"list", "--limit", "5"},
			want: Request{Resource: "repo", Action: "list", Limit: "5"}},
		{name: "repo 명시 list", args: []string{"repo", "list"},
			want: Request{Resource: "repo", Action: "list"}},
		{name: "전역 repo flag", args: []string{"--repo", "https://github.com/o/r", "view"},
			want: Request{Resource: "repo", Action: "view", RepoFlag: "https://github.com/o/r"}},
		{name: "후행 repo flag", args: []string{"issue", "list", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "issue", Action: "list", RepoFlag: "https://github.com/o/r"}},
		{name: "issue view", args: []string{"issue", "view", "42"},
			want: Request{Resource: "issue", Action: "view", Number: "42"}},
		{name: "issue create", args: []string{"issue", "create", "--title", "t", "--body", "b"},
			want: Request{Resource: "issue", Action: "create", Title: "t", Body: "b"}},
		{name: "pr list state", args: []string{"pr", "list", "--state", "all", "--limit", "3"},
			want: Request{Resource: "pr", Action: "list", State: "all", Limit: "3"}},
		{name: "pr create full", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"},
			want: Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true}},
		{name: "repo create", args: []string{"--repo", "https://gitea.com/o/r", "create", "--private", "--description", "d"},
			want: Request{Resource: "repo", Action: "create", RepoFlag: "https://gitea.com/o/r", Private: true, Description: "d"}},
		{name: "clone dir", args: []string{"clone", "https://github.com/o/r", "dst"},
			want: Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", CloneDir: "dst"}},
		{name: "clone allow insecure http", args: []string{"clone", "http://git.example.com/o/r", "--allow-insecure-http"},
			want: Request{Resource: "repo", Action: "clone", CloneURL: "http://git.example.com/o/r", AllowInsecureHTTP: true}},
		{name: "pull 전달", args: []string{"pull", "--rebase", "origin", "main"},
			want: Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase", "origin", "main"}}},
		{name: "push 전달", args: []string{"repo", "push", "--force-with-lease"},
			want: Request{Resource: "repo", Action: "push", GitArgs: []string{"--force-with-lease"}}},
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

func TestParseRequestErrors(t *testing.T) {
	bad := [][]string{
		{},                                   // 명령 없음
		{"unknown"},                          // 알 수 없는 자원
		{"issue"},                            // action 없음
		{"issue", "close", "1"},              // 지원 안 하는 action
		{"issue", "view"},                    // number 없음
		{"issue", "view", "1", "2"},          // 인자 초과
		{"issue", "list", "--wat"},           // 알 수 없는 flag
		{"pr", "list", "--state", "merged"},  // 지원 안 하는 state
		{"pr", "create", "--title"},          // 값 없는 flag
		{"clone"},                            // URL 없음
		{"clone", "u", "d", "x"},             // 인자 초과
		{"create", "--public"},               // --repo 없는 repo create
		{"create", "--repo", "https://x.com/o/r"},                 // 공개 범위 없음
		{"create", "--repo", "https://x.com/o/r", "--public", "--private"}, // 둘 다 지정
		{"list", "extra"},                    // list에 positional
	}
	for _, args := range bad {
		_, err := ParseRequest(args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}

func TestTranslate(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	ghe := RepoURL{Host: "ghe.corp.com", Owner: "o", Name: "r"}
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}
	te := RepoURL{Host: "gitea.example.com", Owner: "o", Name: "r"}

	cases := []struct {
		name string
		req  Request
		repo RepoURL
		p    Provider
		tea  string
		want Invocation
	}{
		// ---- GitHub ----
		{name: "gh issue list",
			req:  Request{Resource: "issue", Action: "list", State: "all", Limit: "5"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"issue", "list", "-R", "github.com/o/r", "--state", "all", "--limit", "5"}}},
		{name: "gh pr view",
			req:  Request{Resource: "pr", Action: "view", Number: "7"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "view", "7", "-R", "github.com/o/r"}}},
		{name: "gh pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "create", "-R", "github.com/o/r", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}}},
		{name: "gh repo list on GHE",
			req:  Request{Resource: "repo", Action: "list", Limit: "3"},
			repo: ghe, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "list", "--limit", "3"}, Env: []string{"GH_HOST=ghe.corp.com"}}},
		{name: "gh repo view",
			req:  Request{Resource: "repo", Action: "view"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "view", "https://github.com/o/r"}}},
		{name: "gh repo create",
			req:  Request{Resource: "repo", Action: "create", Public: true, Description: "d"},
			repo: ghe, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "create", "o/r", "--public", "--description", "d"}, Env: []string{"GH_HOST=ghe.corp.com"}}},
		{name: "gh clone",
			req:  Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", CloneDir: "dst"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "clone", "https://github.com/o/r", "dst"}}},
		{name: "gh clone keeps ssh non-standard port",
			req:  Request{Resource: "repo", Action: "clone", CloneURL: "ssh://git@github.com:2222/o/r.git"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "clone", "ssh://git@github.com:2222/o/r.git"}}},

		// ---- GitLab ----
		{name: "glab issue list closed",
			req:  Request{Resource: "issue", Action: "list", State: "closed", Limit: "5"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "list", "--repo", "https://git.example.com/grp/sub/p", "--closed", "--per-page", "5"}}},
		{name: "glab pr list all",
			req:  Request{Resource: "pr", Action: "list", State: "all"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "list", "--repo", "https://git.example.com/grp/sub/p", "--all"}}},
		{name: "glab pr list open은 flag 없음",
			req:  Request{Resource: "pr", Action: "list", State: "open"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "list", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "create", "--repo", "https://git.example.com/grp/sub/p", "--title", "t", "--description", "b", "--target-branch", "main", "--source-branch", "f", "--draft"}}},
		{name: "glab issue view",
			req:  Request{Resource: "issue", Action: "view", Number: "9"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "view", "9", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab repo list",
			req:  Request{Resource: "repo", Action: "list", Limit: "7"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "list", "--per-page", "7"}, Env: []string{"GITLAB_HOST=git.example.com"}}},
		{name: "glab repo create",
			req:  Request{Resource: "repo", Action: "create", Private: true, Description: "d"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "create", "grp/sub/p", "--private", "--description", "d"}, Env: []string{"GITLAB_HOST=git.example.com"}}},

		// ---- Gitea ----
		{name: "tea issue list",
			req:  Request{Resource: "issue", Action: "list", State: "all", Limit: "5"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"issues", "list", "--login", "corp", "--repo", "o/r", "--state", "all", "--limit", "5"}}},
		{name: "tea pr view",
			req:  Request{Resource: "pr", Action: "view", Number: "3"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"pulls", "3", "--login", "corp", "--repo", "o/r"}}},
		{name: "tea pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"pulls", "create", "--login", "corp", "--repo", "o/r", "--title", "t", "--description", "b", "--base", "main", "--head", "f", "--draft"}}},
		{name: "tea repo view",
			req:  Request{Resource: "repo", Action: "view"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "o/r", "--login", "corp"}}},
		{name: "tea repo create public은 --private 없음",
			req:  Request{Resource: "repo", Action: "create", Public: true, Description: "d"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--description", "d"}}},
		{name: "tea repo create private",
			req:  Request{Resource: "repo", Action: "create", Private: true},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--private"}}},
		{name: "tea clone은 login 불필요",
			req:  Request{Resource: "repo", Action: "clone", CloneURL: "https://gitea.example.com/o/r"},
			repo: te, p: Tea,
			want: Invocation{Bin: "tea", Args: []string{"clone", "https://gitea.example.com/o/r"}}},
	}
	for _, c := range cases {
		got, err := Translate(c.req, c.repo, c.p, c.tea)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestTranslateUnsupportedAction(t *testing.T) {
	req := Request{Resource: "issue", Action: "close"}
	repo := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	for _, p := range []Provider{GH, GLab, Tea} {
		_, err := Translate(req, repo, p, "corp")
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Fatalf("Translate(%s issue close): UsageError 기대, got %v", p, err)
		}
		if !strings.Contains(ue.Msg, "does not support close") {
			t.Errorf("provider %s error = %q, want unsupported action message", p, ue.Msg)
		}
	}
}

func TestPlanPullGoesToGit(t *testing.T) {
	req := Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase"}}
	inv, err := plan(req)
	if err != nil {
		t.Fatal(err)
	}
	want := Invocation{Bin: "git", Args: []string{"pull", "--rebase"}}
	if !reflect.DeepEqual(inv, want) {
		t.Errorf("plan = %+v, want %+v", inv, want)
	}
}

func TestPlanUsesRemoteWhenNoRepoFlag(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD": "main",
		"git remote get-url origin":       "git@github.com:o/r.git",
	})
	inv, err := plan(Request{Resource: "issue", Action: "list"})
	if err != nil {
		t.Fatal(err)
	}
	want := Invocation{Bin: "gh", Args: []string{"issue", "list", "-R", "github.com/o/r"}}
	if !reflect.DeepEqual(inv, want) {
		t.Errorf("plan = %+v, want %+v", inv, want)
	}
}

func TestPlanTeaNeedsLogin(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"tea logins list --output json": `[]`,
	})
	_, err := plan(Request{Resource: "issue", Action: "list",
		RepoFlag: "https://gitea.com/o/r"})
	if err == nil || !strings.Contains(err.Error(), "tea login add") {
		t.Errorf("tea login 안내 기대, got %v", err)
	}
}

func TestPlanCloneRejectsHTTPByDefault(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	_, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "http://github.com/o/r"})
	if err == nil || !strings.Contains(err.Error(), "HTTP clone is blocked by default") {
		t.Fatalf("HTTP 차단 오류 기대, got %v", err)
	}
}

func TestRunExitCodes(t *testing.T) {
	if code := run([]string{"unknown"}); code != 2 {
		t.Errorf("usage error = %d, want 2", code)
	}
	fakeExec(t, map[string]string{}) // git 실패
	if code := run([]string{"view"}); code != 1 {
		t.Errorf("route error = %d, want 1", code)
	}
}

func TestExecChildExit127(t *testing.T) {
	origLook := lookPath
	t.Cleanup(func() { lookPath = origLook })
	lookPath = func(bin string) (string, error) {
		return "", fmt.Errorf("%s not found", bin)
	}
	if code := execChild(Invocation{Bin: "gh"}); code != 127 {
		t.Errorf("execChild = %d, want 127", code)
	}
}
