package cli

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

type cloneHost struct {
	provider Provider
	baseURL  string
	login    string // tea는 같은 host에도 여러 login을 등록할 수 있다.
}

// resolveCloneShorthand는 clone 없이 읽기 전용 API로 모든 후보를 확인한다.
// 성공한 저장소가 하나일 때만 clone을 계획하며 현재 Git remote는 사용하지 않는다.
func resolveCloneShorthand(req Request) (executionPlan, error) {
	slug, err := cloneSlug(req.CloneURL)
	if err != nil {
		return executionPlan{}, err
	}
	cfg, err := LoadConfig()
	if err != nil {
		return executionPlan{}, err
	}
	hosts, diagnostics := registeredCloneHosts(&cfg)
	var matches []cloneHost
	seen := map[cloneHost]bool{}
	for _, host := range hosts {
		if isHTTPURL(host.baseURL) && !req.AllowInsecureHTTP {
			diagnostics = append(diagnostics, "HTTP clone is blocked by default for "+host.baseURL+"; use --allow-insecure-http to include it")
			continue
		}
		// GitHub와 Gitea는 중첩 namespace를 지원하지 않는다.
		if host.provider != GLab && strings.Count(slug, "/") != 1 {
			continue
		}
		key := cloneHost{provider: host.provider, baseURL: host.baseURL}
		if seen[key] || !cloneRepositoryExists(host, slug) {
			continue
		}
		seen[key] = true
		matches = append(matches, host)
	}
	if len(matches) == 0 {
		message := fmt.Sprintf("cannot find an accessible repository %q with the registered gh, glab, or tea CLIs; check CLI login/network access or use a full repository URL", slug)
		if len(diagnostics) != 0 {
			message += "\n" + strings.Join(diagnostics, "\n")
		}
		return executionPlan{}, errors.New(message)
	}
	if len(matches) > 1 {
		var choices []string
		for _, host := range matches {
			choices = append(choices, "  "+host.baseURL+"/"+slug+" ("+string(host.provider)+")")
		}
		return executionPlan{}, fmt.Errorf("multiple repositories match %q; use a full repository URL:\n%s", slug, strings.Join(choices, "\n"))
	}
	host := matches[0]
	req.CloneURL = host.baseURL + "/" + slug
	if isHTTPURL(req.CloneURL) {
		fmt.Fprintln(os.Stderr, "gg: warning: allowing insecure HTTP clone; credentials or repository data may be exposed")
	}
	repo, err := ParseRepoURL(req.CloneURL)
	if err != nil {
		return executionPlan{}, err
	}
	if host.provider == Tea {
		// tea는 전체 URL을 받으면 --login을 무시하고 host로 계정을 다시 고른다.
		// 조회한 계정을 유지하려면 host 없는 slug를 전달해야 한다.
		req.CloneURL = slug
	}
	inv, err := Translate(req, repo, host.provider, host.login)
	if err != nil {
		return executionPlan{}, err
	}
	if host.provider == Tea {
		// 조회에 성공한 계정을 clone에도 사용한다.
		inv.Args = append([]string{"clone", "--login", host.login}, inv.Args[1:]...)
	}
	return executionPlan{repo: repo, provider: host.provider, inv: inv}, nil
}

func cloneSlug(raw string) (string, error) {
	slug := strings.TrimSuffix(raw, ".git")
	parts := strings.Split(slug, "/")
	valid := len(parts) >= 2
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, "-") {
			valid = false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
				valid = false
			}
		}
	}
	if !valid {
		return "", usageErr("clone needs a repository URL or <namespace>/<name>")
	}
	return slug, nil
}

func cloneRepositoryExists(host cloneHost, slug string) bool {
	u, _ := url.Parse(host.baseURL)
	var out string
	var err error
	switch host.provider {
	case GH:
		out, err = runOut("gh", "api", "--hostname", u.Host, "repos/"+slug)
	case GLab:
		out, err = runOut("glab", "api", "--hostname", u.Host, "projects/"+url.PathEscape(slug))
	case Tea:
		out, err = runOut("tea", "api", "repos/"+slug, "--login", host.login, "--repo", slug)
	}
	// CLI의 성공 종료만으로는 오류 응답이나 프록시 로그인 페이지를 저장소로
	// 오인할 수 있다. 세 Forge가 공통으로 반환하는 저장소 id도 확인한다.
	var repo struct {
		ID int64 `json:"id"`
	}
	return err == nil && json.Unmarshal([]byte(out), &repo) == nil && repo.ID > 0
}

