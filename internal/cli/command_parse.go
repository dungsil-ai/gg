package cli

import (
	"strings"
)

// UsageError는 exit code 2로 이어지는 사용법 오류다.
type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }
func usageErr(m string) error      { return UsageError{Msg: m} }

// Request는 파싱된 공통 명령이다.
type Request struct {
	Resource                   string // "repo" | "issue" | "label" | "pr" | "release" | "ci" | "config"
	Action                     string // resource action 또는 Git passthrough action
	RepoFlag                   string // --repo 값
	RemoteFlag                 string // --remote 값
	Number                     string // issue/pr/ci 대상 번호
	CommentID                  string // pr comment edit/delete의 댓글 ID
	CloneURL                   string
	CloneDir                   string
	GitArgs                    []string // passthrough action은 검사 없이 git으로 전달
	ConfigHost, ConfigProvider string

	Title, Body, Base, Head, State, Limit, Description       string
	Name, Color                                              string
	Tag, Notes, Ref, Pattern, Dir, Asset                     string // release: 태그, 노트, 태그 생성 기준 ref, download 필터·경로, asset 이름
	Branch                                                   string
	Files                                                    []string
	Draft, Undo, Public, Private, AllowInsecureHTTP, Explain bool
	Yes, Prerelease, CleanupTag                              bool

	Merge, Squash, Rebase bool
	DeleteBranch, Auto    bool

	Parent, Blocker, IssueType string // issue 관계 등록: 부모·blocker 번호와 issue 종류 이름
	RelatedID                  string // plan 단계에서 gh로 조회한 관계 issue의 numeric database id

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
			// 2단어 action(pr comment list 등)은 이어지는 토큰을 action에 합친다.
			if len(rest) > 0 && rd.action(req.Action+" "+rest[0]) != nil {
				req.Action += " " + rest[0]
				rest = rest[1:]
			}
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
