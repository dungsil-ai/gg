package cli

import "strings"

// labelResourceDef는 "label" 최상위 명령의 정의다: list, create. glab builder만
// 등록하며, gh/tea는 각 invocation 함수의 사전 가드에서 미지원을 확정한다
// (pr ready의 tea 가드와 같은 원칙: provider별 예외는 감추지 않고 명시적으로 남긴다).
var labelResourceDef = &resourceDef{
	name:    "label",
	summary: "List or create labels",
	desc:    "List or create labels.",
	usage:   "gg label <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List labels", usage: "gg label list [flags]",
			flags:    []flagDef{limitFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "create", summary: "Create a label", usage: "gg label create [flags]",
			flags:    []flagDef{nameFlag, colorFlag, descriptionFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.Name) == "" {
					return usageErr("usage: gg label create --name <text>")
				}
				return nil
			},
		},
	},
}

var labelListBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--per-page", c.req.Limit), nil
	},
}

var labelCreateBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--name", c.req.Name)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
}

// labelInvocationTable은 "label <action>" 키로 glab의 arg-builder를 모은다.
var labelInvocationTable = map[string]providerBuilders{
	"label list":   labelListBuilders,
	"label create": labelCreateBuilders,
}
