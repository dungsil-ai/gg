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
		{name: "issue comment", args: []string{"issue", "comment", "42", "--body", "hello world"},
			want: Request{Resource: "issue", Action: "comment", Number: "42", Body: "hello world"}},
		{name: "issue close", args: []string{"issue", "close", "42"},
			want: Request{Resource: "issue", Action: "close", Number: "42"}},
		{name: "issue reopen", args: []string{"issue", "reopen", "42"},
			want: Request{Resource: "issue", Action: "reopen", Number: "42"}},
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
		{name: "commit 전달", args: []string{"commit", "--allow-empty", "-m", "test"},
			want: Request{Resource: "repo", Action: "commit", GitArgs: []string{"--allow-empty", "-m", "test"}}},
		{name: "pull 전달", args: []string{"pull", "--rebase", "origin", "main"},
			want: Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase", "origin", "main"}}},
		{name: "push 전달", args: []string{"repo", "push", "--force-with-lease"},
			want: Request{Resource: "repo", Action: "push", GitArgs: []string{"--force-with-lease"}}},
		{name: "config list", args: []string{"config", "list"},
			want: Request{Resource: "config", Action: "list"}},
		{name: "config set", args: []string{"config", "set", "Git.Example.com:8443", "glab"},
			want: Request{Resource: "config", Action: "set", ConfigHost: "Git.Example.com:8443", ConfigProvider: "glab"}},
		{name: "config unset", args: []string{"config", "unset", "git.example.com"},
			want: Request{Resource: "config", Action: "unset", ConfigHost: "git.example.com"}},
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

func TestParseRequestPRReady(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "ready", args: []string{"pr", "ready", "42"},
			want: Request{Resource: "pr", Action: "ready", Number: "42"}},
		{name: "undo", args: []string{"pr", "ready", "42", "--undo"},
			want: Request{Resource: "pr", Action: "ready", Number: "42", Undo: true}},
		{name: "repo context", args: []string{"pr", "ready", "42", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "pr", Action: "ready", Number: "42", RepoFlag: "https://github.com/o/r"}},
		{name: "remote context", args: []string{"pr", "ready", "42", "--remote", "upstream"},
			want: Request{Resource: "pr", Action: "ready", Number: "42", RemoteFlag: "upstream"}},
		{name: "explain", args: []string{"pr", "ready", "42", "--explain"},
			want: Request{Resource: "pr", Action: "ready", Number: "42", Explain: true}},
		{name: "help", args: []string{"pr", "ready", "42", "--help"},
			want: Request{Resource: "pr", Action: "ready", Number: "42", Help: true}},
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

	bad := []struct {
		args []string
		want string
	}{
		{args: []string{"pr", "ready"}, want: "usage: gg pr ready <number>"},
		{args: []string{"pr", "ready", "42", "43"}, want: "usage: gg pr ready <number>"},
		{args: []string{"pr", "ready", "42", "--wat"}, want: "unknown flag --wat"},
		{args: []string{"pr", "status", "42", "--undo"}, want: "unknown flag --undo"},
	}
	for _, c := range bad {
		_, err := ParseRequest(c.args)
		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", c.args, err)
			continue
		}
		if usage.Msg != c.want {
			t.Errorf("ParseRequest(%v) = %q, want %q", c.args, usage.Msg, c.want)
		}
	}
}

