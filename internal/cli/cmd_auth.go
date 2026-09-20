package cli

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
)

// authResourceDef는 "auth" 최상위 명령의 정의다: status와 provider CLI 계정 작업
// 릴레이(login, logout, refresh, setup-git, switch, token). auth는 forge 명령이
// 아니므로 저장소 문맥 flag와 --explain이 없다(ADR 0006). 릴레이 action은 모든
// 인자를 provider CLI에 그대로 전달하는 passthrough이고, --provider flag로
// gh(기본값) 외에 glab을 고를 수 있다(ADR 0008).
var authResourceDef = &resourceDef{
	name:    "auth",
	summary: "Show provider CLI login status, and relay GitHub CLI account operations",
	desc:    "Show provider CLI login status, and relay GitHub CLI account operations. Use --provider glab on a relay action to target the GitLab CLI instead.",
	usage:   "gg auth <command>",
	actions: append([]actionDef{
		{
			name: "status", summary: "Show login status for each host", usage: "gg auth status",
			posErr: "usage: gg auth status",
		},
	}, authRelayActions()...),
}

// authRelayActionNames는 gh CLI 계정 작업을 그대로 전달하는 auth action이다.
var authRelayActionNames = []string{"login", "logout", "refresh", "setup-git", "switch", "token"}

// glabAuthActions은 --provider glab을 받아들일 수 있는 auth action이다. glab에
// 없는 gh 하위 명령(refresh·setup-git·switch)은 거부한다.
var glabAuthActions = []string{"login", "logout", "token"}

func authRelayActions() []actionDef {
	actions := make([]actionDef, len(authRelayActionNames))
	for i, name := range authRelayActionNames {
		actions[i] = actionDef{
			name:        name,
			summary:     "Run gh auth " + name + " (all args pass through; --provider glab targets the GitLab CLI)",
			usage:       "gg auth " + name + " [args...]",
			passthrough: true, maxPos: -1,
		}
	}
	return actions
}

// isAuthRelayAction은 이름이 gh 릴레이 auth action인지 본다. --help 전달 등
// passthrough 예외 처리에서 쓴다.
func isAuthRelayAction(name string) bool {
	for _, n := range authRelayActionNames {
		if n == name {
			return true
		}
	}
	return false
}

// splitAuthProvider는 auth 릴레이 인자에서 gg가 해석하는 --provider 값을 뽑아
// 내고 나머지를 돌려준다. "--provider glab"과 "--provider=glab" 형태를 모두
// 받고 위치는 action 뒤 어디든 좋다. 기본값은 gh다.
func splitAuthProvider(args []string) (rest []string, p Provider, err error) {
	p = GH
	rest = make([]string, 0, len(args))
	take := func(raw string) error {
		got, vErr := validateAuthProvider(raw)
		if vErr != nil {
			return vErr
		}
		p = got
		return nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--provider" {
			if i+1 >= len(args) {
				return nil, "", usageErr("--provider needs a value")
			}
			i++
			if err := take(args[i]); err != nil {
				return nil, "", err
			}
			continue
		}
		if name, value, hasValue := strings.Cut(a, "="); hasValue && name == "--provider" {
			if err := take(value); err != nil {
				return nil, "", err
			}
			continue
		}
		rest = append(rest, a)
	}
	return rest, p, nil
}

func validateAuthProvider(raw string) (Provider, error) {
	switch p := Provider(raw); p {
	case GH, GLab:
		return p, nil
	case Tea:
		return "", usageErr("--provider tea is not supported; tea logins is the tea CLI's own local auth store (ADR 0007)")
	}
	return "", usageErr("--provider must be gh or glab")
}

// authRelayInvocation은 auth 릴레이 action을 provider CLI 호출로 옮긴다. action
// 뒤의 모든 인자(--help 포함)는 검사 없이 그대로 전달되고, --provider만 gg가
// 뽑아 내어 대상 CLI를 고른다. glab이 없는 하위 명령(refresh·setup-git·switch)은
// 거부한다(ADR 0008).
func authRelayInvocation(req Request) (Invocation, error) {
	rest, p, err := splitAuthProvider(req.GitArgs)
	if err != nil {
		return Invocation{}, err
	}
	switch p {
	case GLab:
		if !slices.Contains(glabAuthActions, req.Action) {
			return Invocation{}, usageErr("glab auth " + req.Action + " is not supported; glab has login, logout, status, and token")
		}
		return Invocation{Bin: "glab", Args: append([]string{"auth", req.Action}, rest...)}, nil
	default:
		return Invocation{Bin: "gh", Args: append([]string{"auth", req.Action}, rest...)}, nil
	}
}

func runAuth(req Request) error {
	switch req.Action {
	case "status":
		return runAuthStatus()
	}
	return usageErr("auth does not support " + req.Action)
}

// runAuthStatus는 Provider 설정의 host와 기본 domain의 로그인 상태를 한 표로
// 조회한다(ADR 0006). 표 조회 자체가 실패할 때만 오류를 내고, 행별 로그인
// 여부는 결과 값이다.
func runAuthStatus() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "HOST\tPROVIDER\tLOGIN")
	for _, row := range authStatusRows(&cfg) {
		fmt.Fprintf(w, "%s\t%s\t%s\n", row.Host, row.Provider, row.Login)
	}
	return w.Flush()
}

// authStatusRow는 auth status 표의 한 행이다.
type authStatusRow struct {
	Host     string
	Provider Provider
	Login    string // 로그인 이름 | yes | no | no cli
}

// authStatusRows는 기본 domain과 Provider 설정의 host를 host 기준으로 정렬해
// 조회한다. 기본 domain이 항상 목록에 있으므로 빈 표는 없다.
func authStatusRows(cfg *Config) []authStatusRow {
	providers := make(map[string]Provider, len(defaultProviders)+len(cfg.Hosts))
	for host, p := range defaultProviders {
		providers[host] = p
	}
	for host, raw := range cfg.Hosts {
		if p, err := ParseProvider(raw); err == nil {
			providers[host] = p
		}
	}
	hosts := make([]string, 0, len(providers))
	for host := range providers {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	rows := make([]authStatusRow, 0, len(hosts))
	for _, host := range hosts {
		p := providers[host]
		rows = append(rows, authStatusRow{Host: host, Provider: p, Login: authLoginValue(p, host)})
	}
	return rows
}

// authLoginValue는 한 host의 LOGIN 열 값을 조회한다. provider CLI 미설치는
// hasBin으로 먼저 걸러 내어 조회를 시도하지 않는다.
func authLoginValue(p Provider, host string) string {
	if !hasBin(p) {
		return "no cli"
	}
	switch p {
	case GH:
		if name, ok := ghLoginName(host); ok {
			if name != "" {
				return name
			}
			return "yes"
		}
	case GLab:
		if glabHasLogin(host) {
			return "yes"
		}
	case Tea:
		if name := teaLoginName(host); name != "" {
			return name
		}
	}
	return "no"
}
