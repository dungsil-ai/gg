package cli

import "strings"

// labelResourceDef는 "label" 최상위 명령의 정의다: list, create, edit, clone,
// delete.
// list와 create는 gh, glab, tea builder를, edit는 gh builder만, delete는 gh와
// glab builder를 등록한다. tea의 edit·delete는 label 이름이 아니라 numeric
// label id를 요구해 teaInvocation 사전 가드에서 미지원을 확정한다 (pr ready의
// tea 가드와 같은 원칙: provider별 예외는 감추지 않고 명시적으로 남긴다).
var labelResourceDef = &resourceDef{
	name:    "label",
	summary: "List, create, edit, clone, or delete labels",
	desc:    "List, create, edit, clone, or delete labels.",
	usage:   "gg label <command> [flags]",
	actions: []actionDef{
		{
			name: "clone", summary: "Clone labels from another repository (GitHub only)",
			usage:    "gg label clone <source-repository> [flags]",
			flags:    []flagDef{labelCloneForceFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg label clone <source-repository>",
			setPos: func(req *Request, pos []string) error {
				req.Source = pos[0]
				return nil
			},
		},
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
		{
			name: "edit", summary: "Edit a label (GitHub only)", usage: "gg label edit <name> [flags]",
			flags:    []flagDef{labelEditNameFlag, colorFlag, descriptionFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg label edit <name>",
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.NewName) == "" && strings.TrimSpace(req.Color) == "" &&
					strings.TrimSpace(req.Description) == "" {
					return usageErr("label edit needs --name, --color, or --description")
				}
				req.Name = pos[0]
				return nil
			},
		},
		{
			name: "delete", summary: "Delete a label", usage: "gg label delete <name> [flags]",
			flags:    []flagDef{yesFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg label delete <name>",
			setPos: func(req *Request, pos []string) error {
				req.Name = pos[0]
				return nil
			},
		},
	},
}

var labelListBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--limit", c.req.Limit), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--per-page", c.req.Limit), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--limit", c.req.Limit), nil
	},
}

var labelCreateBuilders = providerBuilders{
	// gh label create는 name을 positional 인자로 받는다(glab은 --name flag).
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create", c.req.Name}, c.target...)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--name", c.req.Name)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
	// tea labels create도 glab과 같은 --name/--color/--description 표면이다.
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--name", c.req.Name)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
}

// labelEditBuilders는 label 이름 바꾸기와 색·설명 수정을 gh label edit로
// 중계한다. positional은 고칠 label이고 --name은 새 이름이다. glab label edit은
// 이름이 아니라 numeric label id(--label-id)를 요구하고, tea labels update도
// numeric label id(--id)를 요구한다 — 두 provider 모두 builder가 없어 사전
// 가드(glab은 dispatch의 builder 부재 오류)로 걸러진다.
var labelEditBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "edit", c.req.Name}, c.target...)
		args = appendKV(args, "--name", c.req.NewName)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
}

// labelDeleteBuilders는 label 삭제를 중계한다. gh는 대화형 확인을 건너뛰는
// --yes flag가 있지만 glab에는 확인 flag가 없으므로 gg의 --yes는 gh에만
// 전달한다 (issue delete와 같은 원칙). tea labels delete도 numeric label
// id(--id)가 필수라 edit와 함께 사전 가드에서 미지원으로 걸러진다.
var labelDeleteBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{c.res, "delete", c.req.Name}
		if c.req.Yes {
			args = append(args, "--yes")
		}
		return append(args, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "delete", c.req.Name}, c.target...), nil
	},
}

// labelCloneBuilders는 다른 저장소의 label을 현재 저장소 문맥 저장소로 복사한다.
// gh 전용 기능이라 gh builder만 등록한다. glab은 dispatch의 builder 부재 오류로,
// tea는 사전 가드와 run.go의 tea login 건너뛰기 목록이 미지원을 확정한다.
var labelCloneBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "clone", c.req.Source}, c.target...)
		if c.req.Force {
			args = append(args, "--force")
		}
		return args, nil
	},
}

// labelInvocationTable은 "label <action>" 키로 provider별 arg-builder를 모은다.
// edit는 gh builder만 있어 glab은 dispatch의 미지원 오류로, tea는 사전 가드로
// 걸러진다. delete의 tea도 사전 가드로 걸러진다.
var labelInvocationTable = map[string]providerBuilders{
	"label clone":  labelCloneBuilders,
	"label list":   labelListBuilders,
	"label create": labelCreateBuilders,
	"label edit":   labelEditBuilders,
	"label delete": labelDeleteBuilders,
}