func TestParseRequestCommandAliasUsesCanonicalCommand(t *testing.T) {
	tests := []struct {
		name      string
		canonical []string
		alias     []string
	}{
		{
			name:      "list with global and command flags",
			canonical: []string{"--repo", "https://github.com/o/r", "pr", "list", "--state", "all", "--limit", "3"},
			alias:     []string{"--repo", "https://github.com/o/r", "mr", "list", "--state", "all", "--limit", "3"},
		},
		{
			name:      "view with remote",
			canonical: []string{"pr", "view", "42", "--remote", "upstream"},
			alias:     []string{"mr", "view", "42", "--remote", "upstream"},
		},
		{
			name:      "status",
			canonical: []string{"pr", "status", "42", "--repo", "https://github.com/o/r"},
			alias:     []string{"mr", "status", "42", "--repo", "https://github.com/o/r"},
		},
		{
			name:      "ready",
			canonical: []string{"pr", "ready", "42", "--undo", "--repo", "https://github.com/o/r"},
			alias:     []string{"mr", "ready", "42", "--undo", "--repo", "https://github.com/o/r"},
		},
		{
			name:      "create",
			canonical: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "feature", "--draft"},
			alias:     []string{"mr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "feature", "--draft"},
		},
		{
			name:      "merge",
			canonical: []string{"pr", "merge", "42", "--squash", "--delete-branch", "--auto"},
			alias:     []string{"mr", "merge", "42", "--squash", "--delete-branch", "--auto"},
		},
		{
			name:      "explain before alias command",
			canonical: []string{"--explain", "pr", "list"},
			alias:     []string{"--explain", "mr", "list"},
		},
	}
	for _, tt := range tests {
		want, err := ParseRequest(tt.canonical)
		if err != nil {
			t.Fatalf("ParseRequest(%v): %v", tt.canonical, err)
		}
		got, err := ParseRequest(tt.alias)
		if err != nil {
			t.Fatalf("ParseRequest(%v): %v", tt.alias, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ParseRequest(%v) = %+v, want ParseRequest(%v) = %+v", tt.alias, got, tt.canonical, want)
		}
	}
}

func TestParseRequestRepositoryContextRemoteBeforeAndAfterCommand(t *testing.T) {
	want := Request{Resource: "issue", Action: "list", RemoteFlag: "upstream"}
	for _, args := range [][]string{
		{"--remote", "upstream", "issue", "list"},
		{"issue", "list", "--remote", "upstream"},
	} {
		got, err := ParseRequest(args)
		if err != nil {
			t.Fatalf("ParseRequest(%v): %v", args, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ParseRequest(%v) = %+v, want %+v", args, got, want)
		}
	}
}

func TestParseRequestRejectsRepositoryContextFlagsTogether(t *testing.T) {
	for _, args := range [][]string{
		{"--repo", "https://github.com/o/r", "issue", "list", "--remote", "upstream"},
		{"--remote", "upstream", "issue", "list", "--repo", "https://github.com/o/r"},
	} {
		_, err := ParseRequest(args)
		var usage UsageError
		if !errors.As(err, &usage) || !strings.Contains(err.Error(), "cannot be used together") {
			t.Errorf("ParseRequest(%v): conflict UsageError 기대, got %v", args, err)
		}
	}
}

func TestParseRequestRepositoryContextRemoteScope(t *testing.T) {
	allowed := [][]string{
		{"repo", "list", "--remote", "upstream"},
		{"repo", "view", "--remote", "upstream"},
		{"issue", "list", "--remote", "upstream"},
		{"issue", "view", "1", "--remote", "upstream"},
		{"issue", "create", "--remote", "upstream"},
		{"pr", "list", "--remote", "upstream"},
		{"pr", "view", "1", "--remote", "upstream"},
		{"issue", "comment", "1", "--body", "b", "--remote", "upstream"},
		{"issue", "close", "1", "--remote", "upstream"},
		{"issue", "reopen", "1", "--remote", "upstream"},
		{"pr", "create", "--remote", "upstream"},
	}
	for _, args := range allowed {
		got, err := ParseRequest(args)
		if err != nil {
			t.Errorf("ParseRequest(%v): %v", args, err)
		} else if got.RemoteFlag != "upstream" {
			t.Errorf("ParseRequest(%v).RemoteFlag = %q", args, got.RemoteFlag)
		}
	}

	for _, args := range [][]string{
		{"--remote", "upstream", "clone", "https://github.com/o/r"},
		{"--remote", "upstream", "commit"},
		{"--remote", "upstream", "pull"},
		{"--remote", "upstream", "push"},
	} {
		_, err := ParseRequest(args)
		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}

func TestParseRequestKeepsPullRemoteArgument(t *testing.T) {
	got, err := ParseRequest([]string{"pull", "--remote", "upstream"})
	if err != nil {
		t.Fatal(err)
	}
	want := Request{Resource: "repo", Action: "pull", GitArgs: []string{"--remote", "upstream"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseRequest = %+v, want %+v", got, want)
	}
}

func TestParseRequestErrors(t *testing.T) {
	bad := [][]string{
		{},                                      // 명령 없음
		{"unknown"},                             // 알 수 없는 자원
		{"issue"},                               // action 없음
		{"issue", "delete", "1"},                // 지원 안 하는 action
		{"issue", "view"},                       // number 없음
		{"issue", "view", "1", "2"},             // 인자 초과
		{"issue", "comment"},                    // number 없음
		{"issue", "comment", "1"},               // body 없음
		{"issue", "comment", "1", "--body", ""}, // 빈 body
		{"issue", "comment", "1", "--body", "   "}, // 공백 body
		{"issue", "comment", "1", "2", "--body", "b"},
		{"issue", "close"},                        // number 없음
		{"issue", "close", "1", "2"},              // 인자 초과
		{"issue", "reopen"},                       // number 없음
		{"issue", "reopen", "1", "2"},             // 인자 초과
		{"issue", "list", "--wat"},                // 알 수 없는 flag
		{"pr", "list", "--state", "merged"},       // 지원 안 하는 state
		{"pr", "create", "--title"},               // 값 없는 flag
		{"clone"},                                 // URL 없음
		{"clone", "u", "d", "x"},                  // 인자 초과
		{"create", "--public"},                    // --repo 없는 repo create
		{"create", "--repo", "https://x.com/o/r"}, // 공개 범위 없음
		{"create", "--repo", "https://x.com/o/r", "--public", "--private"}, // 둘 다 지정
		{"list", "extra"}, // list에 positional
		{"config"},
		{"config", "show"},
		{"config", "list", "extra"},
		{"config", "set", "only-host"},
		{"config", "set", "host", "gh", "extra"},
		{"config", "unset"},
		{"config", "unset", "host", "extra"},
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
		{name: "gh issue comment",
			req:  Request{Resource: "issue", Action: "comment", Number: "18", Body: "hello"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"issue", "comment", "18", "--body", "hello", "-R", "github.com/o/r"}}},
		{name: "gh issue close",
			req:  Request{Resource: "issue", Action: "close", Number: "18"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"issue", "close", "18", "-R", "github.com/o/r"}}},
		{name: "gh issue reopen",
			req:  Request{Resource: "issue", Action: "reopen", Number: "18"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"issue", "reopen", "18", "-R", "github.com/o/r"}}},
		{name: "gh pr view",
			req:  Request{Resource: "pr", Action: "view", Number: "7"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "view", "7", "-R", "github.com/o/r"}}},
		{name: "gh pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "create", "-R", "github.com/o/r", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}}},
		{name: "gh pr ready",
			req:  Request{Resource: "pr", Action: "ready", Number: "7"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "ready", "7", "-R", "github.com/o/r"}}},
		{name: "gh pr ready undo",
			req:  Request{Resource: "pr", Action: "ready", Number: "7", Undo: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "ready", "7", "--undo", "-R", "github.com/o/r"}}},
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
		{name: "glab pr ready",
			req:  Request{Resource: "pr", Action: "ready", Number: "7"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "update", "7", "--ready", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab pr ready undo",
			req:  Request{Resource: "pr", Action: "ready", Number: "7", Undo: true},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "update", "7", "--draft", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab issue view",
			req:  Request{Resource: "issue", Action: "view", Number: "9"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "view", "9", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab issue comment",
			req:  Request{Resource: "issue", Action: "comment", Number: "18", Body: "hello"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "note", "18", "--message", "hello", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab issue close",
			req:  Request{Resource: "issue", Action: "close", Number: "18"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "close", "18", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab issue reopen",
			req:  Request{Resource: "issue", Action: "reopen", Number: "18"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "reopen", "18", "--repo", "https://git.example.com/grp/sub/p"}}},
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
		{name: "tea issue comment",
			req:  Request{Resource: "issue", Action: "comment", Number: "18", Body: "hello"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"comment", "18", "hello", "--login", "corp", "--repo", "o/r"}}},
		{name: "tea issue close",
			req:  Request{Resource: "issue", Action: "close", Number: "18"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"issues", "close", "18", "--login", "corp", "--repo", "o/r"}}},
		{name: "tea issue reopen",
			req:  Request{Resource: "issue", Action: "reopen", Number: "18"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"issues", "reopen", "18", "--login", "corp", "--repo", "o/r"}}},
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

func TestTranslateTeaPRReadyUnsupported(t *testing.T) {
	_, err := Translate(
		Request{Resource: "pr", Action: "ready", Number: "7"},
		RepoURL{Host: "gitea.com", Owner: "o", Name: "r"},
		Tea,
		"",
	)
	var usage UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("Translate(pr ready, tea): UsageError 기대, got %v", err)
	}
	if usage.Msg != "pr ready is not supported for tea" {
		t.Errorf("Tea 오류 = %q", usage.Msg)
	}
}

