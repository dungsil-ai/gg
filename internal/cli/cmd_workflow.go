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
			flags:    []flagDef{workflowRefFlag, workflowYamlFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg workflow view <name-or-id>",
			setPos: func(req *Request, pos []string) error {
				// gh workflow view는 --ref를 --yaml과 함께 쓰지 않으면 거부한다.
				// gh가 나중에 틀리게 알려주는 대신 gg가 먼저 사용법 오류로 막는다.
				if req.Ref != "" && !req.Yaml {
					return usageErr("workflow view --ref needs --yaml")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "run", summary: "Run a workflow with a workflow_dispatch event", usage: "gg workflow run <name-or-id> [flags]",
			flags:    []flagDef{workflowRefFlag},
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

// workflowRefFlag는 workflow view·run의 --ref다. release의 --ref(tag 생성
// 기준)와 뜻이 다르므로 help 문구를 따로 둔다.
var workflowRefFlag = flagDef{name: "--ref", arg: "<ref>", desc: "Branch or tag to run or view the workflow from",
	str: func(r *Request) *string { return &r.Ref }}

// workflowYamlFlag는 workflow view의 --yaml이다. gh는 --ref와 함께 쓰기를
// 요구하므로 view의 setPos가 ref⇒yaml 관계를 검증한다.
var workflowYamlFlag = flagDef{name: "--yaml", desc: "View the workflow YAML content",
	bin: func(r *Request) *bool { return &r.Yaml }}

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
			// gh workflow view는 --ref를 --yaml 없이 쓰면 거부한다. parse 단계에서
			// ref⇒yaml을 검증하므로 여기서는 요청받은 flag를 그대로 옮긴다.
			args = append([]string{"workflow", "view", c.req.Number}, c.target...)
			if c.req.Yaml {
				args = append(args, "--yaml")
			}
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
