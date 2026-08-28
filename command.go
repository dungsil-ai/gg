package main

import (
	"strings"
)

// UsageError는 exit code 2로 이어지는 사용법 오류다.
type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }
func usageErr(m string) error      { return UsageError{Msg: m} }

// Request는 파싱된 공통 명령이다.
type Request struct {
	Resource   string // "repo" | "issue" | "pr"
	Action     string // list | view | create | clone | pull | push
	RepoFlag   string // --repo 값
	RemoteFlag string // --remote 값
	Number     string // issue/pr view 대상
	CloneURL   string
	CloneDir   string
	GitArgs    []string // pull/push는 검사 없이 git으로 전달

	Title, Body, Base, Head, State, Limit, Description string
	Draft, Public, Private, AllowInsecureHTTP          bool
}

var repoActions = map[string]bool{
	"list": true, "view": true, "create": true,
	"clone": true, "pull": true, "push": true,
}

func ParseRequest(args []string) (Request, error) {
	var req Request
	for len(args) >= 2 && (args[0] == "--repo" || args[0] == "--remote") {
		if err := setContextFlag(&req, args[0], args[1]); err != nil {
			return req, err
		}
		args = args[2:]
	}
	if len(args) == 1 && (args[0] == "--repo" || args[0] == "--remote") {
		return req, setContextFlag(&req, args[0], "")
	}
	if len(args) == 0 {
		return req, usageErr("missing command")
	}
	head, rest := args[0], args[1:]
	switch {
	case head == "issue" || head == "pr":
		req.Resource = head
		if len(rest) == 0 {
			return req, usageErr(head + " needs an action: list, view, create")
		}
		req.Action, rest = rest[0], rest[1:]
		if req.Action != "list" && req.Action != "view" && req.Action != "create" {
			return req, usageErr(head + " does not support " + req.Action)
		}
	case head == "repo":
		req.Resource = "repo"
		if len(rest) == 0 || !repoActions[rest[0]] {
			return req, usageErr("repo needs an action: list, view, create, clone, pull, push")
		}
		req.Action, rest = rest[0], rest[1:]
	case repoActions[head]: // gg list == gg repo list
		req.Resource, req.Action = "repo", head
	default:
		return req, usageErr("unknown command " + head)
	}
	if err := parseRest(&req, rest); err != nil {
		return req, err
	}
	if req.RepoFlag != "" && req.RemoteFlag != "" {
		return req, usageErr("--repo and --remote cannot be used together")
	}
	if req.RemoteFlag != "" && !supportsRemote(req) {
		return req, usageErr("--remote is not supported for " + req.Resource + " " + req.Action)
	}
	return req, nil
}

func supportsRemote(req Request) bool {
	return req.Resource != "repo" || req.Action == "list" || req.Action == "view"
}

func setContextFlag(req *Request, flag, value string) error {
	if value == "" {
		if flag == "--repo" {
			return usageErr("--repo needs a URL")
		}
		return usageErr("--remote needs a name")
	}
	if flag == "--repo" {
		req.RepoFlag = value
	} else {
		req.RemoteFlag = value
	}
	return nil
}

// flagLoop은 허용된 flag만 소비하고 positional 인자를 돌려준다.
func flagLoop(req *Request, args []string, strs map[string]*string, bools map[string]*bool) ([]string, error) {
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		if a == "--repo" || a == "--remote" {
			if i+1 >= len(args) {
				return nil, setContextFlag(req, a, "")
			}
			if err := setContextFlag(req, a, args[i+1]); err != nil {
				return nil, err
			}
			i++
			continue
		}
		if p, ok := bools[a]; ok {
			*p = true
			continue
		}
		if p, ok := strs[a]; ok {
			if i+1 >= len(args) {
				return nil, usageErr(a + " needs a value")
			}
			*p = args[i+1]
			i++
			continue
		}
		return nil, usageErr("unknown flag " + a)
	}
	return pos, nil
}

func parseRest(req *Request, args []string) error {
	switch req.Resource + " " + req.Action {
	case "repo pull", "repo push":
		req.GitArgs = args
		return nil
	case "repo clone":
		pos, err := flagLoop(req, args, nil, map[string]*bool{"--allow-insecure-http": &req.AllowInsecureHTTP})
		if err != nil {
			return err
		}
		if len(pos) < 1 || len(pos) > 2 {
			return usageErr("usage: gg clone <URL> [DIR]")
		}
		req.CloneURL = pos[0]
		if len(pos) == 2 {
			req.CloneDir = pos[1]
		}
		return nil
	case "repo list":
		return noPositional(req, args, map[string]*string{"--limit": &req.Limit}, nil)
	case "repo view":
		return noPositional(req, args, nil, nil)
	case "repo create":
		err := noPositional(req, args,
			map[string]*string{"--description": &req.Description},
			map[string]*bool{"--public": &req.Public, "--private": &req.Private})
		if err != nil {
			return err
		}
		if req.RepoFlag == "" {
			return usageErr("repo create needs --repo <new-repository-URL>")
		}
		if req.Public == req.Private {
			return usageErr("repo create needs exactly one of --public or --private")
		}
		return nil
	case "issue list", "pr list":
		if err := noPositional(req, args, map[string]*string{
			"--state": &req.State, "--limit": &req.Limit,
		}, nil); err != nil {
			return err
		}
		switch req.State {
		case "", "open", "closed", "all":
			return nil
		}
		return usageErr("--state must be open, closed, or all")
	case "issue view", "pr view":
		pos, err := flagLoop(req, args, nil, nil)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return usageErr("usage: gg " + req.Resource + " view <number>")
		}
		req.Number = pos[0]
		return nil
	case "issue create":
		return noPositional(req, args, map[string]*string{
			"--title": &req.Title, "--body": &req.Body,
		}, nil)
	case "pr create":
		return noPositional(req, args, map[string]*string{
			"--title": &req.Title, "--body": &req.Body,
			"--base": &req.Base, "--head": &req.Head,
		}, map[string]*bool{"--draft": &req.Draft})
	}
	return usageErr("unknown command")
}

