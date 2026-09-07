package cli

import (
	"net/url"
	"strings"
)

// prResourceDef는 "pr" 최상위 명령의 정의다: list, view, checkout, create,
// edit, comment(하위 list/edit/delete), status, checks, ready, merge, close,
// reopen, diff, lock, unlock, review, update-branch. alias: mr
// (command_registry.go의 commandAliases에서 연결).
var prResourceDef = &resourceDef{
	name:    "pr",
	summary: "List, view, check out, create, edit, comment on, diff, check CI, merge, lock, review, update, or close pull requests, and check merge readiness (alias: mr)",
	desc:    "List, view, check out, create, edit, comment on, diff, check CI, merge, lock, review, update, or close pull requests, and check merge readiness.",
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
			name: "checkout", summary: "Check out a pull request locally", usage: "gg pr checkout <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr checkout <number>",
			setPos: setNumber,
		},
		{
			name: "checks", summary: "Show CI status of a pull request (GitHub only)", usage: "gg pr checks <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr checks <number>",
			setPos: setNumber,
		},
		{
			name: "update-branch", summary: "Update a pull request branch with its base branch (GitHub only)",
			usage:    "gg pr update-branch <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr update-branch <number>",
			setPos: setNumber,
		},
		{
			name: "diff", summary: "Show changes of a pull request", usage: "gg pr diff <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr diff <number>",
			setPos: setNumber,
		},
		{
			name: "create", summary: "Create a pull request", usage: "gg pr create [flags]",
			flags:    []flagDef{titleFlag, bodyFlag, baseFlag, headFlag, draftFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "edit", summary: "Edit a pull request title or body", usage: "gg pr edit <number> [flags]",
			flags:    []flagDef{titleFlag, bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr edit <number>",
			setPos: func(req *Request, pos []string) error {
				if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Body) == "" {
					return usageErr("pr edit needs --title or --body")
				}
				req.Number = pos[0]
				return nil
			},
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
			name: "delete", summary: "Delete a pull request (GitLab only)", usage: "gg pr delete <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr delete <number>",
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
		{
			name: "lock", summary: "Lock a pull request conversation (GitHub only)", usage: "gg pr lock <number> [flags]",
			flags:    []flagDef{lockReasonFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr lock <number>",
			setPos: func(req *Request, pos []string) error {
				switch req.Reason {
				case "", "off_topic", "resolved", "spam", "too_heated":
				default:
					return usageErr("--reason must be off_topic, resolved, spam, or too_heated")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "unlock", summary: "Unlock a locked pull request conversation (GitHub only)", usage: "gg pr unlock <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr unlock <number>",
			setPos: setNumber,
		},
		{
			name: "rebase", summary: "Rebase a pull request against its base branch (GitLab only)",
			usage:    "gg pr rebase <number> [flags]",
			flags:    []flagDef{rebaseSkipCIFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr rebase <number>",
			setPos: setNumber,
		},
		{
			name: "review", summary: "Review a pull request (approve, request changes, or comment)",
			usage:    "gg pr review <number> (--approve | --request-changes | --comment) [flags]",
			flags:    []flagDef{approveFlag, requestChangesFlag, reviewCommentFlag, bodyFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr review <number> (--approve | --request-changes | --comment)",
			setPos: func(req *Request, pos []string) error {
				kinds := 0
				for _, b := range []bool{req.Approve, req.RequestChanges, req.ReviewComment} {
					if b {
						kinds++
					}
				}
				if kinds != 1 {
					return usageErr("usage: gg pr review <number> (--approve | --request-changes | --comment)")
				}
				if req.RequestChanges && strings.TrimSpace(req.Body) == "" {
					return usageErr("pr review --request-changes needs --body <text>")
				}
				req.Number = pos[0]
				return nil
			},
		},
		{
			name: "approvers", summary: "List approvers of a pull request (GitLab only)", usage: "gg pr approvers <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr approvers <number>",
			setPos: setNumber,
		},
		{
			name: "revoke", summary: "Revoke your approval of a pull request (GitLab only)", usage: "gg pr revoke <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr revoke <number>",
			setPos: setNumber,
		},
		{
			name: "todo", summary: "Add a pull request to your To-Do List (GitLab only)", usage: "gg pr todo <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr todo <number>",
			setPos: setNumber,
		},
		{
			name: "subscribe", summary: "Subscribe to a pull request (GitLab only)", usage: "gg pr subscribe <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr subscribe <number>",
			setPos: setNumber,
		},
		{
			name: "unsubscribe", summary: "Unsubscribe from a pull request (GitLab only)", usage: "gg pr unsubscribe <number> [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg pr unsubscribe <number>",
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

// prCheckoutBuilders는 PR을 로컬 작업 트리로 check out한다. 3개 provider 모두
// 번호 하나를 받는 같은 모양의 표면이다. flag 없이 번호만 중계한다.
var prCheckoutBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "checkout", c.req.Number}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "checkout", c.req.Number}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "checkout", c.req.Number}, c.target...), nil
	},
}

// prChecksBuilders는 PR의 CI 체크 상태를 보여준다. gh 전용 기능이라 gh
// builder만 등록하고, glab·tea는 glabInvocation·teaInvocation 사전 가드와
// run.go의 tea login 건너뛰기 목록이 미지원을 확정한다 (tea login을 묻기
// 전에 거부된다). GitLab MR의 pipeline은 gg ci list로도 볼 수 있다.
var prChecksBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "checks", c.req.Number}, c.target...), nil
	},
}

