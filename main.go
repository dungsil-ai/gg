package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"sort"
	"strings"
	"text/tabwriter"
)

var version = "dev"

const repositoryContextFlags = `  --repo <URL>      이 URL을 저장소 문맥으로 사용
  --remote <name>   이 Git remote를 저장소 문맥으로 사용`
const topLevelHelpFlag = `  -h, --help        Show top-level help`
const nestedHelpFlag = `  --help            Show help`

const topLevelHelp = `gg sends common Git forge commands to gh, glab, or tea.

Usage:
  gg [flags] <command>
  gg config --help
  gg issue --help
  gg issue list --help
  gg pr create --help

Commands:
  repo       List, view, create, or clone repositories
  issue      List, view, or create issues
  pr         List, view, or create pull requests
  config     Provider 설정 관리
  commit     Run git commit without signing
  pull       Run git pull
  push       Run git push
  version    Show gg version
  help       Show this help

Flags:
` + repositoryContextFlags + `
  --explain         선택한 저장소 문맥, Provider, 실행할 CLI를 설명
` + topLevelHelpFlag + `
  --version         Show gg version`

const configHelp = `Manage Provider 설정 for self-hosted hosts.

Usage:
  gg config <command>

Commands:
  gg config list
  gg config set <host> <gh|glab|tea>
  gg config unset <host>

Flags:
` + nestedHelpFlag

const issueHelp = `List, view, or create issues.

Usage:
  gg issue <command> [flags]

Commands:
  list      List issues
  view      View one issue
  create    Create an issue

Flags:
` + repositoryContextFlags + `
  --explain         선택한 저장소 문맥, Provider, 실행할 CLI를 설명
` + nestedHelpFlag

const issueListHelp = `List issues.

Usage:
  gg issue list [flags]

Flags:
  --state <open|closed|all>   Filter by state
  --limit <N>                 Limit the result count
` + repositoryContextFlags + `
  --explain                   선택한 저장소 문맥, Provider, 실행할 CLI를 설명
` + nestedHelpFlag

const prCreateHelp = `Create a pull request.

Usage:
  gg pr create [flags]

Flags:
  --title <text>     Set the title
  --body <text>      Set the body
  --base <branch>    Set the base branch
  --head <branch>    Set the head branch
  --draft            Create a draft pull request
` + repositoryContextFlags + `
  --explain          선택한 저장소 문맥, Provider, 실행할 CLI를 설명
` + nestedHelpFlag

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		fmt.Fprintln(os.Stdout, topLevelHelp)
		return 0
	}
	if help, ok := nestedHelp(args); ok {
		fmt.Fprintln(os.Stdout, help)
		return 0
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version") {
		fmt.Fprintln(os.Stdout, "gg "+getVersion())
		return 0
	}
	req, err := ParseRequest(args)
	if err != nil {
		return fail(err)
	}
	if req.Resource == "config" {
		if err := runConfig(req); err != nil {
			return fail(err)
		}
		return 0
	}
	if req.Explain {
		ep, err := resolvePlan(req)
		if err != nil {
			return fail(err)
		}
		explain(os.Stdout, ep)
		return 0
	}
	inv, err := plan(req)
	if err != nil {
		return fail(err)
	}
	return execChild(inv)
}
func getVersion() string {
	if version != "dev" && version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs" {
				return "dev"
			}
		}
		v := info.Main.Version
		if v != "" && v != "(devel)" && !strings.HasPrefix(v, "v0.0.0-") {
			return v
		}
	}
	return "dev"
}

func nestedHelp(args []string) (string, bool) {
	if len(args) < 2 || args[len(args)-1] != "--help" {
		return "", false
	}
	switch strings.Join(args, " ") {
	case "config --help":
		return configHelp, true
	case "issue --help":
		return issueHelp, true
	case "issue list --help":
		return issueListHelp, true
	case "pr create --help":
		return prCreateHelp, true
	}
	return "", false
}

