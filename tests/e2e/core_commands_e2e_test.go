package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECoreForgeCommands(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct {
		name, provider, host, slug string
		args, want                 []string
	}{
		{name: "gh issue list", provider: "gh", host: "github.com", slug: "o/r", args: []string{"issue", "list", "--state", "all", "--limit", "5"}, want: []string{"gh", "issue", "list", "-R", "github.com/o/r", "--state", "all", "--limit", "5"}},
		{name: "gh pr view", provider: "gh", host: "github.com", slug: "o/r", args: []string{"pr", "view", "7"}, want: []string{"gh", "pr", "view", "7", "-R", "github.com/o/r"}},
		{name: "gh pr create", provider: "gh", host: "github.com", slug: "o/r", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}, want: []string{"gh", "pr", "create", "-R", "github.com/o/r", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}},
		{name: "gh repo list on GHE", provider: "gh", host: "ghe.corp.com", slug: "o/r", args: []string{"repo", "list", "--limit", "3"}, want: []string{"gh", "repo", "list", "--limit", "3"}},
		{name: "gh repo view", provider: "gh", host: "github.com", slug: "o/r", args: []string{"repo", "view"}, want: []string{"gh", "repo", "view", "https://github.com/o/r"}},
		{name: "gh repo create", provider: "gh", host: "ghe.corp.com", slug: "o/r", args: []string{"repo", "create", "--public", "--description", "d"}, want: []string{"gh", "repo", "create", "o/r", "--public", "--description", "d"}},
		{name: "glab issue list closed", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"issue", "list", "--state", "closed", "--limit", "5"}, want: []string{"glab", "issue", "list", "--repo", "https://git.example.com/grp/sub/p", "--closed", "--per-page", "5"}},
		{name: "glab pr list all", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"pr", "list", "--state", "all"}, want: []string{"glab", "mr", "list", "--repo", "https://git.example.com/grp/sub/p", "--all"}},
		{name: "glab pr list open은 flag 없음", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"pr", "list", "--state", "open"}, want: []string{"glab", "mr", "list", "--repo", "https://git.example.com/grp/sub/p"}},
		{name: "glab pr create", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}, want: []string{"glab", "mr", "create", "--repo", "https://git.example.com/grp/sub/p", "--title", "t", "--description", "b", "--target-branch", "main", "--source-branch", "f", "--draft"}},
		{name: "glab issue view", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"issue", "view", "9"}, want: []string{"glab", "issue", "view", "9", "--repo", "https://git.example.com/grp/sub/p"}},
		{name: "glab repo list", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"repo", "list", "--limit", "7"}, want: []string{"glab", "repo", "list", "--per-page", "7"}},
		{name: "glab repo create", provider: "glab", host: "git.example.com", slug: "grp/sub/p", args: []string{"repo", "create", "--private", "--description", "d"}, want: []string{"glab", "repo", "create", "grp/sub/p", "--private", "--description", "d"}},
		{name: "tea issue list", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"issue", "list", "--state", "all", "--limit", "5"}, want: []string{"tea", "issues", "list", "--login", "corp", "--repo", "o/r", "--state", "all", "--limit", "5"}},
		{name: "tea pr view", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"pr", "view", "3"}, want: []string{"tea", "pulls", "3", "--login", "corp", "--repo", "o/r"}},
		{name: "tea pr create", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}, want: []string{"tea", "pulls", "create", "--login", "corp", "--repo", "o/r", "--title", "t", "--description", "b", "--base", "main", "--head", "f", "--draft"}},
		{name: "tea repo view", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"repo", "view"}, want: []string{"tea", "repos", "o/r", "--login", "corp"}},
		{name: "tea repo create public은 --private 없음", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"repo", "create", "--public", "--description", "d"}, want: []string{"tea", "repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--description", "d"}},
		{name: "tea repo create private", provider: "tea", host: "gitea.example.com", slug: "o/r", args: []string{"repo", "create", "--private"}, want: []string{"tea", "repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--private"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, home, work := t.TempDir(), t.TempDir(), t.TempDir()
			log := filepath.Join(t.TempDir(), "calls.log")
			config := fakeCLIConfig{LogFile: log, Stdout: "operation output\n", Rules: []fakeCLIRule{
				{Args: []string{"auth", "status", "--hostname", tc.host, "--json", "hosts"}, Stdout: `{"hosts":{"` + tc.host + `":[{"active":true}]}}`},
				{Args: []string{"auth", "status", "--hostname", tc.host}},
				{Args: []string{"logins", "list", "--output", "csv"}, Stdout: "Name,URL\ncorp,https://" + tc.host + "\n"},
			}}
			writeFakeCLI(t, dir, tc.provider, config)
			if tc.host != "github.com" {
				if out, code := runGGWithPath(t, bin, work, home, dir, "config", "set", tc.host, tc.provider); code != 0 {
					t.Fatalf("config: %d: %s", code, out)
				}
			}
			args := append([]string{"--repo", "https://" + tc.host + "/" + tc.slug}, tc.args...)
			out, code := runGGWithPath(t, bin, work, home, dir, args...)
			if code != 0 || out != "operation output\n" {
				t.Fatalf("exit %d: %q", code, out)
			}
			calls := readLogLines(t, log)
			if len(calls) == 0 || calls[len(calls)-1] != wantCall(tc.want...) {
				t.Fatalf("calls = %s; want %s", strings.Join(calls, "\n"), wantCall(tc.want...))
			}
		})
	}
}
