package cli

import "slices"

// Invocation은 실행할 자식 process다.
type Invocation struct {
	Bin  string
	Args []string
	Env  []string // os.Environ()에 덧붙일 KEY=VALUE
}

func Translate(req Request, r RepoURL, p Provider, teaLogin string) (Invocation, error) {
	switch p {
	case GH:
		return ghInvocation(req, r)
	case GLab:
		return glabInvocation(req, r)
	case Tea:
		return teaInvocation(req, r, teaLogin)
	}
	return Invocation{}, usageErr("unknown provider " + string(p))
}

func appendKV(args []string, flag, val string) []string {
	if val == "" {
		return args
	}
	return append(args, flag, val)
}

func visFlag(req Request) string {
	if req.Private {
		return "--private"
	}
	return "--public"
}

// invocationContext는 provider별 arg-builder가 쓸 값을 담는다. res/target/auth는
// provider마다 다르게 계산되며, 각 builder는 자신에게 필요한 필드만 읽는다.
type invocationContext struct {
	req    Request
	r      RepoURL
	res    string   // 이 provider에서의 resource 이름(예: gh/tea "pr", glab "mr")
	target []string // issue/pr 계열에서 저장소를 가리키는 인자
	auth   []string // tea 전용 --login 인자; gh/glab은 nil
}

// invocationBuilder는 하나의 provider가 "resource action" 하나를 Args/Env로
// 옮긴다. env가 nil이면 추가 환경변수가 없다는 뜻이다.
type invocationBuilder func(c invocationContext) (args, env []string)

// providerBuilders는 "resource action" 하나에 대한 gh/glab/tea builder를 모은다.
// 필드가 nil인 provider는 그 action을 지원하지 않는다.
type providerBuilders struct {
	gh, glab, tea invocationBuilder
}

// dispatch는 provider 이름으로 invocationTable을 조회해 Invocation을 만든다.
// 대상 action이 없거나 이 provider용 builder가 없으면 3개 provider가 공유하는
// "does not support" 오류를 낸다.
func dispatch(provider string, c invocationContext) (Invocation, error) {
	entry := invocationTable[c.req.Resource+" "+c.req.Action]
	var build invocationBuilder
	switch provider {
	case "gh":
		build = entry.gh
	case "glab":
		build = entry.glab
	case "tea":
		build = entry.tea
	}
	if build == nil {
		return Invocation{}, usageErr(c.req.Resource + " does not support " + c.req.Action)
	}
	args, env := build(c)
	return Invocation{Bin: provider, Args: args, Env: env}, nil
}

func ghInvocation(req Request, r RepoURL) (Invocation, error) {
	c := invocationContext{
		req:    req,
		r:      r,
		res:    req.Resource,
		target: []string{"-R", r.Host + "/" + r.Slug()},
	}
	return dispatch("gh", c)
}

func glabInvocation(req Request, r RepoURL) (Invocation, error) {
	// glab release create에는 draft와 prerelease 개념이 없다. 조용히 무시하면
	// 사용자가 예상과 다르게 공개 release를 만들게 되므로 사용법 오류로 막는다.
	if req.Resource == "release" && req.Action == "create" && (req.Draft || req.Prerelease) {
		return Invocation{}, usageErr("release create --draft/--prerelease is not supported for glab")
	}
	// 관계 등록과 PR 잠금·해제는 GitHub 고유 기능이다. builder가 등록될 때까지
	// 여기서 미지원을 확정한다.
	if req.Resource == "issue" && ghOnlyIssueActions[req.Action] {
		return Invocation{}, usageErr("issue " + req.Action + " is not supported for glab")
	}
	if req.Resource == "pr" && (req.Action == "lock" || req.Action == "unlock") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for glab")
	}
	// glab에는 MR 단위 체크 조회 명령이 없다. pipeline 조회는 gg ci list를 쓴다.
	if req.Resource == "pr" && req.Action == "checks" {
		return Invocation{}, usageErr("pr checks is not supported for glab")
	}
	// glab mr rebase는 동작이 다른 별도 명령이라 update-branch 표면으로
	// 중계하지 않는다.
	if req.Resource == "pr" && req.Action == "update-branch" {
		return Invocation{}, usageErr("pr update-branch is not supported for glab")
	}
	// glab에는 changes 요청과 리뷰 본문 달기 명령이 없어 approve만 중계한다.
	if req.Resource == "pr" && req.Action == "review" && (req.RequestChanges || req.ReviewComment) {
		return Invocation{}, usageErr("pr review --request-changes/--comment is not supported for glab")
	}
	res := req.Resource
	if res == "pr" {
		res = "mr"
	}
	c := invocationContext{
		req:    req,
		r:      r,
		res:    res,
		target: []string{"--repo", r.HTTPS()},
	}
	return dispatch("glab", c)
}

