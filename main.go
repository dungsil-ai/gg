package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

var version = "dev"

const repositoryContextFlags = `  --repo <URL>      이 URL을 저장소 문맥으로 사용
  --remote <name>   이 Git remote를 저장소 문맥으로 사용`
const helpFlag = `  -h, --help        Show help`

const topLevelHelp = `gg sends common Git forge commands to gh, glab, or tea.

Usage:
  gg [flags] <command>
  gg <command> --help

Commands:
  repo       List, view, create, or clone repositories
  issue      List, view, or create issues
  pr         List, view, or create pull requests
  config     Provider 설정 관리
  pull       Run git pull
  push       Run git push
  version    Show gg version
  help       Show this help

Flags:
` + repositoryContextFlags + `
` + helpFlag + `
  --version         Show gg version`

const configHelp = `Manage Provider 설정 for self-hosted hosts.

Usage:
  gg config <command>

Commands:
  gg config list
  gg config set <host> <gh|glab|tea>
  gg config unset <host>`

const issueHelp = `List, view, or create issues.

Usage:
  gg issue <command> [flags]

Commands:
  list      List issues
  view      View one issue
  create    Create an issue

Flags:
` + repositoryContextFlags + `
` + helpFlag

const issueListHelp = `List issues.

Usage:
  gg issue list [flags]

Flags:
  --state <open|closed|all>   Filter by state
  --limit <N>                 Limit the result count
` + repositoryContextFlags + `
` + helpFlag

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
` + helpFlag

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
		fmt.Fprintln(os.Stdout, "gg "+version)
		return 0
	}
	req, err := ParseRequest(args)
	if err != nil {
		return fail(err)
	}
	inv, err := plan(req)
	if err != nil {
		return fail(err)
	}
	return execChild(inv)
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

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "gg:", err)
	var ue UsageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, topLevelHelp)
		return 2
	}
	return 1
}

// plan은 파싱된 요청을 자식 invocation으로 바꾼다.
func plan(req Request) (Invocation, error) {
	if req.Action == "pull" || req.Action == "push" {
		return Invocation{Bin: "git", Args: append([]string{req.Action}, req.GitArgs...)}, nil
	}
	if req.Action == "clone" && isHTTPURL(req.CloneURL) {
		if !req.AllowInsecureHTTP {
			return Invocation{}, usageErr("HTTP clone is blocked by default; use HTTPS or SSH (or pass --allow-insecure-http)")
		}
		fmt.Fprintln(os.Stderr, "gg: warning: allowing insecure HTTP clone; credentials or repository data may be exposed")
	}
	rawURL := req.RepoFlag
	if req.Action == "clone" {
		rawURL = req.CloneURL
	}
	if rawURL == "" {
		var err error
		rawURL, err = CurrentRemoteURL()
		if err != nil {
			return Invocation{}, err
		}
	}
	repo, err := ParseRepoURL(rawURL)
	if err != nil {
		return Invocation{}, err
	}
	cfg, err := LoadConfig()
	if err != nil {
		return Invocation{}, err
	}
	p, err := DetectProvider(repo.Host, &cfg, stdinIsTerminal())
	if err != nil {
		return Invocation{}, err
	}
	teaLogin := ""
	if p == Tea && req.Action != "clone" {
		if teaLogin = teaLoginName(repo.Host); teaLogin == "" {
			return Invocation{}, fmt.Errorf("no tea login for %s (run: tea login add)", repo.Host)
		}
	}
	return Translate(req, repo, p, teaLogin)
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