func registeredCloneHosts(cfg *Config) ([]cloneHost, []string) {
	var hosts []cloneHost
	var diagnostics []string
	seen := map[cloneHost]bool{}
	installed := map[Provider]bool{GH: hasBin(GH), GLab: hasBin(GLab), Tea: hasBin(Tea)}
	add := func(p Provider, base, login string) {
		u, err := url.Parse(strings.TrimRight(base, "/"))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || !validBareHostname(u.Hostname()) ||
			u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return
		}
		u.Host = strings.ToLower(u.Host)
		if fixed, ok := defaultProviders[u.Hostname()]; ok && fixed != p {
			return
		}
		host := cloneHost{provider: p, baseURL: u.String(), login: login}
		if installed[p] && !seen[host] {
			seen[host] = true
			hosts = append(hosts, host)
		}
	}
	// 기본 host와 gg 설정도 포함하므로 환경변수로만 인증한 CLI도 탐색한다.
	for host, p := range defaultProviders {
		if p != Tea {
			add(p, "https://"+host, "")
		}
	}
	for host, raw := range cfg.Hosts {
		if p := Provider(raw); p != Tea {
			add(p, "https://"+host, "")
		}
	}
	if installed[GH] {
		out, err := runOut("gh", "auth", "status", "--json", "hosts")
		var status struct {
			Hosts map[string]json.RawMessage `json:"hosts"`
		}
		if err != nil || json.Unmarshal([]byte(out), &status) != nil {
			diagnostics = append(diagnostics, "could not list gh hosts (run: gh auth status)")
		} else {
			for host := range status.Hosts {
				add(GH, "https://"+host, "")
			}
		}
		if host := os.Getenv("GH_HOST"); host != "" {
			add(GH, "https://"+host, "")
		}
	}
	if installed[GLab] {
		data, err := glabCloneConfig()
		var config struct {
			Host  string               `yaml:"host"`
			Hosts map[string]yaml.Node `yaml:"hosts"`
		}
		if err != nil || yaml.Unmarshal(data, &config) != nil {
			// 설정에는 token이 있을 수 있으므로 파싱 오류 원문을 출력하지 않는다.
			diagnostics = append(diagnostics, "could not read glab hosts (check the glab global config)")
		} else {
			for host := range config.Hosts {
				add(GLab, "https://"+host, "")
			}
			if config.Host != "" {
				add(GLab, "https://"+config.Host, "")
			}
		}
		for _, key := range []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST"} {
			if host := os.Getenv(key); host != "" {
				if !strings.Contains(host, "://") {
					host = "https://" + host
				}
				add(GLab, host, "")
				break
			}
		}
	}
	if installed[Tea] {
		out, err := runOut("tea", "logins", "list", "--output", "csv")
		reader := csv.NewReader(strings.NewReader(out))
		reader.FieldsPerRecord = -1
		rows, parseErr := reader.ReadAll()
		if err != nil || parseErr != nil {
			diagnostics = append(diagnostics, "could not list tea logins (gg requires classic tea v0.x)")
		} else {
			for i, row := range rows {
				if i > 0 && len(row) >= 2 && row[0] != "" {
					add(Tea, row[1], row[0])
				}
			}
		}
	}
	sort.Slice(hosts, func(i, j int) bool {
		if hosts[i].baseURL != hosts[j].baseURL {
			return hosts[i].baseURL < hosts[j].baseURL
		}
		if hosts[i].provider != hosts[j].provider {
			return hosts[i].provider < hosts[j].provider
		}
		return hosts[i].login < hosts[j].login
	})
	return hosts, diagnostics
}

// glab config path를 우선 사용하고, 구버전에서는 문서화된 전역 설정 경로를
// 순서대로 확인한다. 인증은 CLI에 맡기며 설정에서 host 목록만 추출한다.
func glabCloneConfig() ([]byte, error) {
	if path, err := runOut("glab", "config", "path"); err == nil && strings.TrimSpace(path) != "" {
		data, err := os.ReadFile(strings.TrimSpace(path))
		if !errors.Is(err, os.ErrNotExist) {
			return data, err
		}
	}
	if dir := os.Getenv("GLAB_CONFIG_DIR"); dir != "" {
		data, err := os.ReadFile(filepath.Join(dir, "config.yml"))
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return data, err
	}
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "glab-cli"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "glab-cli"))
	} else if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "glab-cli"))
		}
	} else if dir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(dir, "glab-cli"))
	}
	systemDirs := os.Getenv("XDG_CONFIG_DIRS")
	if systemDirs == "" && runtime.GOOS != "windows" {
		systemDirs = "/etc/xdg"
	}
	for _, dir := range filepath.SplitList(systemDirs) {
		dirs = append(dirs, filepath.Join(dir, "glab-cli"))
	}
	for _, dir := range dirs {
		data, err := os.ReadFile(filepath.Join(dir, "config.yml"))
		if !errors.Is(err, os.ErrNotExist) {
			return data, err
		}
	}
	return nil, nil
}
