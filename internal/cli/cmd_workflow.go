package cli

// workflowResourceDef는 "workflow" 최상위 명령의 정의다: list, view, run,
// enable, disable. GitHub Actions의 workflow 파일 전용 기능이라 gh builder만
// 등록한다 — glab과 tea는 dispatch의 builder 부재 오류로 걸러지며, tea는
// run.go의 tea login 건너뛰기 목록이 login을 묻기 전에 미지원을 확정한다.
var workflowResourceDef = &resourceDef{
	name:    "workflow",
	summary: "List, view, run, enable, or disable GitHub Actions workflows",
	desc:    "List, view, run, enable, or disable GitHub Actions workflows.",
	usage:   "gg workflow <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List workflows", usage: "gg workflow list [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "view", summary: "View a workflow", usage: "gg workflow view <name-or-id> [flags]",
			flags:    []flagDef{refFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg workflow view <name-or-id>",
			setPos: setNumber,
		},
		{
			name: "run", summary: "Run a workflow with a workflow_dispatch event", usage: "gg workflow run <name-or-id> [flags]",
			flags:    []flagDef{refFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg workflow run <name-or-id>",
			setPos: setNumber,
		},
		{
			name: "enable", summary: "Enable a workflow", usage: "gg workflow enable <name-or-id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg workflow enable <name-or-id>",
			setPos: setNumber,
		},
		{
			name: "disable", summary: "Disable a workflow", usage: "gg workflow disable <name-or-id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg workflow disable <name-or-id>",
			setPos: setNumber,
		},
	},
}

// workflowInvocationTable은 "workflow <action>" 키로 provider별 arg-builder를
// 모은다. 전부 gh builder만 등록해 glab/tea는 dispatch의 미지원 오류로 걸러진다.
var workflowInvocationTable = map[string]providerBuilders{
	"workflow list": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"workflow", "list"}, c.target...), nil
		},
	},
	"workflow view": {
		gh: func(c invocationContext) (args, env []string) {
			args = append([]string{"workflow", "view", c.req.Number}, c.target...)
			return appendKV(args, "--ref", c.req.Ref), nil
		},
	},
	"workflow run": {
		gh: func(c invocationContext) (args, env []string) {
			args = append([]string{"workflow", "run", c.req.Number}, c.target...)
			return appendKV(args, "--ref", c.req.Ref), nil
		},
	},
	"workflow enable": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"workflow", "enable", c.req.Number}, c.target...), nil
		},
	},
	"workflow disable": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"workflow", "disable", c.req.Number}, c.target...), nil
		},
	},
}
