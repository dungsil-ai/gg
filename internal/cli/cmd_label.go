package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// labelResourceDef는 "label" 최상위 명령의 정의다: list, create, edit, clone,
// delete.
// list·create·delete는 gh, glab, tea builder를 등록한다. edit는 이름 대신
// numeric label id를 요구하는 glab(--label-id)·tea(--id)를 위해 이름→id 사전
// 조회(resolvePlan의 forgeLabelID)를 거친 뒤 중계한다. clone은 gh 전용이다.
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
			name: "edit", summary: "Edit a label", usage: "gg label edit <name> [flags]",
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

// labelEditBuilders는 label 이름 바꾸기와 색·설명 수정을 중계한다. positional은
// 고칠 label이고 --name은 새 이름이다. glab label edit은 numeric label id
// (--label-id)를, tea labels update도 numeric label id(--id)를 요구하므로
// resolvePlan의 forgeLabelID가 미리 조회한 req.LabelID를 쓴다. 새 이름 flag는
// provider마다 다르다(gh --name, glab --new-name, tea --name).
var labelEditBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "edit", c.req.Name}, c.target...)
		args = appendKV(args, "--name", c.req.NewName)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = []string{c.res, "edit", "--label-id", c.req.LabelID}
		args = appendKV(args, "--new-name", c.req.NewName)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return append(args, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "update", "--id", c.req.LabelID}, c.target...)
		args = appendKV(args, "--name", c.req.NewName)
		args = appendKV(args, "--color", c.req.Color)
		args = appendKV(args, "--description", c.req.Description)
		return args, nil
	},
}

// labelDeleteBuilders는 label 삭제를 중계한다. gh는 대화형 확인을 건너뛰는
// --yes flag가 있지만 glab·tea에는 확인 flag가 없으므로 gg의 --yes는 gh에만
// 전달한다 (issue delete와 같은 원칙). glab label delete는 이름을 positional으로
// 받지만 tea labels delete는 numeric label id(--id 필수)를 요구하므로
// forgeLabelID로 조회한 req.LabelID를 쓴다.
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
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "delete", "--id", c.req.LabelID}, c.target...), nil
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
var labelInvocationTable = map[string]providerBuilders{
	"label clone":  labelCloneBuilders,
	"label list":   labelListBuilders,
	"label create": labelCreateBuilders,
	"label edit":   labelEditBuilders,
	"label delete": labelDeleteBuilders,
}

// forgeLabelID는 label 이름을 numeric label id로 바꾼다. glab label edit과
// tea labels update·delete가 id를 요구하기 때문이다. 두 provider의 api 목록
// 응답은 같은 {"id", "name"} 배열이다 — glab은 glab api, tea는 tea api로
// 조회한다.
func forgeLabelID(p Provider, r RepoURL, name, teaLogin string) (string, error) {
	var out string
	var err error
	switch p {
	case GLab:
		inv := Invocation{
			Bin:  "glab",
			Args: []string{"api", "projects/" + url.PathEscape(r.Slug()) + "/labels"},
			Env:  []string{"GITLAB_HOST=" + r.Host},
		}
		out, err = captureChild(inv)
	case Tea:
		if teaLogin == "" {
			return "", teaLoginError(r.Host)
		}
		out, err = runOut("tea", "api", "repos/"+r.Slug()+"/labels", "--login", teaLogin, "--repo", r.Slug())
	default:
		return "", fmt.Errorf("label id lookup is not supported for %s", string(p))
	}
	if err != nil {
		return "", fmt.Errorf("cannot look up label %q: %w", name, err)
	}
	return findLabelID(out, name)
}

// findLabelID는 label 목록 JSON에서 이름이 일치하는 label의 id를 찾는다.
// 정확히 일치하는 이름이 없으면 대소문자를 무시하고 찾는다.
func findLabelID(out, name string) (string, error) {
	var labels []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &labels); err != nil {
		return "", fmt.Errorf("cannot parse label list output: %w", err)
	}
	for _, l := range labels {
		if l.Name == name {
			return strconv.FormatInt(l.ID, 10), nil
		}
	}
	lower := strings.ToLower(name)
	for _, l := range labels {
		if strings.ToLower(l.Name) == lower {
			return strconv.FormatInt(l.ID, 10), nil
		}
	}
	return "", fmt.Errorf("label %q not found", name)
}