func TestTranslateUnsupportedAction(t *testing.T) {
	req := Request{Resource: "issue", Action: "delete"}
	repo := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	for _, p := range []Provider{GH, GLab, Tea} {
		_, err := Translate(req, repo, p, "corp")
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Fatalf("Translate(%s issue delete): UsageError 기대, got %v", p, err)
		}
		if !strings.Contains(ue.Msg, "does not support delete") {
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

func TestPlanCommitDisablesSigning(t *testing.T) {
	req := Request{Resource: "repo", Action: "commit", GitArgs: []string{"--allow-empty", "-m", "test"}}
	inv, err := plan(req)
	if err != nil {
		t.Fatal(err)
	}
	want := Invocation{Bin: "git", Args: []string{"commit", "--no-gpg-sign", "--allow-empty", "-m", "test"}}
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

func TestPlanTeaReadyUnsupportedSkipsLogin(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{})

	_, err := plan(Request{
		Resource: "pr",
		Action:   "ready",
		Number:   "7",
		RepoFlag: "https://gitea.com/o/r",
	})
	var usage UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("plan(pr ready, tea): UsageError 기대, got %v", err)
	}
	if usage.Msg != "pr ready is not supported for tea" {
		t.Errorf("Tea 오류 = %q", usage.Msg)
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

func TestParseRequestExplain(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{
			name: "gg --explain issue list",
			args: []string{"--explain", "issue", "list"},
			want: Request{Resource: "issue", Action: "list", Explain: true},
		},
		{
			name: "gg issue list --explain",
			args: []string{"issue", "list", "--explain"},
			want: Request{Resource: "issue", Action: "list", Explain: true},
		},
		{
			name: "gg --explain --remote upstream issue list --limit 5",
			args: []string{"--explain", "--remote", "upstream", "issue", "list", "--limit", "5"},
			want: Request{Resource: "issue", Action: "list", RemoteFlag: "upstream", Limit: "5", Explain: true},
		},
		{
			name: "gg issue list --remote upstream --limit 5 --explain",
			args: []string{"issue", "list", "--remote", "upstream", "--limit", "5", "--explain"},
			want: Request{Resource: "issue", Action: "list", RemoteFlag: "upstream", Limit: "5", Explain: true},
		},
		{
			name: "gg --explain repo view",
			args: []string{"--explain", "repo", "view"},
			want: Request{Resource: "repo", Action: "view", Explain: true},
		},
		{
			name: "gg --explain repo create with flags",
			args: []string{"--explain", "repo", "create", "--repo", "https://github.com/o/r", "--public"},
			want: Request{Resource: "repo", Action: "create", RepoFlag: "https://github.com/o/r", Public: true, Explain: true},
		},
		{
			name: "gg --explain clone with url",
			args: []string{"--explain", "clone", "https://github.com/o/r"},
			want: Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", Explain: true},
		},
		{
			name: "gg --explain pr list",
			args: []string{"--explain", "pr", "list"},
			want: Request{Resource: "pr", Action: "list", Explain: true},
		},
		{
			name: "gg pr view 1 --explain",
			args: []string{"pr", "view", "1", "--explain"},
			want: Request{Resource: "pr", Action: "view", Number: "1", Explain: true},
		},
		{
			name: "gg --explain pr create",
			args: []string{"--explain", "pr", "create", "--title", "t", "--body", "b"},
			want: Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Explain: true},
		},
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

func TestParseRequestExplainErrors(t *testing.T) {
	bad := [][]string{
		{"--explain"},
		{"--explain", "config", "list"},
		{"config", "list", "--explain"},
		{"--explain", "pull"},
		{"--explain", "push"},
		{"--explain", "--repo", "https://github.com/o/r", "--remote", "upstream", "issue", "list"},
	}
	for _, args := range bad {
		_, err := ParseRequest(args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}

func TestExplainFormatting(t *testing.T) {
	var buf strings.Builder
	ep := executionPlan{
		repo:     RepoURL{Host: "github.com", Owner: "o", Name: "r"},
		provider: GH,
		inv:      Invocation{Bin: "gh"},
	}
	explain(&buf, ep)
	got := buf.String()
	wants := []string{
		"저장소 문맥: https://github.com/o/r\n",
		"Provider: gh\n",
		"CLI: gh\n",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("explain output missing %q:\n%s", want, got)
		}
	}
}

// TestParseRequestErrorMessages는 오류 메시지 계약을 고정한다.
func TestParseRequestErrorMessages(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"unknown"}, "unknown command unknown"},
		{[]string{"config"}, "config needs an action: list, set, unset"},
		{[]string{"issue"}, "issue needs an action: list, view, create, comment, close, reopen"},
		{[]string{"pr"}, "pr needs an action: list, view, create, status, ready, merge"},
		{[]string{"repo"}, "repo needs an action: list, view, create, clone, commit, pull, push"},
		{[]string{"issue", "delete", "1"}, "issue does not support delete"},
		{[]string{"pr", "delete", "1"}, "pr does not support delete"},
		{[]string{"issue", "list", "--wat"}, "unknown flag --wat"},
		{[]string{"issue", "view"}, "usage: gg issue view <number>"},
		{[]string{"pr", "view"}, "usage: gg pr view <number>"},
		{[]string{"issue", "close"}, "usage: gg issue close <number>"},
		{[]string{"issue", "comment", "1"}, "usage: gg issue comment <number> --body <text>"},
		{[]string{"pr", "merge"}, "usage: gg pr merge <number>"},
		{[]string{"pr", "merge", "1", "--merge", "--squash"}, "--merge, --squash, --rebase are mutually exclusive; use at most one"},
		{[]string{"clone", "https://x.com/o/r", "d", "x"}, "usage: gg clone <URL> [DIR]"},
		{[]string{"create", "--public"}, "repo create needs --repo <new-repository-URL>"},
		{[]string{"create", "--repo", "https://x.com/o/r"}, "repo create needs exactly one of --public or --private"},
		{[]string{"list", "extra"}, "unexpected argument extra"},
		{[]string{"config", "list", "extra"}, "usage: gg config list"},
		{[]string{"config", "set", "only-host"}, "usage: gg config set <host> <provider>"},
		{[]string{"config", "unset"}, "usage: gg config unset <host>"},
		{[]string{"issue", "list", "--state", "merged"}, "--state must be open, closed, or all"},
		{[]string{"pr", "create", "--title"}, "--title needs a value"},
		{[]string{"--remote", "upstream", "clone", "https://github.com/o/r"}, "--remote is not supported for repo clone"},
		{[]string{"--explain", "pull"}, "--explain is not supported for repo pull"},
		{[]string{"--explain", "config", "list"}, "--explain is not supported for config list"},
		{[]string{"--repo", "https://github.com/o/r", "--remote", "upstream", "issue", "list"}, "--repo and --remote cannot be used together"},
		{[]string{"--repo"}, "--repo needs a URL"},
		{[]string{"--remote"}, "--remote needs a name"},
	}
	for _, c := range cases {
		_, err := ParseRequest(c.args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", c.args, err)
			continue
		}
		if !strings.Contains(ue.Msg, c.want) {
			t.Errorf("ParseRequest(%v) = %q, want %q 포함", c.args, ue.Msg, c.want)
		}
	}
}

func TestParseRequestHelpFlag(t *testing.T) {
	got, err := ParseRequest([]string{"issue", "list", "--limit", "5", "--help"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Help {
		t.Errorf("issue list --help: Help = false, want true")
	}

	// git 전달 명령의 --help는 flag가 아니라 git 인자다
	for _, args := range [][]string{{"pull", "--help"}, {"commit", "--help"}, {"push", "--help"}} {
		got, err := ParseRequest(args)
		if err != nil {
			t.Fatalf("ParseRequest(%v): %v", args, err)
		}
		if got.Help {
			t.Errorf("ParseRequest(%v): Help = true, want false", args)
		}
		if !reflect.DeepEqual(got.GitArgs, []string{"--help"}) {
			t.Errorf("ParseRequest(%v): GitArgs = %v, want [--help]", args, got.GitArgs)
		}
	}
}
