package main

import (
	"strings"
)

// 이 파일은 gg가 지원하는 명령, action, flag의 유일한 정의다.
// ParseRequest는 이 정의로 인자를 검사하고, help 렌더링도 같은 정의를 쓴다.

// flagDef는 action이 받는 flag 하나다.
type flagDef struct {
	name string                 // "--limit"
	arg  string                 // 값 placeholder(예: "<N>"). 빈 문자열이면 boolean flag다.
	desc string                 // help에 표시할 설명
	str  func(*Request) *string // 값이 저장될 Request 필드
	bin  func(*Request) *bool   // 켤 boolean Request 필드
}

// actionDef는 resource가 지원하는 action 하나다.
type actionDef struct {
	name    string
	summary string // resource help Commands 목록에 쓰는 한 줄 설명
	usage   string // action help의 Usage 줄
	flags   []flagDef

	showRepo    bool // help에 --repo 표시
	showRemote  bool // help에 --remote 표시
	showExplain bool // help에 --explain 표시
	remoteOK    bool // parser가 --remote를 허용
	explainOK   bool // parser가 --explain을 허용

	passthrough    bool                                   // flag 검사 없이 모든 인자를 git으로 전달
	minPos, maxPos int                                    // positional 개수 범위. maxPos가 -1이면 제한 없음
	posErr         string                                 // positional 개수 위반 오류. 비었으면 unexpected argument 오류
	setPos         func(req *Request, pos []string) error // positional 저장과 추가 검증
}

// resourceDef는 최상위 명령 하나다.
type resourceDef struct {
	name    string
	summary string // 최상위 help Commands 목록 설명
	desc    string // resource help 첫 줄
	usage   string // resource help Usage 줄
	actions []actionDef
}

func (rd *resourceDef) action(name string) *actionDef {
	for i := range rd.actions {
		if rd.actions[i].name == name {
			return &rd.actions[i]
		}
	}
	return nil
}

var (
	limitFlag = flagDef{name: "--limit", arg: "<N>", desc: "Limit the result count",
		str: func(r *Request) *string { return &r.Limit }}
	stateFlag = flagDef{name: "--state", arg: "<open|closed|all>", desc: "Filter by state",
		str: func(r *Request) *string { return &r.State }}
	titleFlag = flagDef{name: "--title", arg: "<text>", desc: "Set the title",
		str: func(r *Request) *string { return &r.Title }}
	bodyFlag = flagDef{name: "--body", arg: "<text>", desc: "Set the body",
		str: func(r *Request) *string { return &r.Body }}
	baseFlag = flagDef{name: "--base", arg: "<branch>", desc: "Set the base branch",
		str: func(r *Request) *string { return &r.Base }}
	headFlag = flagDef{name: "--head", arg: "<branch>", desc: "Set the head branch",
		str: func(r *Request) *string { return &r.Head }}
	draftFlag = flagDef{name: "--draft", desc: "Create a draft pull request",
		bin: func(r *Request) *bool { return &r.Draft }}
	undoFlag = flagDef{name: "--undo", desc: "Convert the pull request to a draft",
		bin: func(r *Request) *bool { return &r.Undo }}
	descriptionFlag = flagDef{name: "--description", arg: "<text>", desc: "Set the description",
		str: func(r *Request) *string { return &r.Description }}
	publicFlag = flagDef{name: "--public", desc: "Create a public repository",
		bin: func(r *Request) *bool { return &r.Public }}
	privateFlag = flagDef{name: "--private", desc: "Create a private repository",
		bin: func(r *Request) *bool { return &r.Private }}
	allowInsecureHTTPFlag = flagDef{name: "--allow-insecure-http", desc: "Allow insecure HTTP clone",
		bin: func(r *Request) *bool { return &r.AllowInsecureHTTP }}
	mergeFlag = flagDef{name: "--merge", desc: "Merge the pull request",
		bin: func(r *Request) *bool { return &r.Merge }}
	squashFlag = flagDef{name: "--squash", desc: "Squash and merge the pull request",
		bin: func(r *Request) *bool { return &r.Squash }}
	rebaseFlag = flagDef{name: "--rebase", desc: "Rebase and merge the pull request",
		bin: func(r *Request) *bool { return &r.Rebase }}
	deleteBranchFlag = flagDef{name: "--delete-branch", desc: "Delete the source branch after merging",
		bin: func(r *Request) *bool { return &r.DeleteBranch }}
	autoMergeFlag = flagDef{name: "--auto", desc: "Enable auto-merge after required approvals and CI pass",
		bin: func(r *Request) *bool { return &r.Auto }}
)

