package main

var listBuilders = providerBuilders{
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

var viewBuilders = providerBuilders{
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

var closeReopenBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{"issue", c.req.Action, c.req.Number}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{"issue", c.req.Action, c.req.Number}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{"issues", c.req.Action, c.req.Number}, c.target...), nil
	},
}

var createBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--base", c.req.Base)
			args = appendKV(args, "--head", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--target-branch", c.req.Base)
			args = appendKV(args, "--source-branch", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		if c.req.Resource == "pr" {
			args = appendKV(args, "--base", c.req.Base)
			args = appendKV(args, "--head", c.req.Head)
			if c.req.Draft {
				args = append(args, "--draft")
			}
		}
		return args, nil
	},
}

// invocationTable은 "<resource> <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// tea의 pr merge/status/ready는 teaInvocation의 사전 가드에서 걸러지므로 여기에는
// 등록하지 않는다 — provider별 예외는 감추지 않고 그 함수에 명시적으로 남긴다.
var invocationTable = map[string]providerBuilders{
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
	"issue list": listBuilders,
	"pr list":    listBuilders,
	"issue view": viewBuilders,
	"pr view":    viewBuilders,
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
	"issue comment": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"issue", "comment", c.req.Number, "--body", c.req.Body}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{"issue", "note", c.req.Number, "--message", c.req.Body}, c.target...), nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{"comment", c.req.Number, c.req.Body}, c.target...), nil
		},
	},
	"issue close":  closeReopenBuilders,
	"issue reopen": closeReopenBuilders,
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
	"issue create": createBuilders,
	"pr create":    createBuilders,
}
