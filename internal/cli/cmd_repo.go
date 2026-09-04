package cli

// repoResourceDef는 "repo" 최상위 명령의 정의다: list, view, create, clone, fork,
// delete, edit, rename, sync, set-default, commit, pull, push와 Git 전달 명령
// (cmd_git.go) 전체를 포함한다. archive는 git archive passthrough가 쓰는 이름이라
// gh repo archive(저장소 보관)는 이 자리에 둘 수 없다.
var repoResourceDef = &resourceDef{
	name:    "repo",
	summary: "List, view, create, or manage repositories, or run supported Git commands",
	desc:    "List, view, create, or manage repositories, or run supported Git commands.",
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
			name: "fork", summary: "Create a fork of the repository", usage: "gg repo fork [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "delete", summary: "Delete a repository", usage: "gg repo delete [flags]",
			flags:    []flagDef{yesFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "edit", summary: "Edit repository settings (GitHub only)", usage: "gg repo edit [flags]",
			flags:    []flagDef{descriptionFlag, editPublicFlag, editPrivateFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			setPos: func(req *Request, pos []string) error {
				if req.Public && req.Private {
					return usageErr("repo edit needs at most one of --public or --private")
				}
				if req.Description == "" && !req.Public && !req.Private {
					return usageErr("repo edit needs at least one of --description, --public, or --private")
				}
				return nil
			},
		},
		{
			name: "rename", summary: "Rename a repository (GitHub only)", usage: "gg repo rename [<new-name>] [flags]",
			flags:    []flagDef{yesFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			maxPos: 1,
			posErr: "usage: gg repo rename [<new-name>]",
			setPos: func(req *Request, pos []string) error {
				if len(pos) == 1 {
					req.Name = pos[0]
				}
				return nil
			},
		},
		{
			name: "sync", summary: "Sync a repository (GitHub only)", usage: "gg repo sync [flags]",
			flags:    []flagDef{syncBranchFlag, sourceFlag, forceFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			// set-default는 gh의 디렉터리별 기본 저장소 설정을 건드리는 로컬 명령이다.
			// --unset/--view는 저장소 문맥이 필요 없으므로 explain과 함께 지원하지 않는다.
			name: "set-default", summary: "Set the default repository for this directory (GitHub only)", usage: "gg repo set-default [flags]",
			flags:    []flagDef{unsetDefaultFlag, viewDefaultFlag},
			showRepo: true, showRemote: true,
			remoteOK: true,
			setPos: func(req *Request, pos []string) error {
				if req.Unset && req.View {
					return usageErr("repo set-default needs at most one of --unset or --view")
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
}

// repoInvocationTable은 "repo <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
var repoInvocationTable = map[string]providerBuilders{
	"repo list": {
		gh: func(c invocationContext) (args, env []string) {
			args = appendKV([]string{"repo", "list"}, "--limit", c.req.Limit)
			if c.r.Host != "github.com" {
				env = []string{"GH_HOST=" + c.r.Host}
			}
			return args, env
		},
		glab: func(c invocationContext) (args, env []string) {
			return appendKV([]string{"repo", "list"}, "--per-page", c.req.Limit), []string{"GITLAB_HOST=" + c.r.Host}
		},
		tea: func(c invocationContext) (args, env []string) {
			return appendKV(append([]string{"repos", "list"}, c.auth...), "--limit", c.req.Limit), nil
		},
	},
	"repo view": {
		gh: func(c invocationContext) (args, env []string) {
			return []string{"repo", "view", c.r.HTTPS()}, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return []string{"repo", "view", c.r.HTTPS()}, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{"repos", c.r.Slug()}, c.auth...), nil
		},
	},
	"repo create": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "create", c.r.Slug(), visFlag(c.req)}
			args = appendKV(args, "--description", c.req.Description)
			if c.r.Host != "github.com" {
				env = []string{"GH_HOST=" + c.r.Host}
			}
			return args, env
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "create", c.r.Slug(), visFlag(c.req)}
			args = appendKV(args, "--description", c.req.Description)
			return args, []string{"GITLAB_HOST=" + c.r.Host}
		},
		tea: func(c invocationContext) (args, env []string) {
			args = append([]string{"repos", "create"}, c.auth...)
			args = append(args, "--owner", c.r.Owner, "--name", c.r.Name)
			if c.req.Private {
				args = append(args, "--private")
			}
			return appendKV(args, "--description", c.req.Description), nil
		},
	},
	"repo clone": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			args = []string{"clone", c.req.CloneURL}
			if c.req.CloneDir != "" {
				args = append(args, c.req.CloneDir)
			}
			return args, nil
		},
	},
	// fork는 gh/glab/tea 모두 원본 저장소를 복제하는 같은 의미라 세 provider를
	// 중계한다. fork 대상 소유자를 고르는 flag는 provider마다 체계가 달라
	// (gh --org, glab --name/--path, tea --owner) 공통 flag로 내보내지 않는다.
	"repo fork": {
		gh: func(c invocationContext) (args, env []string) {
			return []string{"repo", "fork", c.r.HTTPS()}, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return []string{"repo", "fork", c.r.Slug()}, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{"repos", "fork"}, c.target...), nil
		},
	},
	// delete는 위험한 명령이라 확인 생략 flag만 provider별 이름으로 옮긴다.
	// gh/glab은 --yes, tea는 --force다. tea는 --owner·--name을 따로 받으므로
	// GitLab namespace처럼 /를 포함하는 owner와 구분해 나눠 넘긴다.
	"repo delete": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "delete", c.r.HTTPS()}
			if c.req.Yes {
				args = append(args, "--yes")
			}
			return args, nil
		},
		glab: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "delete", c.r.Slug()}
			if c.req.Yes {
				args = append(args, "--yes")
			}
			return args, nil
		},
		tea: func(c invocationContext) (args, env []string) {
			args = append([]string{"repos", "delete"}, c.auth...)
			args = append(args, "--owner", c.r.Owner, "--name", c.r.Name)
			if c.req.Yes {
				args = append(args, "--force")
			}
			return args, nil
		},
	},
	// edit/rename/sync/set-default는 glab과 tea에 대응 하위 명령이 없다. builder를
	// 등록하지 않아 dispatch의 미지원 오류로 걸러진다 (release edit과 같은 원칙).
	"repo edit": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "edit", c.r.HTTPS()}
			args = appendKV(args, "--description", c.req.Description)
			if c.req.Public || c.req.Private {
				if c.req.Private {
					args = append(args, "--visibility", "private")
				} else {
					args = append(args, "--visibility", "public")
				}
				// 최근 gh는 가시성 변경 시 영향 수락 flag을 요구한다.
				args = append(args, "--accept-visibility-change-consequences")
			}
			return args, nil
		},
	},
	"repo rename": {
		gh: func(c invocationContext) (args, env []string) {
			// rename의 positional은 새 저장소 이름이므로 대상 저장소는 -R로 넘긴다.
			args = []string{"repo", "rename"}
			if c.req.Name != "" {
				args = append(args, c.req.Name)
			}
			args = append(args, c.target...)
			if c.req.Yes {
				args = append(args, "--yes")
			}
			return args, nil
		},
	},
	"repo sync": {
		gh: func(c invocationContext) (args, env []string) {
			args = []string{"repo", "sync", c.r.HTTPS()}
			args = appendKV(args, "--branch", c.req.Branch)
			args = appendKV(args, "--source", c.req.Source)
			if c.req.Force {
				args = append(args, "--force")
			}
			return args, nil
		},
	},
	"repo set-default": {
		gh: func(c invocationContext) (args, env []string) {
			return []string{"repo", "set-default", c.r.HTTPS()}, nil
		},
	},
}
