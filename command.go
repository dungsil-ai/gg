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
	Resource                   string // "repo" | "issue" | "pr" | "config"
	Action                     string // resource action 또는 Git passthrough action
	RepoFlag                   string // --repo 값
	RemoteFlag                 string // --remote 값
	Number                     string // issue/pr 대상 번호
	CloneURL                   string
	CloneDir                   string
	GitArgs                    []string // passthrough action은 검사 없이 git으로 전달
	ConfigHost, ConfigProvider string

	Title, Body, Base, Head, State, Limit, Description       string
	Draft, Undo, Public, Private, AllowInsecureHTTP, Explain bool

	Merge, Squash, Rebase bool
	DeleteBranch, Auto    bool

	Help bool // action 단계의 --help 출력 요청 여부
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
	head, aliasAction := resolveAlias(args[0])
	rest := args[1:]
	var ad *actionDef
	if rd, ok := commandDefs[head]; ok {
		req.Resource = head
		if aliasAction != "" {
			req.Action = aliasAction
		} else {
			if len(rest) == 0 {
				return req, usageErr(needsAction(head, rd))
			}
			req.Action, rest = rest[0], rest[1:]
		}
		if ad = rd.action(req.Action); ad == nil {
			return req, usageErr(head + " does not support " + req.Action)
		}
	} else {
		return req, usageErr("unknown command " + head)
	}
	// passthrough는 gg 저장소 문맥을 사용하지 않는다. action 뒤의 --repo와
	// --remote는 parseRest가 raw Git 인자로 보존하지만, 명령 앞의 --repo는 거부한다.
	if ad.passthrough && req.RepoFlag != "" {
		return req, usageErr("--repo is not supported for " + req.Resource + " " + req.Action)
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
		if len(pos) == 0 {
			return usageErr(req.Resource + " " + req.Action + " needs an argument")
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

func visFlag(req Request) string {
	if req.Private {
		return "--private"
	}
	return "--public"
}

// invocationContext는 provider별 arg-builder가 쓸 값을 담는다. res/target/auth는
// provider마다 다르게 계산되며, 각 builder는 자신에게 필요한 필드만 읽는다.
type invocationContext struct {
	req    Request
	r      RepoURL
	res    string   // 이 provider에서의 resource 이름(예: gh/tea "pr", glab "mr")
	target []string // issue/pr 계열에서 저장소를 가리키는 인자
	auth   []string // tea 전용 --login 인자; gh/glab은 nil
}

// invocationBuilder는 하나의 provider가 "resource action" 하나를 Args/Env로
// 옮긴다. env가 nil이면 추가 환경변수가 없다는 뜻이다.
type invocationBuilder func(c invocationContext) (args, env []string)

// providerBuilders는 "resource action" 하나에 대한 gh/glab/tea builder를 모은다.
// 필드가 nil인 provider는 그 action을 지원하지 않는다.
type providerBuilders struct {
	gh, glab, tea invocationBuilder
}

var listBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		args = appendKV(args, "--state", c.req.State)
		args = appendKV(args, "--limit", c.req.Limit)
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		switch c.req.State {
		case "closed":
			args = append(args, "--closed")
		case "all":
			args = append(args, "--all")
		}
		return appendKV(args, "--per-page", c.req.Limit), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		args = appendKV(args, "--state", c.req.State)
		args = appendKV(args, "--limit", c.req.Limit)
		return args, nil
	},
}

var viewBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "view", c.req.Number}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "view", c.req.Number}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, c.req.Number}, c.target...), nil
	},
}

var closeReopenBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{"issue", c.req.Action, c.req.Number}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{"issue", c.req.Action, c.req.Number}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{"issues", c.req.Action, c.req.Number}, c.target...), nil
	},
}

var createBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--base", c.req.Base)
			args = appendKV(args, "--head", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--target-branch", c.req.Base)
			args = appendKV(args, "--source-branch", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--base", c.req.Base)
			args = appendKV(args, "--head", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
}

// invocationTable은 "<resource> <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// tea의 pr merge/status/ready는 teaInvocation의 사전 가드에서 걸러지므로 여기에는
// 등록하지 않는다 — provider별 예외는 감추지 않고 그 함수에 명시적으로 남긴다.
var invocationTable = map[string]providerBuilders{
	"repo list": {
		gh: func(c invocationContext) (args, env []string) {
			args = appendKV([]string{"repo", "list"}, "--limit", c.req.Limit)
			if c.r.Host != "github.com" {
				env = []string{"GH_HOST=" + c.r.Host}
			}
			return args, env
		},
		glab: func(c invocationContext) (args, env []string) {
			return appendKV([]string{"repo", "list"}, "--per-page", c.req.Limit), []string{"GITLAB_HOST=" + c.r.Host}
		},
		tea: func(c invocationContext) (args, env []string) {
			return appendKV(append([]string{"repos", "list"}, c.auth...), "--limit", c.req.Limit), nil
		},
	},
	"repo view": {
		gh: func(c invocationContext) (args, env []string) {
			return []string{"repo", "view", c.r.HTTPS()}, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return []string{"repo", "view", c.r.HTTPS()}, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{"repos", c.r.Slug()}, c.auth...), nil
		},
	},
	"repo create": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "create", c.r.Slug(), visFlag(c.req)}
			args = appendKV(args, "--description", c.req.Description)
			if c.r.Host != "github.com" {
				env = []string{"GH_HOST=" + c.r.Host}
			}
			return args, env
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "create", c.r.Slug(), visFlag(c.req)}
			args = appendKV(args, "--description", c.req.Description)
			return args, []string{"GITLAB_HOST=" + c.r.Host}
		},
		tea: func(c invocationContext) (args, env []string) {
			args = append([]string{"repos", "create"}, c.auth...)
			args = append(args, "--owner", c.r.Owner, "--name", c.r.Name)
			if c.req.Private {
				args = append(args, "--private")
			}
			return appendKV(args, "--description", c.req.Description), nil
		},
	},
	"repo clone": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			args = []string{"clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
	},
	"issue list": listBuilders,
	"pr list":    listBuilders,
	"issue view": viewBuilders,
	"pr view":    viewBuilders,
	"pr status": {
		gh: func(c invocationContext) (args, env []string) {
			return []string{"pr", "view", c.req.Number, "-R", c.r.Host + "/" + c.r.Slug(), "--json", ghStatusFields()}, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "view", c.req.Number, "--output", "json"}, c.target...), nil
		},
	},
	"pr ready": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"pr", "ready", c.req.Number}
			if c.req.Undo {
				args = append(args, "--undo")
			}
			return append(args, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"mr", "update", c.req.Number}
			if c.req.Undo {
				args = append(args, "--draft")
			} else {
				args = append(args, "--ready")
			}
			return append(args, c.target...), nil
		},
	},
	"issue comment": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"issue", "comment", c.req.Number, "--body", c.req.Body}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{"issue", "note", c.req.Number, "--message", c.req.Body}, c.target...), nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{"comment", c.req.Number, c.req.Body}, c.target...), nil
		},
	},
	"issue close":  closeReopenBuilders,
	"issue reopen": closeReopenBuilders,
	"pr merge": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"pr", "merge", c.req.Number}
			if c.req.Merge {
				args = append(args, "--merge")
			}
			if c.req.Squash {
				args = append(args, "--squash")
			}
			if c.req.Rebase {
				args = append(args, "--rebase")
			}
			if c.req.DeleteBranch {
				args = append(args, "--delete-branch")
			}
			if c.req.Auto {
				args = append(args, "--auto")
			}
			return append(args, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"mr", "merge", c.req.Number}
			if c.req.Squash {
				args = append(args, "--squash")
			}
			if c.req.DeleteBranch {
				args = append(args, "--remove-source-branch")
			}
			if c.req.Auto {
				args = append(args, "--auto-merge")
			} else {
				// pipeline 성공 대기 자동 병합을 명시적으로 끈다
				args = append(args, "--when-pipeline-succeeds=false")
			}
			return append(args, c.target...), nil
		},
	},
	"issue create": createBuilders,
	"pr create":    createBuilders,
}

// dispatch는 provider 이름으로 invocationTable을 조회해 Invocation을 만든다.
// 대상 action이 없거나 이 provider용 builder가 없으면 3개 provider가 공유하는
// "does not support" 오류를 낸다.
func dispatch(provider string, c invocationContext) (Invocation, error) {
	entry := invocationTable[c.req.Resource+" "+c.req.Action]
	var build invocationBuilder
	switch provider {
	case "gh":
		build = entry.gh
	case "glab":
		build = entry.glab
	case "tea":
		build = entry.tea
	}
	if build == nil {
		return Invocation{}, usageErr(c.req.Resource + " does not support " + c.req.Action)
	}
	args, env := build(c)
	return Invocation{Bin: provider, Args: args, Env: env}, nil
}

func ghInvocation(req Request, r RepoURL) (Invocation, error) {
	c := invocationContext{
		req:    req,
		r:      r,
		res:    req.Resource,
		target: []string{"-R", r.Host + "/" + r.Slug()},
	}
	return dispatch("gh", c)
}

func glabInvocation(req Request, r RepoURL) (Invocation, error) {
	res := req.Resource
	if res == "pr" {
		res = "mr"
	}
	c := invocationContext{
		req:    req,
		r:      r,
		res:    res,
		target: []string{"--repo", r.HTTPS()},
	}
	return dispatch("glab", c)
}

func teaInvocation(req Request, r RepoURL, login string) (Invocation, error) {
	if req.Resource == "pr" && req.Action == "merge" {
		return Invocation{}, errors.New("pr merge is not supported for tea")
	}
	if req.Resource == "pr" && (req.Action == "status" || req.Action == "ready") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
	var res string
	switch req.Resource {
	case "repo":
		res = "repos"
	case "issue":
		res = "issues"
	case "pr":
		res = "pulls"
	}
	auth := []string{"--login", login}
	c := invocationContext{
		req:    req,
		r:      r,
		res:    res,
		target: append(append([]string{}, auth...), "--repo", r.Slug()),
		auth:   auth,
	}
	return dispatch("tea", c)
}
