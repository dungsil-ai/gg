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
	Resource string // "repo" | "issue" | "pr"
	Action   string // list | view | create | clone | pull | push
	RepoFlag string // --repo 값
	Number   string // issue/pr view 대상
	CloneURL string
	CloneDir string
	GitArgs  []string // pull/push는 검사 없이 git으로 전달

	Title, Body, Base, Head, State, Limit, Description string
	Draft, Public, Private                             bool
}

var repoActions = map[string]bool{
	"list": true, "view": true, "create": true,
	"clone": true, "pull": true, "push": true,
}

func ParseRequest(args []string) (Request, error) {
	var req Request
	for len(args) >= 2 && args[0] == "--repo" {
		req.RepoFlag = args[1]
		args = args[2:]
	}
	if len(args) == 1 && args[0] == "--repo" {
		return req, usageErr("--repo needs a URL")
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
	return req, parseRest(&req, rest)
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
		if a == "--repo" {
			if i+1 >= len(args) {
				return nil, usageErr("--repo needs a URL")
			}
			req.RepoFlag = args[i+1]
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
		pos, err := flagLoop(req, args, nil, nil)
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