func runConfig(req Request) error {
	switch req.Action {
	case "list":
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		if len(cfg.Hosts) == 0 {
			fmt.Fprintln(os.Stdout, "No provider settings.")
			return nil
		}
		hosts := make([]string, 0, len(cfg.Hosts))
		for host := range cfg.Hosts {
			hosts = append(hosts, host)
		}
		sort.Strings(hosts)
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "HOST\tPROVIDER")
		for _, host := range hosts {
			fmt.Fprintf(w, "%s\t%s\n", host, cfg.Hosts[host])
		}
		return w.Flush()
	case "set":
		host, fixed, err := normalizeProviderSettingHost(req.ConfigHost)
		if err != nil {
			return err
		}
		if fixed {
			return usageErr(host + " is a default domain and cannot be changed")
		}
		provider, err := ParseProvider(req.ConfigProvider)
		if err != nil {
			return err
		}
		return SaveProvider(host, provider)
	case "unset":
		host, fixed, err := normalizeProviderSettingHost(req.ConfigHost)
		if err != nil {
			return err
		}
		if fixed {
			return nil
		}
		return UnsetProvider(host)
	}
	return usageErr("config does not support " + req.Action)
}

func normalizeProviderSettingHost(input string) (host string, fixed bool, err error) {
	host, err = NormalizeConfigHost(input)
	if err != nil {
		return "", false, err
	}
	_, fixed = defaultProviders[host]
	return host, fixed, nil
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "gg:", err)
	var ue UsageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, topLevelHelp)
		return 2
	}
	return 1
}

type executionPlan struct {
	repo     RepoURL
	provider Provider
	inv      Invocation
}

func resolvePlan(req Request) (executionPlan, error) {
	if req.Action == "commit" {
		return executionPlan{inv: Invocation{Bin: "git", Args: append([]string{"commit", "--no-gpg-sign"}, req.GitArgs...)}}, nil
	}
	if req.Action == "pull" || req.Action == "push" {
		return executionPlan{inv: Invocation{Bin: "git", Args: append([]string{req.Action}, req.GitArgs...)}}, nil
	}
	if req.Action == "clone" && isHTTPURL(req.CloneURL) {
		if !req.AllowInsecureHTTP {
			return executionPlan{}, usageErr("HTTP clone is blocked by default; use HTTPS or SSH (or pass --allow-insecure-http)")
		}
		fmt.Fprintln(os.Stderr, "gg: warning: allowing insecure HTTP clone; credentials or repository data may be exposed")
	}
	rawURL := req.RepoFlag
	if req.Action == "clone" {
		rawURL = req.CloneURL
	}
	if rawURL == "" {
		var err error
		if req.RemoteFlag != "" {
			rawURL, err = RemoteURL(req.RemoteFlag)
		} else {
			rawURL, err = CurrentRemoteURL()
		}
		if err != nil {
			return executionPlan{}, err
		}
	}
	repo, err := ParseRepoURL(rawURL)
	if err != nil {
		return executionPlan{}, err
	}
	cfg, err := LoadConfig()
	if err != nil {
		return executionPlan{}, err
	}
	p, err := DetectProvider(repo.Host, &cfg, stdinIsTerminal())
	if err != nil {
		return executionPlan{}, err
	}
	teaLogin := ""
	if p == Tea && req.Action != "clone" {
		if teaLogin = teaLoginName(repo.Host); teaLogin == "" {
			return executionPlan{}, fmt.Errorf("no tea login for %s (run: tea login add)", repo.Host)
		}
	}
	inv, err := Translate(req, repo, p, teaLogin)
	if err != nil {
		return executionPlan{}, err
	}
	return executionPlan{repo: repo, provider: p, inv: inv}, nil
}

func plan(req Request) (Invocation, error) {
	ep, err := resolvePlan(req)
	return ep.inv, err
}

func explain(w io.Writer, ep executionPlan) {
	fmt.Fprintf(w, "저장소 문맥: %s\n", ep.repo.HTTPS())
	fmt.Fprintf(w, "Provider: %s\n", ep.provider)
	fmt.Fprintf(w, "CLI: %s\n", ep.inv.Bin)
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && strings.EqualFold(u.Scheme, "http")
}

func execChild(inv Invocation) int {
	path, err := lookPath(inv.Bin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gg: %s is not installed or not on PATH\n", inv.Bin)
		return 127
	}
	cmd := exec.Command(path, inv.Args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), inv.Env...)
	// Ctrl+C는 자식에게 간다. 부모는 자식 exit code만 보고한다.
	signal.Ignore(os.Interrupt)
	err = cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gg:", err)
		return 1
	}
	return 0
}
