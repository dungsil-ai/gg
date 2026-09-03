package cli

import (
	"errors"
	"slices"
)

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
	// label은 아직 glab만 중계한다. builder가 등록될 때까지 여기서 미지원을 확정한다.
	if req.Resource == "label" {
		return Invocation{}, usageErr("label " + req.Action + " is not supported for gh")
	}
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
	if req.Resource == "pr" && req.Action == "merge" {
		return Invocation{}, errors.New("pr merge is not supported for tea")
	}
	if req.Resource == "pr" && (req.Action == "status" || req.Action == "ready") {
		return Invocation{}, usageErr("pr " + req.Action + " is not supported for tea")
	}
	if req.Resource == "label" {
		return Invocation{}, usageErr("label " + req.Action + " is not supported for tea")
	}
	if req.Resource == "release" {
		return Invocation{}, usageErr("release is not supported for tea")
	}
	var res string
	switch req.Resource {
	case "repo":
		res = "repos"
	case "issue":
		res = "issues"
	case "pr":
		res = "pulls"
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
