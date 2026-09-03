package main

import (
	"errors"
	"strings"
)

// UsageError는 exit code 2로 이어지는 사용법 오류다.
type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }
func usageErr(m string) error      { return UsageError{Msg: m} }

// Request는 파싱된 공통 명령이다.
type Request struct {
	Resource                   string // "repo" | "issue" | "pr"
	Action                     string // list | view | create | comment | close | reopen | clone | pull | push | status | ready | merge
	RepoFlag                   string // --repo 값
	RemoteFlag                 string // --remote 값
	Number                     string // issue/pr 대상 번호
	CloneURL                   string
	CloneDir                   string
	GitArgs                    []string // commit/pull/push는 검사 없이 git으로 전달
	ConfigHost, ConfigProvider string

	Title, Body, Base, Head, State, Limit, Description       string
	Draft, Undo, Public, Private, AllowInsecureHTTP, Explain bool

	Merge, Squash, Rebase bool
	DeleteBranch, Auto    bool

	Help bool // --help: 파싱된 명령의 help를 출력한다
}

func ParseRequest(args []string) (Request, error) {
	var req Request
globalFlags:
	for len(args) > 0 {
		switch args[0] {
		case "--explain":
			req.Explain = true
			args = args[1:]
		case "--repo", "--remote":
			if len(args) < 2 {
				return req, setContextFlag(&req, args[0], "")
			}
			if err := setContextFlag(&req, args[0], args[1]); err != nil {
				return req, err
			}
			args = args[2:]
		default:
			break globalFlags
		}
	}
	if len(args) == 0 {
		return req, usageErr("missing command")
	}
	head, rest := args[0], args[1:]
	var ad *actionDef
	if rd, ok := commandDefs[head]; ok {
		req.Resource = head
		if len(rest) == 0 {
			return req, usageErr(needsAction(head, rd))
		}
		req.Action, rest = rest[0], rest[1:]
		if ad = rd.action(req.Action); ad == nil {
			return req, usageErr(head + " does not support " + req.Action)
		}
	} else if repoAliases[head] { // gg list == gg repo list
		req.Resource, req.Action = "repo", head
		ad = commandDefs["repo"].action(head)
	} else {
		return req, usageErr("unknown command " + head)
	}
	if err := parseRest(&req, ad, rest); err != nil {
		return req, err
	}
	if req.RepoFlag != "" && req.RemoteFlag != "" {
		return req, usageErr("--repo and --remote cannot be used together")
	}
	if req.RemoteFlag != "" && !ad.remoteOK {
		return req, usageErr("--remote is not supported for " + req.Resource + " " + req.Action)
	}
	if req.Explain && !ad.explainOK {
		return req, usageErr("--explain is not supported for " + req.Resource + " " + req.Action)
	}
	return req, nil
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
		if a == "--explain" {
			req.Explain = true
			continue
		}
		if a == "--help" {
			req.Help = true
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

// parseRest는 action 정의대로 나머지 인자를 파싱한다.
func parseRest(req *Request, ad *actionDef, args []string) error {
	if ad.passthrough {
		req.GitArgs = args
		return nil
	}
	strs, bools := actionFlagMaps(ad, req)
	pos, err := flagLoop(req, args, strs, bools)
	if err != nil {
		return err
	}
	if len(pos) < ad.minPos || (ad.maxPos >= 0 && len(pos) > ad.maxPos) {
		if ad.posErr != "" {
			return usageErr(ad.posErr)
		}
		return usageErr("unexpected argument " + pos[0])
	}
	if ad.setPos != nil {
		return ad.setPos(req, pos)
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
	case "pr status":
		inv.Args = []string{"pr", "view", req.Number, "-R", r.Host + "/" + r.Slug(), "--json", ghStatusFields()}
		inv.Env = nil
	case "pr ready":
		inv.Args = []string{"pr", "ready", req.Number}
		if req.Undo {
			inv.Args = append(inv.Args, "--undo")
		}
		inv.Args = append(inv.Args, target...)
		inv.Env = nil
	case "issue comment":
		inv.Args = []string{"issue", "comment", req.Number, "--body", req.Body}
		inv.Args = append(inv.Args, target...)
		inv.Env = nil
	case "issue close", "issue reopen":
		inv.Args = append([]string{"issue", req.Action, req.Number}, target...)
		inv.Env = nil
	case "pr merge":
		inv.Args = []string{"pr", "merge", req.Number}
		if req.Merge {
			inv.Args = append(inv.Args, "--merge")
		}
		if req.Squash {
			inv.Args = append(inv.Args, "--squash")
		}
		if req.Rebase {
			inv.Args = append(inv.Args, "--rebase")
		}
		if req.DeleteBranch {
			inv.Args = append(inv.Args, "--delete-branch")
		}
		if req.Auto {
			inv.Args = append(inv.Args, "--auto")
		}
		inv.Args = append(inv.Args, target...)
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
	case "pr status":
		inv.Args = append([]string{res, "view", req.Number, "--output", "json"}, target...)
	case "pr ready":
		inv.Args = []string{"mr", "update", req.Number}
		if req.Undo {
			inv.Args = append(inv.Args, "--draft")
		} else {
			inv.Args = append(inv.Args, "--ready")
		}
		inv.Args = append(inv.Args, target...)
	case "issue comment":
		inv.Args = []string{"issue", "note", req.Number, "--message", req.Body}
		inv.Args = append(inv.Args, target...)
	case "issue close", "issue reopen":
		inv.Args = append([]string{"issue", req.Action, req.Number}, target...)
	case "pr merge":
		inv.Args = []string{"mr", "merge", req.Number}
		if req.Squash {
			inv.Args = append(inv.Args, "--squash")
		}
		if req.DeleteBranch {
			inv.Args = append(inv.Args, "--remove-source-branch")
		}
		if req.Auto {
			inv.Args = append(inv.Args, "--auto-merge")
		} else {
			// pipeline 성공 대기 자동 병합을 명시적으로 끈다
			inv.Args = append(inv.Args, "--when-pipeline-succeeds=false")
		}
		inv.Args = append(inv.Args, target...)
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
	if req.Resource == "pr" && req.Action == "merge" {
		return Invocation{}, errors.New("pr merge is not supported for tea")
	}
	if req.Resource == "pr" && (req.Action == "status" || req.Action == "ready") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
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
	case "issue comment":
		inv.Args = append([]string{"comment", req.Number, req.Body}, target...)
	case "issue close", "issue reopen":
		inv.Args = append([]string{"issues", req.Action, req.Number}, target...)
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
