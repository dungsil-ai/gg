package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// issueResourceDef는 "issue" 최상위 명령의 정의다: list, view, create, comment,
// close, reopen과 관계 등록(sub-issue, blocked-by, type).
var issueResourceDef = &resourceDef{
	name:    "issue",
	summary: "List, view, create, comment, close, reopen, or link issues",
	desc:    "List, view, create, comment, close, reopen, or link issues.",
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
		{
			name: "sub-issue", summary: "Register an issue as a sub-issue of a parent issue",
			usage:    "gg issue sub-issue <number> --parent <parent> [flags]",
			flags:    []flagDef{parentFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue sub-issue <number> --parent <parent>",
			setPos: setSubIssue,
		},
		{
			name: "blocked-by", summary: "Register a blocked-by issue dependency",
			usage:    "gg issue blocked-by <number> --blocker <blocker> [flags]",
			flags:    []flagDef{blockerFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue blocked-by <number> --blocker <blocker>",
			setPos: setBlockedBy,
		},
		{
			name: "type", summary: "Set the issue type",
			usage:    "gg issue type <number> --name <name> [flags]",
			flags:    []flagDef{issueTypeNameFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg issue type <number> --name <name>",
			setPos: setIssueType,
		},
	},
}

// ghOnlyIssueActions는 현재 gh(GitHub REST API)만 중계하는 issue action이다.
// glab/tea 사전 가드와 tea login 건너뛰기가 이 집합을 함께 본다.
var ghOnlyIssueActions = map[string]bool{
	"sub-issue":  true,
	"blocked-by": true,
	"type":       true,
}

// isIssueNumber는 값이 issue 번호 형태(숫자만)인지 본다. 관계 API는 번호를 endpoint
// 경로에 넣으므로 URL 같은 다른 표기는 여기서 걸러낸다.
func isIssueNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// setSubIssue/setBlockedBy/setIssueType은 관계 등록 action의 positional 저장과
// flag 검증을 한다. 두 번호가 같은 자기 참조는 API에 보내기 전에 걸러낸다.
func setSubIssue(req *Request, pos []string) error {
	if !isIssueNumber(pos[0]) || !isIssueNumber(req.Parent) {
		return usageErr("usage: gg issue sub-issue <number> --parent <parent>")
	}
	if req.Parent == pos[0] {
		return usageErr("--parent must be a different issue than " + pos[0])
	}
	req.Number = pos[0]
	return nil
}

func setBlockedBy(req *Request, pos []string) error {
	if !isIssueNumber(pos[0]) || !isIssueNumber(req.Blocker) {
		return usageErr("usage: gg issue blocked-by <number> --blocker <blocker>")
	}
	if req.Blocker == pos[0] {
		return usageErr("--blocker must be a different issue than " + pos[0])
	}
	req.Number = pos[0]
	return nil
}

func setIssueType(req *Request, pos []string) error {
	if !isIssueNumber(pos[0]) || strings.TrimSpace(req.IssueType) == "" {
		return usageErr("usage: gg issue type <number> --name <name>")
	}
	req.Number = pos[0]
	return nil
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

// issueSubIssueBuilders와 issueBlockedByBuilders는 gh api로 GitHub 관계 endpoint를
// 호출한다. 두 endpoint 모두 body로 issue 번호가 아니라 numeric database id를
// 요구하므로, resolvePlan이 ghIssueDatabaseID로 조회해 req.RelatedID에 채운 값을
// 쓴다. gh api는 -R이 없어 --hostname으로 저장소 문맥 host를 전달한다.
var issueSubIssueBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{"api", "--method", "POST",
			"repos/" + c.r.Slug() + "/issues/" + c.req.Parent + "/sub_issues",
			"--hostname", c.r.Host}, "-F", "sub_issue_id="+c.req.RelatedID)
		return args, nil
	},
}

var issueBlockedByBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{"api", "--method", "POST",
			"repos/" + c.r.Slug() + "/issues/" + c.req.Number + "/dependencies/blocked_by",
			"--hostname", c.r.Host}, "-F", "issue_id="+c.req.RelatedID)
		return args, nil
	},
}

var issueTypeBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{"api", "--method", "PATCH",
			"repos/" + c.r.Slug() + "/issues/" + c.req.Number,
			"--hostname", c.r.Host}, "-F", "type="+c.req.IssueType)
		return args, nil
	},
}

// ghIssueDatabaseID는 issue 번호를 GitHub numeric database id로 바꾼다.
// sub-issue와 blocked-by API가 body로 요구하는 값이다.
func ghIssueDatabaseID(r RepoURL, number string) (string, error) {
	out, err := runOut("gh", "api", "--hostname", r.Host,
		"repos/"+r.Slug()+"/issues/"+number, "--jq", ".id")
	if err != nil {
		return "", fmt.Errorf("cannot resolve issue %s id: %s", number, childErrorDetail(err))
	}
	if out == "" {
		return "", fmt.Errorf("cannot resolve issue %s id: empty response", number)
	}
	return out, nil
}

// childErrorDetail은 자식 실패의 stderr나 오류 문자열을 한 줄로 돌려준다.
func childErrorDetail(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if s := strings.TrimSpace(string(ee.Stderr)); s != "" {
			return s
		}
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
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
	"issue close":      issueCloseReopenBuilders,
	"issue reopen":     issueCloseReopenBuilders,
	"issue create":     issueCreateBuilders,
	"issue sub-issue":  issueSubIssueBuilders,
	"issue blocked-by": issueBlockedByBuilders,
	"issue type":       issueTypeBuilders,
}
