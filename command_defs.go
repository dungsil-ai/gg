package main

import (
	"strings"
)

// 이 파일은 gg가 지원하는 명령, action, flag의 유일한 정의다.
// ParseRequest는 이 정의로 인자를 검사하고, help 렌더링도 같은 정의를 쓴다.

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
// Main Porcelain 37개, ancillary 14개, plumbing 70개를 하나의 registry에서 관리한다.
// clone, commit, pull, push는 기존의 별도 동작을 유지한다.
var gitPassthroughActionNames = []string{
	"add", "am", "archive", "bisect", "branch", "bundle", "checkout", "cherry-pick",
	"citool", "clean", "describe", "diff", "fetch", "format-patch", "gc", "grep", "gui",
	"init", "log", "merge", "mv", "notes", "range-diff", "rebase", "reset", "restore",
	"revert", "rm", "shortlog", "show", "sparse-checkout", "stash", "status", "submodule",
	"switch", "tag", "worktree",
	"annotate", "blame", "bugreport", "count-objects", "diagnose", "difftool", "fsck",
	"instaweb", "maintenance", "merge-tree", "mergetool", "prune-packed", "rerere", "scalar",
	"apply", "cat-file", "check-attr", "check-ignore", "check-mailmap", "check-ref-format",
	"checkout-index", "column", "commit-graph", "commit-tree", "credential", "credential-cache",
	"credential-store", "daemon", "diff-files", "diff-index", "diff-tree", "fast-export",
	"fast-import", "fetch-pack", "for-each-ref", "for-each-repo", "hash-object", "http-backend",
	"http-fetch", "http-push", "index-pack", "ls-files", "ls-remote", "ls-tree", "mailinfo",
	"mailsplit", "merge-base", "merge-file", "merge-index", "mktag", "mktree", "multi-pack-index",
	"name-rev", "pack-objects", "pack-redundant", "pack-refs", "patch-id", "prune", "read-tree",
	"receive-pack", "reflog", "remote", "repack", "replace", "rev-list", "rev-parse", "send-pack",
	"show-branch", "show-index", "show-ref", "stripspace", "symbolic-ref", "unpack-file",
	"unpack-objects", "update-index", "update-ref", "update-server-info", "upload-archive",
	"upload-pack", "var", "verify-commit", "verify-pack", "verify-tag", "write-tree",
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