// prUpdateBranchBuilders는 PR branch를 base branch 최신 상태로 갱신한다. gh
// 전용 기능이라 gh builder만 등록하고, glab·tea는 사전 가드와 run.go의 tea
// login 건너뛰기 목록이 미지원을 확정한다 (glab mr rebase는 동작이 다른 별도
// 명령이라 같은 표면으로 중계하지 않는다).
var prUpdateBranchBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "update-branch", c.req.Number}, c.target...), nil
	},
}

// prDiffBuilders는 PR의 변경 내용을 diff로 보여준다. tea에는 diff 하위 명령이
// 없어 builder를 등록하지 않는다 — teaInvocation의 사전 가드가 tea login을
// 묻기 전에 미지원을 확정한다.
var prDiffBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "diff", c.req.Number}, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "diff", c.req.Number}, c.target...), nil
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

// prEditBuilders는 PR 제목·본문 수정을 중계한다. glab은 mr update를, tea는
// pulls edit을 쓰며 둘 다 본문 flag로 --description을 쓴다. glab mr update는
// pr ready의 draft·ready 전환도 담당하지만 flag 없이 부르면 제목·본문 수정이다.
var prEditBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "edit", c.req.Number}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--body", c.req.Body)
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "update", c.req.Number}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		return args, nil
	},
	tea: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "edit", c.req.Number}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--description", c.req.Body)
		return args, nil
	},
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

// prCommentListBuilders는 PR 대화 댓글 목록을 조회한다. GitHub의 PR 대화 댓글은
// 이슈 댓글과 같은 endpoint를 공유하고, GitLab은 MR note API를 쓴다.
// gh/glab의 api 하위 명령은 --repo flag가 없으므로 호스트는 Env로 전달한다.
// tea는 Gitea에서 PR 댓글이 이슈 댓글 API를 공유하므로 comments list로 조회한다.
var prCommentListBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{"api", "repos/" + c.r.Slug() + "/issues/" + c.req.Number + "/comments"}
		return args, []string{"GH_HOST=" + c.r.Host}
	},
	glab: func(c invocationContext) (args, env []string) {
		args = []string{"api", "projects/" + glabProjectPath(c.r) + "/merge_requests/" + c.req.Number + "/notes"}
		return args, []string{"GITLAB_HOST=" + c.r.Host}
	},
	tea: func(c invocationContext) (args, env []string) {
		return append([]string{"comments", "list", c.req.Number}, c.target...), nil
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

// prLockBuilders는 PR 대화 잠금을 중계한다. gh 전용 기능이라 gh builder만
// 등록하고, glab·tea는 glabInvocation·teaInvocation 사전 가드와 run.go의
// tea login 건너뛰기 목록이 미지원을 확정한다 (tea login을 묻기 전에 거부된다).
// --reason은 gh의 잠금 사유 enum이라 gg에서 미리 검증한다.
var prLockBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "lock", c.req.Number}, c.target...)
		return appendKV(args, "--reason", c.req.Reason), nil
	},
}

var prUnlockBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "unlock", c.req.Number}, c.target...), nil
	},
}

// prReviewBuilders는 PR 리뷰를 중계한다. approve는 세 provider 모두 같은 개념이
// 있다(gh pr review --approve, glab mr approve, tea pulls approve). request
// changes는 tea pulls reject로 중계하고 glab에는 명령이 없다. 리뷰 본문 달기
// (--comment)는 gh 전용이다. glab·tea의 미지원 조합은 glabInvocation·
// teaInvocation 사전 가드에서 확정한다.
var prReviewBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "review", c.req.Number}, c.target...)
		switch {
		case c.req.Approve:
			args = append(args, "--approve")
		case c.req.RequestChanges:
			args = append(args, "--request-changes")
			args = appendKV(args, "--body", c.req.Body)
		case c.req.ReviewComment:
			args = append(args, "--comment")
			args = appendKV(args, "--body", c.req.Body)
		}
		return args, nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "approve", c.req.Number}, c.target...), nil
	},
	tea: func(c invocationContext) (args, env []string) {
		if c.req.RequestChanges {
			// tea pulls reject는 사유를 positional 필수 인자로 받는다.
			return append([]string{c.res, "reject", c.req.Number, c.req.Body}, c.target...), nil
		}
		return append([]string{c.res, "approve", c.req.Number}, c.target...), nil
	},
}