// 저장소 문맥과 설명 모드 flag. 파싱은 전역/flagLoop의 공통 분기가 하고,
// 정의는 help에 표시할 범위를 밝히는 데 쓴다.
var (
	repoContextFlag   = flagDef{name: "--repo", arg: "<URL>", desc: "이 URL을 저장소 문맥으로 사용"}
	remoteContextFlag = flagDef{name: "--remote", arg: "<name>", desc: "이 Git remote를 저장소 문맥으로 사용"}
	explainFlag       = flagDef{name: "--explain", desc: "선택한 저장소 문맥, Provider, 실행할 CLI를 설명"}
	helpFlag          = flagDef{name: "--help", desc: "Show help"}
)

// repo create의 --repo는 저장소 문맥이 아니라 만들 저장소 URL이다.
// 값 파싱은 flagLoop의 공통 --repo 분기가 req.RepoFlag에 채우므로, 이 flagDef는
// str/bin 세터가 없다 — help 텍스트 전용이다.
var createRepoFlag = flagDef{name: "--repo", arg: "<URL>", desc: "이 URL에 새 저장소를 만든다"}

func setState(req *Request, pos []string) error {
	switch req.State {
	case "", "open", "closed", "all":
		return nil
	}
	return usageErr("--state must be open, closed, or all")
}

func setNumber(req *Request, pos []string) error {
	req.Number = pos[0]
	return nil
}

// gitPassthroughActionNames는 forge 라우팅 없이 Git에 직접 전달할 지원 명령이다.
// Main Porcelain 37개와 ancillary 14개를 하나의 registry에서 관리한다.
// clone, commit, pull, push는 기존의 별도 동작을 유지한다.
var gitPassthroughActionNames = []string{
	"add", "am", "archive", "bisect", "branch", "bundle", "checkout", "cherry-pick",
	"citool", "clean", "describe", "diff", "fetch", "format-patch", "gc", "grep", "gui",
	"init", "log", "merge", "mv", "notes", "range-diff", "rebase", "reset", "restore",
	"revert", "rm", "shortlog", "show", "sparse-checkout", "stash", "status", "submodule",
	"switch", "tag", "worktree",
	"annotate", "blame", "bugreport", "count-objects", "diagnose", "difftool", "fsck",
	"instaweb", "maintenance", "merge-tree", "mergetool", "prune-packed", "rerere", "scalar",
}

func isGitPassthroughAction(name string) bool {
	for _, action := range gitPassthroughActionNames {
		if action == name {
			return true
		}
	}
	return false
}

func gitPassthroughActions() []actionDef {
	actions := make([]actionDef, len(gitPassthroughActionNames))
	for i, name := range gitPassthroughActionNames {
		actions[i] = actionDef{
			name: name, summary: "Run git " + name, usage: "gg " + name + " [git args]",
			passthrough: true, maxPos: -1,
		}
	}
	return actions
}

