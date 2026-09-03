package cli

import "strings"

// issueResourceDef는 "issue" 최상위 명령의 정의다: list, view, create, comment,
// close, reopen.
var issueResourceDef = &resourceDef{
	name:    "issue",
	summary: "List, view, create, comment, close, or reopen issues",
	desc:    "List, view, create, comment, close, or reopen issues.",
	usage:   "gg issue <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List issues", usage: "gg issue list [flags]",
			flags:    []flagDef{stateFlag, limitFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			setPos: setState,
		},
		{
			name: "view", summary: "View one issue", usage: "gg issue view <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue view <number>",
			setPos: setNumber,
		},
		{
			name: "create", summary: "Create an issue", usage: "gg issue create [flags]",
			flags:    []flagDef{titleFlag, bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "comment", summary: "Comment on an issue", usage: "gg issue comment <number> [flags]",
			flags:    []flagDef{bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue comment <number> --body <text>",
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.Body) == "" {
					return usageErr("usage: gg issue comment <number> --body <text>")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "close", summary: "Close an issue", usage: "gg issue close <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue close <number>",
			setPos: setNumber,
		},
		{
			name: "reopen", summary: "Reopen an issue", usage: "gg issue reopen <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue reopen <number>",
			setPos: setNumber,
		},
	},
}

var issueListBuilders = providerBuilders{
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

var issueViewBuilders = providerBuilders{
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

var issueCloseReopenBuilders = providerBuilders{
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

var issueCreateBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		return args, nil
	},
}

// issueInvocationTable은 "issue <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
var issueInvocationTable = map[string]providerBuilders{
	"issue list": issueListBuilders,
	"issue view": issueViewBuilders,
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
	"issue close":  issueCloseReopenBuilders,
	"issue reopen": issueCloseReopenBuilders,
	"issue create": issueCreateBuilders,
}