func teaInvocation(req Request, r RepoURL, login string) (Invocation, error) {
	// tea merge에는 자동 병합과 branch 삭제 개념이 없다. 조용히 무시하면
	// 사용자 예상과 다르게 동작하므로 사용법 오류로 막는다.
	if req.Resource == "pr" && req.Action == "merge" && (req.Auto || req.DeleteBranch) {
		return Invocation{}, usageErr("pr merge --auto/--delete-branch is not supported for tea")
	}
	// tea는 PR status/ready/diff 명령이 없다.
	if req.Resource == "pr" && (req.Action == "status" || req.Action == "ready" || req.Action == "diff") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
	// tea는 PR 잠금·해제 명령이 없다.
	if req.Resource == "pr" && (req.Action == "lock" || req.Action == "unlock") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
	// tea에는 PR 단위 체크 조회 명령이 없다.
	if req.Resource == "pr" && req.Action == "checks" {
		return Invocation{}, usageErr("pr checks is not supported for tea")
	}
	// tea에는 PR branch 갱신 명령이 없다.
	if req.Resource == "pr" && req.Action == "update-branch" {
		return Invocation{}, usageErr("pr update-branch is not supported for tea")
	}
	// tea에는 리뷰 본문만 다는 명령이 없어 approve와 reject만 중계한다.
	if req.Resource == "pr" && req.Action == "review" && req.ReviewComment {
		return Invocation{}, usageErr("pr review --comment is not supported for tea")
	}
	// tea label edit·delete는 label 이름이 아니라 numeric label id(--id)를
	// 요구하고 clone은 명령 자체가 없어 중계하지 않는다. list·create는 이름
	// 기반 표면이라 중계한다.
	if req.Resource == "label" && (req.Action == "edit" || req.Action == "delete" || req.Action == "clone") {
		return Invocation{}, usageErr("label " + req.Action + " is not supported for tea")
	}
	if req.Resource == "release" {
		return Invocation{}, usageErr("release is not supported for tea")
	}
	// tea는 PR 댓글 추가만 지원하고 목록/수정/삭제 명령이 없다.
	if req.Resource == "pr" && (req.Action == "comment list" || req.Action == "comment edit" || req.Action == "comment delete") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
	// tea는 이슈 댓글 추가만 지원하고 목록/수정/삭제와 이슈 수정 명령이 없다.
	if req.Resource == "issue" && (req.Action == "edit" ||
		req.Action == "comment list" || req.Action == "comment edit" || req.Action == "comment delete") {
		return Invocation{}, usageErr("issue " + req.Action + " is not supported for tea")
	}
	// tea에는 이슈 삭제 하위 명령이 없다.
	if req.Resource == "issue" && req.Action == "delete" {
		return Invocation{}, usageErr("issue delete is not supported for tea")
	}
	if req.Resource == "issue" && ghOnlyIssueActions[req.Action] {
		return Invocation{}, usageErr("issue " + req.Action + " is not supported for tea")
	}
	if req.Resource == "ci" {
		return Invocation{}, usageErr("ci is not supported for tea")
	}
	var res string
	switch req.Resource {
	case "repo":
		res = "repos"
	case "issue":
		res = "issues"
	case "pr":
		res = "pulls"
	case "label":
		res = "labels"
	}
	auth := []string{"--login", login}
	c := invocationContext{
		req:    req,
		r:      r,
		res:    res,
		target: slices.Concat(auth, []string{"--repo", r.Slug()}),
		auth:   auth,
	}
	return dispatch("tea", c)
}
