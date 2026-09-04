package cli

// ciResourceDef는 "ci" 최상위 명령의 정의다: list, view, watch, retry, cancel.
// alias: actions (command_registry.go의 commandAliases에서 연결). GitHub은 gh run,
// GitLab은 glab ci로 중계하고, tea는 teaInvocation의 사전 가드에서 미지원으로 걸러진다.
var ciResourceDef = &resourceDef{
	name:    "ci",
	summary: "List, view, watch, retry, or cancel CI runs and pipelines (alias: actions)",
	desc:    "List, view, watch, retry, or cancel CI runs and pipelines.",
	usage:   "gg ci <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List CI runs or pipelines", usage: "gg ci list [flags]",
			flags:    []flagDef{limitFlag, branchFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "view", summary: "View one CI run or pipeline (default: latest on the current branch)", usage: "gg ci view [<id>] [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			maxPos: 1,
			posErr: "usage: gg ci view [<id>]",
			setPos: setNumber,
		},
		{
			name: "watch", summary: "Watch CI progress live (GitLab: job id)", usage: "gg ci watch [<id>] [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			maxPos: 1,
			posErr: "usage: gg ci watch [<id>]",
			setPos: setNumber,
		},
		{
			name: "retry", summary: "Retry a CI run or job", usage: "gg ci retry <id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg ci retry <id>",
			setPos: setNumber,
		},
		{
			name: "cancel", summary: "Cancel a CI run or pipeline", usage: "gg ci cancel <id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg ci cancel <id>",
			setPos: setNumber,
		},
	},
}

// ciInvocationTable은 "ci <action>" 키로 gh/glab의 arg-builder를 모은다. GitHub
// Actions의 단위는 workflow run이고 GitLab은 pipeline(단, watch/retry는 job)이라는
// 단위 차이는 각 builder가 감싸는 하위 명령에 명시적으로 남긴다.
var ciInvocationTable = map[string]providerBuilders{
	"ci list": {
		gh: func(c invocationContext) (args, env []string) {
			args = append([]string{"run", "list"}, c.target...)
			args = appendKV(args, "--branch", c.req.Branch)
			args = appendKV(args, "--limit", c.req.Limit)
			return args, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = append([]string{c.res, "list"}, c.target...)
			args = appendKV(args, "--ref", c.req.Branch)
			return appendKV(args, "--per-page", c.req.Limit), nil
		},
	},
	"ci view": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"run", "view"}
			if c.req.Number != "" {
				args = append(args, c.req.Number)
			}
			return append(args, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = appendKV([]string{c.res, "get"}, "--pipeline-id", c.req.Number)
			return append(args, c.target...), nil
		},
	},
	"ci watch": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"run", "watch"}
			if c.req.Number != "" {
				args = append(args, c.req.Number)
			}
			return append(args, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{c.res, "trace"}
			if c.req.Number != "" {
				args = append(args, c.req.Number)
			}
			return append(args, c.target...), nil
		},
	},
	"ci retry": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"run", "rerun", c.req.Number}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "retry", c.req.Number}, c.target...), nil
		},
	},
	"ci cancel": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"run", "cancel", c.req.Number}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "cancel", "pipeline", c.req.Number}, c.target...), nil
		},
	},
}
