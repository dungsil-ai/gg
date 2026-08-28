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

const usage = `usage:
  gg [--repo <URL>] <command>

repository (repo 생략 가능):
  gg list [--limit N]
  gg view
  gg --repo <URL> create (--public|--private) [--description TEXT]
  gg clone <URL> [DIR] [--allow-insecure-http]
  gg pull [git-args...]
  gg push [git-args...]

issue / pr:
  gg issue|pr list [--state open|closed|all] [--limit N]
  gg issue|pr view <number>
  gg issue create [--title TEXT] [--body TEXT]
  gg pr create [--title TEXT] [--body TEXT] [--base BRANCH] [--head BRANCH] [--draft]`

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
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

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "gg:", err)
	var ue UsageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, usage)
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