var commandDefs = map[string]*resourceDef{
	"repo": {
		name:    "repo",
		summary: "List, view, create, or clone repositories, or run supported Git commands",
		desc:    "List, view, create, or clone repositories, or run supported Git commands.",
		usage:   "gg repo <command> [args]",
		actions: append([]actionDef{
			{
				name: "list", summary: "List repositories", usage: "gg repo list [flags]",
				flags:    []flagDef{limitFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
			},
			{
				name: "view", summary: "View one repository", usage: "gg repo view [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
			},
			{
				name: "create", summary: "Create a repository", usage: "gg repo create [flags]",
				flags:       []flagDef{createRepoFlag, descriptionFlag, publicFlag, privateFlag},
				showExplain: true, explainOK: true,
				setPos: func(req *Request, pos []string) error {
					if req.RepoFlag == "" {
						return usageErr("repo create needs --repo <new-repository-URL>")
					}
					if req.Public == req.Private {
						return usageErr("repo create needs exactly one of --public or --private")
					}
					return nil
				},
			},
			{
				name: "clone", summary: "Clone a repository", usage: "gg repo clone <URL> [DIR] [flags]",
				flags:       []flagDef{allowInsecureHTTPFlag},
				showExplain: true, explainOK: true,
				minPos: 1, maxPos: 2,
				posErr: "usage: gg clone <URL> [DIR]",
				setPos: func(req *Request, pos []string) error {
					req.CloneURL = pos[0]
					if len(pos) == 2 {
						req.CloneDir = pos[1]
					}
					return nil
				},
			},
			{
				name: "commit", summary: "Run git commit without signing", usage: "gg repo commit [git args]",
				passthrough: true, maxPos: -1,
			},
			{
				name: "pull", summary: "Run git pull", usage: "gg repo pull [git args]",
				passthrough: true, maxPos: -1,
			},
			{
				name: "push", summary: "Run git push", usage: "gg repo push [git args]",
				passthrough: true, maxPos: -1,
			},
		}, gitPassthroughActions()...),
	},
	"issue": {
		name:    "issue",
		summary: "List, view, create, comment, close, or reopen issues",
		desc:    "List, view, create, comment, close, or reopen issues.",
		usage:   "gg issue <command> [flags]",
		actions: []actionDef{
			{
				name: "list", summary: "List issues", usage: "gg issue list [flags]",
				flags:    []flagDef{stateFlag, limitFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				setPos: setState,
			},
			{
				name: "view", summary: "View one issue", usage: "gg issue view <number> [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg issue view <number>",
				setPos: setNumber,
			},
			{
				name: "create", summary: "Create an issue", usage: "gg issue create [flags]",
				flags:    []flagDef{titleFlag, bodyFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
			},
			{
				name: "comment", summary: "Comment on an issue", usage: "gg issue comment <number> [flags]",
				flags:    []flagDef{bodyFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg issue comment <number> --body <text>",
				setPos: func(req *Request, pos []string) error {
					if strings.TrimSpace(req.Body) == "" {
						return usageErr("usage: gg issue comment <number> --body <text>")
					}
					req.Number = pos[0]
					return nil
				},
			},
			{
				name: "close", summary: "Close an issue", usage: "gg issue close <number> [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg issue close <number>",
				setPos: setNumber,
			},
			{
				name: "reopen", summary: "Reopen an issue", usage: "gg issue reopen <number> [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg issue reopen <number>",
				setPos: setNumber,
			},
		},
	},
	"pr": {
		name:    "pr",
		summary: "List, view, create, or merge pull requests, and check merge readiness (alias: mr)",
		desc:    "List, view, create, or merge pull requests, and check merge readiness.",
		usage:   "gg pr <command> [flags]",
		actions: []actionDef{
			{
				name: "list", summary: "List pull requests", usage: "gg pr list [flags]",
				flags:    []flagDef{stateFlag, limitFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				setPos: setState,
			},
			{
				name: "view", summary: "View one pull request", usage: "gg pr view <number> [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg pr view <number>",
				setPos: setNumber,
			},
			{
				name: "create", summary: "Create a pull request", usage: "gg pr create [flags]",
				flags:    []flagDef{titleFlag, bodyFlag, baseFlag, headFlag, draftFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
			},
			{
				name: "status", summary: "Show merge readiness for one pull request", usage: "gg pr status <number> [flags]",
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg pr status <number>",
				setPos: setNumber,
			},
			{
				name: "ready", summary: "Mark a pull request as ready for review", usage: "gg pr ready <number> [flags]",
				flags:    []flagDef{undoFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg pr ready <number>",
				setPos: setNumber,
			},
			{
				name: "merge", summary: "Merge a pull request", usage: "gg pr merge <number> [flags]",
				flags:    []flagDef{mergeFlag, squashFlag, rebaseFlag, deleteBranchFlag, autoMergeFlag},
				showRepo: true, showRemote: true, showExplain: true,
				remoteOK: true, explainOK: true,
				minPos: 1, maxPos: 1,
				posErr: "usage: gg pr merge <number>",
				setPos: func(req *Request, pos []string) error {
					methods := 0
					for _, b := range []bool{req.Merge, req.Squash, req.Rebase} {
						if b {
							methods++
						}
					}
					if methods > 1 {
						return usageErr("--merge, --squash, --rebase are mutually exclusive; use at most one")
					}
					req.Number = pos[0]
					return nil
				},
			},
		},
	},
	"config": {
		name:    "config",
		summary: "Provider 설정 관리",
		desc:    "Manage Provider 설정 for self-hosted hosts.",
		usage:   "gg config <command>",
		actions: []actionDef{
			{
				name: "list", summary: "List Provider 설정", usage: "gg config list",
				posErr: "usage: gg config list",
			},
			{
				name: "set", summary: "Set the Provider for a host", usage: "gg config set <host> <gh|glab|tea>",
				posErr: "usage: gg config set <host> <provider>",
				minPos: 2, maxPos: 2,
				setPos: func(req *Request, pos []string) error {
					req.ConfigHost, req.ConfigProvider = pos[0], pos[1]
					return nil
				},
			},
			{
				name: "unset", summary: "Remove the Provider for a host", usage: "gg config unset <host>",
				posErr: "usage: gg config unset <host>",
				minPos: 1, maxPos: 1,
				setPos: func(req *Request, pos []string) error {
					req.ConfigHost = pos[0]
					return nil
				},
			},
		},
	},
}

// commandOrder는 최상위 help의 resource 표시 순서다.
var commandOrder = []string{"repo", "issue", "pr", "config"}

// commandAlias는 alias가 가리키는 canonical 명령 경로다.
// action이 비어 있으면 resource alias이고, 있으면 resource action alias다.
type commandAlias struct {
	resource string
	action   string
}

// commandAliases는 alias를 실제 구현이 등록된 canonical 명령 경로로 연결한다.
var commandAliases = func() map[string]commandAlias {
	aliases := map[string]commandAlias{
		"mr": {resource: "pr"},
	}
	for _, action := range commandDefs["repo"].actions {
		if _, isResource := commandDefs[action.name]; isResource {
			continue
		}
		if _, exists := aliases[action.name]; exists {
			continue
		}
		aliases[action.name] = commandAlias{resource: "repo", action: action.name}
	}
	return aliases
}()

func resolveAlias(command string) (resource, action string) {
	if canonical, ok := commandAliases[command]; ok {
		return canonical.resource, canonical.action
	}
	return command, ""
}

// helpAliases는 repo 생략 형태로 --help를 제공하는 action이다.
// --help도 git에 전달해야 하는 alias는 제외한다.
var helpAliases = map[string]bool{
	"list": true, "view": true, "create": true, "clone": true, "pull": true, "push": true,
}

const topLevelHelpHead = `gg sends common Git forge commands to gh, glab, or tea, and runs supported Git commands directly.

Usage:
  gg [flags] <command>
  gg <supported-git-command> [git args]
  gg repo --help
  gg config --help
  gg issue --help
  gg issue list --help
  gg pr create --help
  gg pr status --help
  gg pr ready --help
  gg pr merge --help

Commands:
`

const topLevelHelpTail = `
Flags:
  --repo <URL>      이 URL을 저장소 문맥으로 사용
  --remote <name>   이 Git remote를 저장소 문맥으로 사용
  --explain         선택한 저장소 문맥, Provider, 실행할 CLI를 설명
  -h, --help        Show top-level help
  --version         gg 버전만 표시
  -v, -verison      단독 사용 시 gg와 설치된 git, gh, glab, tea 버전을 표시`

// topLevelHelp는 최상위 help를 명령 정의에서 만든다.
func topLevelHelp() string {
	var b strings.Builder
	b.WriteString(topLevelHelpHead)
	rows := make([][2]string, 0, len(commandOrder)+len(commandDefs["repo"].actions)+2)
	for _, name := range commandOrder {
		rows = append(rows, [2]string{name, commandDefs[name].summary})
	}
	for _, action := range commandDefs["repo"].actions {
		if action.passthrough {
			rows = append(rows, [2]string{action.name, action.summary})
		}
	}
	rows = append(rows,
		[2]string{"version", "Show gg version"},
		[2]string{"help", "Show this help"})
	b.WriteString(renderRows(rows, 4))
	b.WriteString(topLevelHelpTail)
	return b.String()
}

// renderResourceHelp는 resource 단계 help를 만든다.
func renderResourceHelp(rd *resourceDef) string {
	var b strings.Builder
	b.WriteString(rd.desc + "\n\nUsage:\n  " + rd.usage + "\n\nCommands:\n")
	rows := make([][2]string, 0, len(rd.actions))
	for i := range rd.actions {
		rows = append(rows, [2]string{rd.actions[i].name, rd.actions[i].summary})
	}
	b.WriteString(renderRows(rows, 3))
	b.WriteString("\nFlags:\n")
	b.WriteString(renderFlagLines(resourceFlags(rd)))
	return strings.TrimSuffix(b.String(), "\n")
}

// resourceFlags는 resource의 모든 action이 함께 지원하는 flag만 모은다.
func resourceFlags(rd *resourceDef) []flagDef {
	repo, remote, explain := true, true, true
	for i := range rd.actions {
		ad := &rd.actions[i]
		repo = repo && ad.showRepo
		remote = remote && ad.showRemote
		explain = explain && ad.showExplain
	}
	flags := make([]flagDef, 0, 4)
	if repo {
		flags = append(flags, repoContextFlag)
	}
	if remote {
		flags = append(flags, remoteContextFlag)
	}
	if explain {
		flags = append(flags, explainFlag)
	}
	return append(flags, helpFlag)
}

// renderActionHelp는 action 단계 help를 만든다.
func renderActionHelp(rd *resourceDef, ad *actionDef) string {
	var b strings.Builder
	b.WriteString(ad.summary + ".\n\nUsage:\n  " + ad.usage + "\n\nFlags:\n")
	b.WriteString(renderFlagLines(actionFlags(ad)))
	return strings.TrimSuffix(b.String(), "\n")
}

// actionFlags는 action 자체 flag와 문맥 flag, --help를 순서대로 모은다.
func actionFlags(ad *actionDef) []flagDef {
	flags := make([]flagDef, 0, len(ad.flags)+4)
	flags = append(flags, ad.flags...)
	if ad.showRepo {
		flags = append(flags, repoContextFlag)
	}
	if ad.showRemote {
		flags = append(flags, remoteContextFlag)
	}
	if ad.showExplain {
		flags = append(flags, explainFlag)
	}
	return append(flags, helpFlag)
}

// nestedHelp는 마지막 인자가 --help이고 나머지가 명령 경로일 때 그 경로의 help를 돌려준다.
func nestedHelp(args []string) (string, bool) {
	if len(args) < 2 || args[len(args)-1] != "--help" {
		return "", false
	}
	path := args[:len(args)-1]
	hasLeadingGlobal := false
	// gg 전역 flag는 일반 명령의 help 대상을 바꾸지 않는다. passthrough에는
	// 저장소 문맥이나 설명 모드가 없으므로 ParseRequest가 이를 검증하게 둔다.
	for len(path) >= 2 && (path[0] == "--repo" || path[0] == "--remote") {
		hasLeadingGlobal = true
		path = path[2:]
	}
	if len(path) >= 1 && path[0] == "--explain" {
		hasLeadingGlobal = true
		path = path[1:]
	}
	if len(path) == 0 {
		return "", false
	}

	head, aliasAction := resolveAlias(path[0])
	if aliasAction != "" {
		if len(path) != 1 || !helpAliases[path[0]] {
			return "", false
		}
		rd := commandDefs[head]
		ad := rd.action(aliasAction)
		if hasLeadingGlobal && ad.passthrough {
			return "", false
		}
		return renderActionHelp(rd, ad), true
	}

	switch len(path) {
	case 1:
		if rd, ok := commandDefs[head]; ok {
			return renderResourceHelp(rd), true
		}
	case 2:
		if rd, ok := commandDefs[head]; ok {
			if ad := rd.action(path[1]); ad != nil {
				if (rd.name == "repo" && isGitPassthroughAction(ad.name)) || (hasLeadingGlobal && ad.passthrough) {
					return "", false
				}
				return renderActionHelp(rd, ad), true
			}
		}
	}
	return "", false
}

// renderRows는 "이름 설명" 행을 name 열 기준으로 맞춰 만든다.
func renderRows(rows [][2]string, gap int) string {
	width := 0
	for _, r := range rows {
		if len(r[0]) > width {
			width = len(r[0])
		}
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString("  " + r[0] + strings.Repeat(" ", width-len(r[0])+gap) + r[1] + "\n")
	}
	return b.String()
}

// renderFlagLines는 flag 목록을 이름 열 기준으로 맞춰 만든다.
func renderFlagLines(flags []flagDef) string {
	rows := make([][2]string, 0, len(flags))
	for _, f := range flags {
		name := f.name
		if f.arg != "" {
			name += " " + f.arg
		}
		rows = append(rows, [2]string{name, f.desc})
	}
	return strings.TrimSuffix(renderRows(rows, 3), "\n")
}

// actionFlagMaps는 action 정의의 flag을 Request 필드와 연결한다.
func actionFlagMaps(ad *actionDef, req *Request) (map[string]*string, map[string]*bool) {
	strs := make(map[string]*string)
	bools := make(map[string]*bool)
	for i := range ad.flags {
		f := &ad.flags[i]
		switch {
		case f.str != nil:
			strs[f.name] = f.str(req)
		case f.bin != nil:
			bools[f.name] = f.bin(req)
		}
	}
	return strs, bools
}

// needsAction은 action이 빠졌을 때 오류 메시지를 만든다.
func needsAction(head string, rd *resourceDef) string {
	names := make([]string, 0, len(rd.actions))
	for i := range rd.actions {
		names = append(names, rd.actions[i].name)
	}
	return head + " needs an action: " + strings.Join(names, ", ")
}
