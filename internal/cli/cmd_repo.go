package cli

// repoResourceDef는 "repo" 최상위 명령의 정의다: list, view, create, clone, commit,
// pull, push와 Git 전달 명령(cmd_git.go) 전체를 포함한다.
var repoResourceDef = &resourceDef{
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
}
