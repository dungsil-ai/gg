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
	"strings"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		fmt.Fprintln(os.Stdout, topLevelHelp())
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
	if len(args) == 1 && (args[0] == "-verison" || args[0] == "-v") {
		printAllVersions()
		return 0
	}
	req, err := ParseRequest(args)
	if err != nil {
		return fail(err)
	}
	if req.Help {
		rd := commandDefs[req.Resource]
		fmt.Fprintln(os.Stdout, renderActionHelp(rd, rd.action(req.Action)))
		return 0
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
	if req.Resource == "pr" && req.Action == "status" && !req.Explain {
		ep, err := resolvePlan(req)
		if err != nil {
			return fail(err)
		}
		return runPRStatus(ep)
	}
	inv, err := plan(req)
	if err != nil {
		return fail(err)
	}
	return execChild(inv)
}

func printAllVersions() {
	fmt.Fprintln(os.Stdout, "gg "+getVersion())
	for _, name := range []string{"git", "gh", "glab", "tea"} {
		out, err := runOut(name, "--version")
		if err != nil || out == "" {
			continue
		}
		out = strings.ReplaceAll(out, "\r\n", "\n")
		fmt.Fprintln(os.Stdout, out)
	}
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

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "gg:", err)
	var ue UsageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, topLevelHelp())
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
	if req.Resource == "repo" {
		if action := commandDefs["repo"].action(req.Action); action != nil && action.passthrough {
			argCount := len(req.GitArgs) + 1
			if req.Action == "commit" {
				argCount++
			}
			args := make([]string, 0, argCount)
			args = append(args, req.Action)
			if req.Action == "commit" {
				args = append(args, "--no-gpg-sign")
			}
			args = append(args, req.GitArgs...)
			return executionPlan{inv: Invocation{Bin: "git", Args: args}}, nil
		}
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
	// pr status와 pr ready는 provider를 고른 뒤 미지원을 확정하므로 tea login을 묻지 않는다.
	unsupportedTeaPRAction := req.Resource == "pr" && (req.Action == "status" || req.Action == "ready")
	if p == Tea && req.Action != "clone" && !unsupportedTeaPRAction {
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
	// fork 전에 부모 SIGINT를 Notify로 소비한다. Go runtime은 Notify된 신호를
	// caught 상태로 처리하므로 fork 자식에서 exec 시 자동으로 default로 복원된다.
	// Ignore와 달리 Notify는 exec된 자식에 SIG_IGN을 상속시키지 않는다.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	err = cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return childExitCode(ee)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gg:", err)
		return 1
	}
	return 0
}
