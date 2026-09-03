package cli

// prResourceDef는 "pr" 최상위 명령의 정의다: list, view, create, status, ready, merge.
// alias: mr (command_registry.go의 commandAliases에서 연결).
var prResourceDef = &resourceDef{
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
}

var prListBuilders = providerBuilders{
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

var prViewBuilders = providerBuilders{
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

var prCreateBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		args = appendKV(args, "--base", c.req.Base)
		args = appendKV(args, "--head", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		args = appendKV(args, "--target-branch", c.req.Base)
		args = appendKV(args, "--source-branch", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		args = appendKV(args, "--base", c.req.Base)
		args = appendKV(args, "--head", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
}

// prInvocationTable은 "pr <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// tea의 pr merge/status/ready는 teaInvocation의 사전 가드에서 걸러지므로 여기에는
// 등록하지 않는다 — provider별 예외는 감추지 않고 그 함수에 명시적으로 남긴다.
var prInvocationTable = map[string]providerBuilders{
	"pr list": prListBuilders,
	"pr view": prViewBuilders,
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
	"pr create": prCreateBuilders,
}