func noPositional(req *Request, args []string, strs map[string]*string, bools map[string]*bool) error {
	pos, err := flagLoop(req, args, strs, bools)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return usageErr("unexpected argument " + pos[0])
	}
	return nil
}

// Invocation은 실행할 자식 process다.
type Invocation struct {
	Bin  string
	Args []string
	Env  []string // os.Environ()에 덧붙일 KEY=VALUE
}

func Translate(req Request, r RepoURL, p Provider, teaLogin string) (Invocation, error) {
	switch p {
	case GH:
		return ghInvocation(req, r)
	case GLab:
		return glabInvocation(req, r)
	case Tea:
		return teaInvocation(req, r, teaLogin)
	}
	return Invocation{}, usageErr("unknown provider " + string(p))
}

func appendKV(args []string, flag, val string) []string {
	if val == "" {
		return args
	}
	return append(args, flag, val)
}

func ghInvocation(req Request, r RepoURL) (Invocation, error) {
	inv := Invocation{Bin: "gh"}
	if r.Host != "github.com" {
		inv.Env = []string{"GH_HOST=" + r.Host}
	}
	target := []string{"-R", r.Host + "/" + r.Slug()}
	res := map[string]string{"repo": "repo", "issue": "issue", "pr": "pr"}[req.Resource]
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV([]string{"repo", "list"}, "--limit", req.Limit)
	case "repo view":
		inv.Args = []string{"repo", "view", r.HTTPS()}
		inv.Env = nil // URL이 host를 지정하므로 GH_HOST 불필요
	case "repo create":
		inv.Args = []string{"repo", "create", r.Slug(), visFlag(req)}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
	case "repo clone":
		inv.Args = []string{"repo", "clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
		inv.Env = nil
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		inv.Args = appendKV(inv.Args, "--state", req.State)
		inv.Args = appendKV(inv.Args, "--limit", req.Limit)
		inv.Env = nil
	case "issue view", "pr view":
		inv.Args = append([]string{res, "view", req.Number}, target...)
		inv.Env = nil
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--body", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--base", req.Base)
			inv.Args = appendKV(inv.Args, "--head", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
		inv.Env = nil
	default:
		return Invocation{}, usageErr(req.Resource + " does not support " + req.Action)
	}
	return inv, nil
}

func visFlag(req Request) string {
	if req.Private {
		return "--private"
	}
	return "--public"
}

func glabInvocation(req Request, r RepoURL) (Invocation, error) {
	inv := Invocation{Bin: "glab"}
	target := []string{"--repo", r.HTTPS()}
	res := map[string]string{"repo": "repo", "issue": "issue", "pr": "mr"}[req.Resource]
	stateFlags := map[string]string{"closed": "--closed", "all": "--all"}
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV([]string{"repo", "list"}, "--per-page", req.Limit)
		inv.Env = []string{"GITLAB_HOST=" + r.Host}
	case "repo view":
		inv.Args = []string{"repo", "view", r.HTTPS()}
	case "repo create":
		inv.Args = []string{"repo", "create", r.Slug(), visFlag(req)}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
		inv.Env = []string{"GITLAB_HOST=" + r.Host}
	case "repo clone":
		inv.Args = []string{"repo", "clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		if f := stateFlags[req.State]; f != "" {
			inv.Args = append(inv.Args, f)
		}
		inv.Args = appendKV(inv.Args, "--per-page", req.Limit)
	case "issue view", "pr view":
		inv.Args = append([]string{res, "view", req.Number}, target...)
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--description", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--target-branch", req.Base)
			inv.Args = appendKV(inv.Args, "--source-branch", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
	default:
		return Invocation{}, usageErr(req.Resource + " does not support " + req.Action)
	}
	return inv, nil
}

func teaInvocation(req Request, r RepoURL, login string) (Invocation, error) {
	inv := Invocation{Bin: "tea"}
	auth := []string{"--login", login}
	target := append(append([]string{}, auth...), "--repo", r.Slug())
	res := map[string]string{"repo": "repos", "issue": "issues", "pr": "pulls"}[req.Resource]
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV(append([]string{"repos", "list"}, auth...), "--limit", req.Limit)
	case "repo view":
		inv.Args = append([]string{"repos", r.Slug()}, auth...)
	case "repo create":
		inv.Args = append([]string{"repos", "create"}, auth...)
		inv.Args = append(inv.Args, "--owner", r.Owner, "--name", r.Name)
		if req.Private {
			inv.Args = append(inv.Args, "--private")
		}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
	case "repo clone":
		inv.Args = []string{"clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		inv.Args = appendKV(inv.Args, "--state", req.State)
		inv.Args = appendKV(inv.Args, "--limit", req.Limit)
	case "issue view", "pr view":
		inv.Args = append([]string{res, req.Number}, target...)
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--description", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--base", req.Base)
			inv.Args = appendKV(inv.Args, "--head", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
	default:
		return Invocation{}, usageErr(req.Resource + " does not support " + req.Action)
	}
	return inv, nil
}
