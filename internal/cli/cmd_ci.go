package cli

// ciResourceDef는 "ci" 최상위 명령의 정의다: list, view, watch, retry, cancel,
// delete, download, lint, run, status, trigger. alias: actions
// (command_registry.go의 commandAliases에서 연결). GitHub은 gh run, GitLab은
// glab ci로 중계하고, tea는 teaInvocation의 사전 가드에서 미지원으로 걸러진다.
// download는 gh 전용, lint·run·status·trigger는 glab 전용이다.
var ciResourceDef = &resourceDef{
	name:    "ci",
	summary: "List, view, watch, retry, cancel, or delete CI runs and pipelines, download artifacts, lint or run pipelines, and trigger manual jobs (alias: actions)",
	desc:    "List, view, watch, retry, cancel, or delete CI runs and pipelines, download artifacts, lint or run pipelines, and trigger manual jobs.",
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
		{
			name: "delete", summary: "Delete a CI run or pipeline", usage: "gg ci delete <id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg ci delete <id>",
			setPos: setNumber,
		},
		{
			name: "download", summary: "Download artifacts of a CI run (GitHub only)", usage: "gg ci download <id> [flags]",
			flags:    []flagDef{patternFlag, dirFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg ci download <id>",
			setPos: setNumber,
		},
		{
			name: "lint", summary: "Validate a CI configuration file (GitLab only)", usage: "gg ci lint [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "run", summary: "Run a pipeline for the current or given branch (GitLab only)", usage: "gg ci run [--branch <branch>] [flags]",
			flags:    []flagDef{branchFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "status", summary: "Show pipeline status for a branch (GitLab only)", usage: "gg ci status [--branch <branch>] [flags]",
			flags:    []flagDef{branchFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "trigger", summary: "Trigger a manual job (GitLab only)", usage: "gg ci trigger <job-id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg ci trigger <job-id>",
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
	"ci delete": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"run", "delete", c.req.Number}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "delete", c.req.Number}, c.target...), nil
		},
	},
	// download는 GitHub Actions의 run artifact 전용 기능이라 gh builder만 등록한다.
	"ci download": {
		gh: func(c invocationContext) (args, env []string) {
			args = append([]string{"run", "download", c.req.Number}, c.target...)
			args = appendKV(args, "--pattern", c.req.Pattern)
			args = appendKV(args, "--dir", c.req.Dir)
			return args, nil
		},
	},
	// lint와 run은 GitLab CI 전용 기능이라 glab builder만 등록한다 — gh와 tea는
	// dispatch의 builder 부재 오류로 걸러진다.
	"ci lint": {
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "lint"}, c.target...), nil
		},
	},
	"ci run": {
		glab: func(c invocationContext) (args, env []string) {
			args = append([]string{c.res, "run"}, c.target...)
			return appendKV(args, "--branch", c.req.Branch), nil
		},
	},
	"ci status": {
		glab: func(c invocationContext) (args, env []string) {
			args = append([]string{c.res, "status"}, c.target...)
			return appendKV(args, "--branch", c.req.Branch), nil
		},
	},
	"ci trigger": {
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "trigger", c.req.Number}, c.target...), nil
		},
	},
}
