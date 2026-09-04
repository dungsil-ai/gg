package cli

import (
	"net/url"
	"strings"
)

// prResourceDef는 "pr" 최상위 명령의 정의다: list, view, create, comment(하위
// list/edit/delete), status, ready, merge, close, reopen.
// alias: mr (command_registry.go의 commandAliases에서 연결).
var prResourceDef = &resourceDef{
	name:    "pr",
	summary: "List, view, create, comment on, merge, or close pull requests, and check merge readiness (alias: mr)",
	desc:    "List, view, create, comment on, merge, or close pull requests, and check merge readiness.",
	usage:   "gg pr <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List pull requests", usage: "gg pr list [flags]",
			flags:    []flagDef{stateFlag, limitFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			setPos: setState,
		},
		{
			name: "view", summary: "View one pull request", usage: "gg pr view <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr view <number>",
			setPos: setNumber,
		},
		{
			name: "create", summary: "Create a pull request", usage: "gg pr create [flags]",
			flags:    []flagDef{titleFlag, bodyFlag, baseFlag, headFlag, draftFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "comment", summary: "Comment on a pull request", usage: "gg pr comment <number> [flags]",
			flags:    []flagDef{bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr comment <number> --body <text>",
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.Body) == "" {
					return usageErr("usage: gg pr comment <number> --body <text>")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "comment list", summary: "List comments on a pull request", usage: "gg pr comment list <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr comment list <number>",
			setPos: setNumber,
		},
		{
			name: "comment edit", summary: "Edit a comment on a pull request", usage: "gg pr comment edit <number> <comment-id> [flags]",
			flags:    []flagDef{bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 2, maxPos: 2,
			posErr: "usage: gg pr comment edit <number> <comment-id> --body <text>",
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.Body) == "" {
					return usageErr("usage: gg pr comment edit <number> <comment-id> --body <text>")
				}
				req.Number = pos[0]
				req.CommentID = pos[1]
				return nil
			},
		},
		{
			name: "comment delete", summary: "Delete a comment on a pull request", usage: "gg pr comment delete <number> <comment-id> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 2, maxPos: 2,
			posErr: "usage: gg pr comment delete <number> <comment-id>",
			setPos: func(req *Request, pos []string) error {
				req.Number = pos[0]
				req.CommentID = pos[1]
				return nil
			},
		},
		{
			name: "status", summary: "Show merge readiness for one pull request", usage: "gg pr status <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr status <number>",
			setPos: setNumber,
		},
		{
			name: "ready", summary: "Mark a pull request as ready for review", usage: "gg pr ready <number> [flags]",
			flags:    []flagDef{undoFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr ready <number>",
			setPos: setNumber,
		},
		{
			name: "merge", summary: "Merge a pull request", usage: "gg pr merge <number> [flags]",
			flags:    []flagDef{mergeFlag, squashFlag, rebaseFlag, deleteBranchFlag, autoMergeFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr merge <number>",
			setPos: func(req *Request, pos []string) error {
				methods := 0
				for _, b := range []bool{req.Merge, req.Squash, req.Rebase} {
					if b {
						methods++
					}
				}
				if methods > 1 {
					return usageErr("--merge, --squash, --rebase are mutually exclusive; use at most one")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "close", summary: "Close a pull request", usage: "gg pr close <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr close <number>",
			setPos: setNumber,
		},
		{
			name: "reopen", summary: "Reopen a closed pull request", usage: "gg pr reopen <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr reopen <number>",
			setPos: setNumber,
		},
	},
}

var prListBuilders = providerBuilders{
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

var prViewBuilders = providerBuilders{
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

var prCreateBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		args = appendKV(args, "--base", c.req.Base)
		args = appendKV(args, "--head", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		args = appendKV(args, "--target-branch", c.req.Base)
		args = appendKV(args, "--source-branch", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create"}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		args = appendKV(args, "--base", c.req.Base)
		args = appendKV(args, "--head", c.req.Head)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		return args, nil
	},
}

// glabProjectPath는 glab api endpoint에 쓸 프로젝트 경로다. GitLab namespace는
// "grp/sub"처럼 /를 포함할 수 있으므로 경로 세그먼트 하나로 인코딩한다.
func glabProjectPath(r RepoURL) string {
	return url.PathEscape(r.Slug())
}

// prCommentBuilders는 pr comment(입력)의 builder다.
var prCommentBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "comment", c.req.Number, "--body", c.req.Body}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "note", c.req.Number, "--message", c.req.Body}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		// Gitea는 PR 댓글도 이슈 댓글 API를 공유하므로 tea comment로 댄다.
		return append([]string{"comment", c.req.Number, c.req.Body}, c.target...), nil
	},
}

// prCommentListBuilders는 PR 대화 댓글 목록을 JSON으로 조회한다. GitHub의 PR
// 대화 댓글은 이슈 댓글과 같은 endpoint를 공유하고, GitLab은 MR note API를 쓴다.
// gh/glab의 api 하위 명령은 --repo flag가 없으므로 호스트는 Env로 전달한다.
var prCommentListBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{"api", "repos/" + c.r.Slug() + "/issues/" + c.req.Number + "/comments"}
		return args, []string{"GH_HOST=" + c.r.Host}
	},
	glab: func(c invocationContext) (args, env []string) {
		args = []string{"api", "projects/" + glabProjectPath(c.r) + "/merge_requests/" + c.req.Number + "/notes"}
		return args, []string{"GITLAB_HOST=" + c.r.Host}
	},
}

var prCommentEditBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{"api", "-X", "PATCH", "repos/" + c.r.Slug() + "/issues/comments/" + c.req.CommentID, "-f", "body=" + c.req.Body}
		return args, []string{"GH_HOST=" + c.r.Host}
	},
	glab: func(c invocationContext) (args, env []string) {
		args = []string{"api", "-X", "PUT", "projects/" + glabProjectPath(c.r) + "/merge_requests/" + c.req.Number + "/notes/" + c.req.CommentID, "-f", "body=" + c.req.Body}
		return args, []string{"GITLAB_HOST=" + c.r.Host}
	},
}

var prCommentDeleteBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{"api", "-X", "DELETE", "repos/" + c.r.Slug() + "/issues/comments/" + c.req.CommentID}
		return args, []string{"GH_HOST=" + c.r.Host}
	},
	glab: func(c invocationContext) (args, env []string) {
		args = []string{"api", "-X", "DELETE", "projects/" + glabProjectPath(c.r) + "/merge_requests/" + c.req.Number + "/notes/" + c.req.CommentID}
		return args, []string{"GITLAB_HOST=" + c.r.Host}
	},
}

// prInvocationTable은 "pr <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// tea의 pr merge/status/ready와 pr comment list/edit/delete는 teaInvocation의
// 사전 가드에서 걸러지므로 여기에는 등록하지 않는다 — provider별 예외는 감추지
// 않고 그 함수에 명시적으로 남긴다.
var prInvocationTable = map[string]providerBuilders{
	"pr list": prListBuilders,
	"pr view": prViewBuilders,
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
	"pr close": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"pr", "close", c.req.Number}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "close", c.req.Number}, c.target...), nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "close", c.req.Number}, c.target...), nil
		},
	},
	"pr reopen": {
		gh: func(c invocationContext) (args, env []string) {
			return append([]string{"pr", "reopen", c.req.Number}, c.target...), nil
		},
		glab: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "reopen", c.req.Number}, c.target...), nil
		},
		tea: func(c invocationContext) (args, env []string) {
			return append([]string{c.res, "reopen", c.req.Number}, c.target...), nil
		},
	},
	"pr create": prCreateBuilders,

	"pr comment":        prCommentBuilders,
	"pr comment list":   prCommentListBuilders,
	"pr comment edit":   prCommentEditBuilders,
	"pr comment delete": prCommentDeleteBuilders,
}