// prDeleteBuilders는 PR을 삭제한다. glab 전용 기능이라 glab builder만 등록한다
// — gh와 tea는 dispatch의 builder 부재 오류로 걸러지며, tea는 run.go의 tea
// login 건너뛰기 목록이 login을 묻기 전에 미지원을 확정한다. glab에는 확인
// flag가 없어 gg의 --yes는 전달하지 않는다 (issue delete와 같은 원칙).
var prDeleteBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "delete", c.req.Number}, c.target...), nil
	},
}

// prApproversBuilders와 prRevokeBuilders는 MR 승인자 조회와 승인 철회를
// 중계한다. glab 전용 기능이라 glab builder만 등록한다 — gh와 tea는 dispatch의
// builder 부재 오류로 걸러지며, tea는 run.go의 tea login 건너뛰기 목록이 login을
// 묻기 전에 미지원을 확정한다.
var prApproversBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "approvers", c.req.Number}, c.target...), nil
	},
}

var prRevokeBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "revoke", c.req.Number}, c.target...), nil
	},
}

// prTodoBuilders는 MR을 To-Do List에 추가한다. glab 전용 기능이라 glab
// builder만 등록한다 — gh와 tea는 dispatch의 builder 부재 오류로 걸러지며,
// tea는 run.go의 tea login 건너뛰기 목록이 login을 묻기 전에 미지원을 확정한다.
var prTodoBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "todo", c.req.Number}, c.target...), nil
	},
}

// prSubscribeBuilders와 prUnsubscribeBuilders는 MR 알림 구독을 관리한다.
// glab 전용 기능이라 glab builder만 등록한다 — gh와 tea는 dispatch의 builder
// 부재 오류로 걸러지며, tea는 run.go의 tea login 건너뛰기 목록이 login을 묻기
// 전에 미지원을 확정한다.
var prSubscribeBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "subscribe", c.req.Number}, c.target...), nil
	},
}

var prUnsubscribeBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		return append([]string{c.res, "unsubscribe", c.req.Number}, c.target...), nil
	},
}

// prRebaseBuilders는 MR source branch를 target branch 기준으로 리베이스한다.
// glab 전용 기능이라 glab builder만 등록한다 — gh와 tea는 dispatch의 builder
// 부재 오류로 걸러지며, tea는 run.go의 tea login 건너뛰기 목록이 login을 묻기
// 전에 미지원을 확정한다.
var prRebaseBuilders = providerBuilders{
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "rebase", c.req.Number}, c.target...)
		if c.req.SkipCI {
			args = append(args, "--skip-ci")
		}
		return args, nil
	},
}

// prInvocationTable은 "pr <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// tea의 pr status/ready와 pr comment list/edit/delete는 teaInvocation의
// 사전 가드에서 걸러지므로 여기에는 등록하지 않는다 — provider별 예외는 감추지
// 않고 그 함수에 명시적으로 남긴다.
var prInvocationTable = map[string]providerBuilders{
	"pr list":          prListBuilders,
	"pr view":          prViewBuilders,
	"pr checkout":      prCheckoutBuilders,
	"pr checks":        prChecksBuilders,
	"pr update-branch": prUpdateBranchBuilders,
	"pr diff":          prDiffBuilders,
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
		// tea는 --style merge|rebase|squash로 병합 방식을 고른다. 방식 flag가
		// 없으면 tea의 기본 방식(merge)을 따른다.
		tea: func(c invocationContext) (args, env []string) {
			args = []string{c.res, "merge", c.req.Number}
			switch {
			case c.req.Squash:
				args = append(args, "--style", "squash")
			case c.req.Rebase:
				args = append(args, "--style", "rebase")
			case c.req.Merge:
				args = append(args, "--style", "merge")
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
	"pr create":      prCreateBuilders,
	"pr edit":        prEditBuilders,
	"pr lock":        prLockBuilders,
	"pr unlock":      prUnlockBuilders,
	"pr delete":      prDeleteBuilders,
	"pr rebase":      prRebaseBuilders,
	"pr approvers":   prApproversBuilders,
	"pr revoke":      prRevokeBuilders,
	"pr todo":        prTodoBuilders,
	"pr subscribe":   prSubscribeBuilders,
	"pr unsubscribe": prUnsubscribeBuilders,
	"pr review":      prReviewBuilders,

	"pr comment":        prCommentBuilders,
	"pr comment list":   prCommentListBuilders,
	"pr comment edit":   prCommentEditBuilders,
	"pr comment delete": prCommentDeleteBuilders,
}
