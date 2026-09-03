package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Provider string

const (
	GH   Provider = "gh"
	GLab Provider = "glab"
	Tea  Provider = "tea"
)

// RepoURL은 저장소 URL에서 얻은 forge 좌표다. Host는 lowercase, port 없음.
type RepoURL struct {
	Host  string
	Owner string // GitLab namespace는 "grp/sub"처럼 /를 포함할 수 있다
	Name  string // 끝의 .git 제거
}

func (r RepoURL) Slug() string  { return r.Owner + "/" + r.Name }
func (r RepoURL) HTTPS() string { return "https://" + r.Host + "/" + r.Slug() }

// ParseRepoURL은 https/ssh/SCP 형식 저장소 URL을 파싱한다.
func ParseRepoURL(raw string) (RepoURL, error) {
	raw = strings.TrimSpace(raw)
	var host, path string
	switch {
	case strings.HasPrefix(raw, "https://"), strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "ssh://"):
		u, err := url.Parse(raw)
		if err != nil {
			return RepoURL{}, fmt.Errorf("invalid repository URL %q: %w", raw, err)
		}
		host = u.Hostname()
		path = u.Path
	case raw == "", strings.Contains(raw, "://"):
		return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
	default: // SCP 형식: [user@]host:owner/repo
		i := strings.Index(raw, ":")
		if i < 0 {
			return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
		}
		host = raw[:i]
		if at := strings.Index(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		path = raw[i+1:]
	}
	host = strings.ToLower(host)
	if !validBareHostname(host) {
		return RepoURL{}, fmt.Errorf("repository URL %q has an invalid host", raw)
	}
	var segs []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if len(segs) < 2 {
		return RepoURL{}, fmt.Errorf("repository URL %q must look like host/owner/repo", raw)
	}
	for _, s := range segs {
		if strings.HasPrefix(s, "-") {
			return RepoURL{}, fmt.Errorf("repository URL %q has a path segment starting with %q", raw, "-")
		}
	}
	name := strings.TrimSuffix(segs[len(segs)-1], ".git")
	if name == "" {
		return RepoURL{}, fmt.Errorf("repository URL %q has an empty repository name", raw)
	}
	return RepoURL{Host: host, Owner: strings.Join(segs[:len(segs)-1], "/"), Name: name}, nil
}

// 테스트 교체점: 자식 실행/탐색/입력.
var (
	runOut = func(name string, args ...string) (string, error) {
		out, err := exec.Command(name, args...).Output()
		return strings.TrimSpace(string(out)), err
	}
	lookPath           = exec.LookPath
	stdin    io.Reader = os.Stdin
)

// CurrentRemoteURL: branch upstream → origin → 유일 remote 순서.
func CurrentRemoteURL() (string, error) {
	// commit이 없는 저장소에서는 rev-parse가 실패한다. upstream 단계만
	// 건너뛰고 origin/유일 remote fallback은 계속 시도한다.
	if branch, err := runOut("git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil && branch != "" {
		if remote, _ := runOut("git", "config", "--get", "branch."+branch+".remote"); remote != "" {
			if u, ok := gitRemoteURL(remote); ok {
				return u, nil
			}
		}
	}
	if u, ok := gitRemoteURL("origin"); ok {
		return u, nil
	}
	names, err := gitRemoteNames()
	if err != nil {
		return "", errors.New("not a git repository (use --repo <URL>)")
	}
	if len(names) == 1 {
		if u, ok := gitRemoteURL(names[0]); ok {
			return u, nil
		}
	}
	return "", errors.New("cannot pick a remote (use --repo <URL>)")
}

func RemoteURL(name string) (string, error) {
	if u, ok := gitRemoteURL(name); ok {
		return u, nil
	}
	names, _ := gitRemoteNames()
	available := "none"
	if len(names) != 0 {
		available = strings.Join(names, ", ")
	}
	return "", fmt.Errorf("remote %q not found (available remotes: %s)", name, available)
}

func gitRemoteURL(name string) (string, bool) {
	u, err := runOut("git", "remote", "get-url", name)
	return u, err == nil && u != ""
}

func gitRemoteNames() ([]string, error) {
	remotes, err := runOut("git", "remote")
	return strings.Fields(remotes), err
}

var defaultProviders = map[string]Provider{
	"github.com": GH,
	"gitlab.com": GLab,
	"gitea.com":  Tea,
}

func hasBin(p Provider) bool { _, err := lookPath(string(p)); return err == nil }

func ghHasLogin(host string) bool {
	out, err := runOut("gh", "auth", "status", "--hostname", host, "--json", "hosts")
	if err != nil {
		return false
	}
	var v struct {
		Hosts map[string]json.RawMessage `json:"hosts"`
	}
	if json.Unmarshal([]byte(out), &v) != nil {
		return false
	}
	for h := range v.Hosts {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func glabHasLogin(host string) bool {
	_, err := runOut("glab", "auth", "status", "--hostname", host)
	return err == nil
}

// teaLoginName은 host에 맞는 tea login 이름을 돌려준다. 없으면 "".
func teaLoginName(host string) string {
	out, err := runOut("tea", "logins", "list", "--output", "json")
	if err != nil {
		return ""
	}
	var logins []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if json.Unmarshal([]byte(out), &logins) != nil {
		return ""
	}
	for _, l := range logins {
		if u, err := url.Parse(l.URL); err == nil && strings.EqualFold(u.Hostname(), host) {
			return l.Name
		}
	}
	return ""
}

func hasLogin(p Provider, host string) bool {
	switch p {
	case GH:
		return ghHasLogin(host)
	case GLab:
		return glabHasLogin(host)
	case Tea:
		return teaLoginName(host) != ""
	}
	return false
}

// DetectProvider: 기본 domain → 저장값 → 로그인 후보 → 선택/오류.
func DetectProvider(host string, cfg *Config, interactive bool) (Provider, error) {
	if p, ok := defaultProviders[host]; ok {
		return p, nil
	}
	if saved, err := ParseProvider(cfg.Hosts[host]); err == nil {
		if hasBin(saved) && hasLogin(saved, host) {
			return saved, nil
		}
	}
	var cands []Provider
	for _, p := range []Provider{GH, GLab, Tea} {
		if hasBin(p) && hasLogin(p, host) {
			cands = append(cands, p)
		}
	}
	switch len(cands) {
	case 0:
		return "", fmt.Errorf(
			"no logged-in CLI for %s\nrun one of:\n  gh auth login --hostname %s\n  glab auth login --hostname %s\n  tea login add",
			host, host, host)
	case 1:
		return cands[0], nil
	}
	if !interactive {
		names := make([]string, len(cands))
		for i, p := range cands {
			names[i] = string(p)
		}
		return "", fmt.Errorf(
			"multiple providers match %s: %s\nset it once in %s:\n  {\"hosts\": {\"%s\": \"%s\"}}",
			host, strings.Join(names, ", "), ConfigPath(), host, names[0])
	}
	p, err := chooseProvider(host, cands)
	if err != nil {
		return "", err
	}
	if err := SaveProvider(host, p); err != nil {
		return "", err
	}
	return p, nil
}

func chooseProvider(host string, cands []Provider) (Provider, error) {
	fmt.Fprintf(os.Stderr, "Multiple providers match %s:\n", host)
	for i, p := range cands {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, p)
	}
	fmt.Fprint(os.Stderr, "Choose provider: ")
	var n int
	if _, err := fmt.Fscanln(stdin, &n); err != nil || n < 1 || n > len(cands) {
		return "", errors.New("invalid choice")
	}
	return cands[n-1], nil
}

func stdinIsTerminal() bool {
	f, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
