package cli

import (
	"fmt"
)

// runAPIRelay는 "gg api [args...]" 원시 passthrough를 처리한다(ADR 0007).
// 저장소 문맥으로 provider를 고르고, host는 env(GH_HOST/GITLAB_HOST)로 주입
// 하며, args는 검사 없이 provider의 api 하위 명령에 전달된다. tea는 --login과
// --repo를 붙인다. 명령 앞에는 --repo/--remote만 올 수 있고, "api" 뒤의 인자는
// --repo/--remote라는 이름이라도 api 인자로 relay한다.
func runAPIRelay(args []string) int {
	idx, ok := apiCommandIndex(args)
	if !ok {
		return fail(usageErr("unknown command api"))
	}
	contextArgs := args[:idx]
	relayArgs := args[idx+1:]

	var repoFlag, remoteFlag string
	for i := 0; i < len(contextArgs); i += 2 {
		if i+1 >= len(contextArgs) {
			return fail(usageErr("missing value for " + contextArgs[i]))
		}
		switch contextArgs[i] {
		case "--repo":
			repoFlag = contextArgs[i+1]
		case "--remote":
			remoteFlag = contextArgs[i+1]
		default:
			return fail(usageErr("unknown flag " + contextArgs[i]))
		}
	}

	rawURL, err := apiRelayRawURL(repoFlag, remoteFlag)
	if err != nil {
		return fail(err)
	}
	repo, err := ParseRepoURL(rawURL)
	if err != nil {
		return fail(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		return fail(err)
	}
	p, err := DetectProvider(repo.Host, &cfg, stdinIsTerminal())
	if err != nil {
		return fail(err)
	}

	var inv Invocation
	switch p {
	case GH:
		var env []string
		if repo.Host != "github.com" {
			env = []string{"GH_HOST=" + repo.Host}
		}
		inv = Invocation{Bin: "gh", Args: append([]string{"api"}, relayArgs...), Env: env}
	case GLab:
		var env []string
		if repo.Host != "gitlab.com" {
			env = []string{"GITLAB_HOST=" + repo.Host}
		}
		inv = Invocation{Bin: "glab", Args: append([]string{"api"}, relayArgs...), Env: env}
	case Tea:
		login := teaLoginName(repo.Host)
		if login == "" {
			return fail(fmt.Errorf("no tea login for %s (run: tea login add)", repo.Host))
		}
		inv = Invocation{Bin: "tea", Args: append([]string{"api", "--login", login, "--repo", repo.Slug()}, relayArgs...)}
	}
	return execChild(inv)
}

// apiCommandIndex는 명령 앞의 --repo/--remote를 건너뛴 뒤 "api"가 오는지
// 확인하고 그 위치를 돌려준다. 다른 토큰이 앞에 오면 ok는 false다.
func apiCommandIndex(args []string) (int, bool) {
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--repo", "--remote":
			if i+1 >= len(args) {
				return 0, false
			}
			i += 2
		default:
			if args[i] == "api" {
				return i, true
			}
			return 0, false
		}
	}
	return 0, false
}

// apiRelayRawURL은 api 릴레이의 저장소 문맥을 얻는다. context flag가 없으면
// 현재 remote를 따른다.
func apiRelayRawURL(repoFlag, remoteFlag string) (string, error) {
	if repoFlag != "" {
		return repoFlag, nil
	}
	if remoteFlag != "" {
		return RemoteURL(remoteFlag)
	}
	return CurrentRemoteURL()
}
