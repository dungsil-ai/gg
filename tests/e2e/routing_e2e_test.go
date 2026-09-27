package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ERepositoryURLFormats(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct{ url, provider, host, slug string }{
		{"https://github.com/cli/cli.git", "gh", "github.com", "cli/cli"},
		{"https://GitHub.com/cli/cli", "gh", "github.com", "cli/cli"},
		{"git@github.com:cli/cli.git", "gh", "github.com", "cli/cli"},
		{"https://gitlab.com/grp/sub/proj.git", "glab", "gitlab.com", "grp/sub/proj"},
		{"http://git.example.com/o/r", "glab", "git.example.com", "o/r"},
		{"ssh://git@git.example.com:2222/o/r.git", "glab", "git.example.com", "o/r"},
		{"git@gitea.com:gitea/tea", "tea", "gitea.com", "gitea/tea"},
	} {
		t.Run(tc.url, func(t *testing.T) {
			dir, home, work := t.TempDir(), t.TempDir(), t.TempDir()
			log := filepath.Join(t.TempDir(), "calls.log")
			writeFakeCLI(t, dir, tc.provider, fakeCLIConfig{LogFile: log, TeaLogin: tc.provider == "tea"})
			if tc.host == "git.example.com" {
				if out, code := runGGWithHome(t, bin, dir, work, home, "config", "set", tc.host, tc.provider); code != 0 {
					t.Fatalf("config: %d: %s", code, out)
				}
			}
			out, code := runGGWithHome(t, bin, dir, work, home, "--repo", tc.url, "issue", "view", "42")
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			want := wantCall("gh", "issue", "view", "42", "-R", tc.host+"/"+tc.slug)
			if tc.provider == "glab" {
				want = wantCall("glab", "issue", "view", "42", "--repo", "https://"+tc.host+"/"+tc.slug)
			}
			if tc.provider == "tea" {
				want = wantCall("tea", "issues", "42", "--login", "pub", "--repo", tc.slug)
			}
			if calls := readLog(t, log); !strings.HasSuffix(calls, want) {
				t.Fatalf("calls = %s; want final %s", calls, want)
			}
		})
	}
}

func TestE2EInvalidRepositoryURLsDoNotCallProviders(t *testing.T) {
	bin := buildGG(t)
	for _, url := range []string{"ftp://x.com/a/b", "https://github.com/onlyowner", "https:///a/b", "plain-text"} {
		t.Run(url, func(t *testing.T) {
			dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			for _, name := range []string{"gh", "glab", "tea"} {
				writeFakeBin(t, dir, name, log)
			}
			out, code := runGG(t, bin, dir, t.TempDir(), "--repo", url, "issue", "list")
			if code != 1 || !strings.Contains(out, "gg:") {
				t.Fatalf("exit %d: %s", code, out)
			}
			if calls := readLog(t, log); calls != "" {
				t.Fatalf("invalid URL called providers: %s", calls)
			}
		})
	}
}

func TestE2EProviderDiscovery(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct {
		name        string
		gh, glab    bool
		saved, want string
		code        int
	}{
		{"no login", false, false, "", "auth login", 1},
		{"ambiguous", true, true, "", "multiple providers match", 1},
		{"saved provider", true, true, "glab", "Provider: glab", 0},
		{"stale provider", true, false, "glab", "Provider: gh", 0},
		{"single provider", true, false, "", "Provider: gh", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, home, work := t.TempDir(), t.TempDir(), t.TempDir()
			log := filepath.Join(t.TempDir(), "calls.log")
			if tc.gh {
				writeFakeCLI(t, dir, "gh", fakeCLIConfig{LogFile: log, Stdout: `{"hosts":{"git.example.com":[{"active":true,"login":"test"}]}}`})
			}
			if tc.glab {
				writeFakeCLI(t, dir, "glab", fakeCLIConfig{LogFile: log})
			}
			if tc.saved != "" {
				if out, code := runGGWithPath(t, bin, work, home, dir, "config", "set", "git.example.com", tc.saved); code != 0 {
					t.Fatalf("config: %d: %s", code, out)
				}
			}
			out, code := runGGWithPath(t, bin, work, home, dir, "--repo", "https://git.example.com/o/r", "--explain", "issue", "list")
			if code != tc.code || !strings.Contains(out, tc.want) {
				t.Fatalf("exit %d: %s; want %d: %s", code, out, tc.code, tc.want)
			}
			if strings.Contains(readLog(t, log), `"issue","list"`) {
				t.Fatal("discovery/explain executed the requested mutation")
			}
		})
	}
}

func TestE2ETeaVersionAndLoginFormats(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct {
		name, login, version, want string
		code                       int
	}{
		{"quoted", "\"Name\",\"URL\"\n\"pub\",\"https://gitea.com\"\n", "tea version 0.9.2", "", 0},
		{"unquoted", "Name,URL\npub,https://gitea.com\n", "tea version 0.16.0", "", 0},
		{"missing", "Name,URL\nother,https://other.example\n", "tea version 0.16.0", "no tea login", 1},
		{"v1", "", "tea version v1.3.0", "tea v1.x is not supported", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			writeFakeCLI(t, dir, "tea", fakeCLIConfig{LogFile: log, Rules: []fakeCLIRule{
				{Args: []string{"logins", "list", "--output", "csv"}, Stdout: tc.login},
				{Args: []string{"--version"}, Stdout: tc.version},
			}})
			out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, "--repo", "https://gitea.com/o/r", "issue", "list")
			if code != tc.code || !strings.Contains(out, tc.want) {
				t.Fatalf("exit %d: %s", code, out)
			}
			called := strings.Contains(readLog(t, log), `"issues","list"`)
			if called != (tc.code == 0) {
				t.Fatalf("unexpected issue dispatch: %s", readLog(t, log))
			}
		})
	}
}

func TestE2EChildExitAndMissingExecutable(t *testing.T) {
	bin := buildGG(t)
	for _, code := range []int{0, 1, 42, 127} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			dir := t.TempDir()
			writeFakeCLI(t, dir, "git", fakeCLIConfig{Stdout: "child stdout", Stderr: "child stderr", ExitCode: code})
			stdout, stderr, got := runGGStreamsWithFake(t, bin, dir, t.TempDir(), "status")
			if got != code || stdout != "child stdout" || stderr != "child stderr" {
				t.Fatalf("exit %d, stdout %q, stderr %q", got, stdout, stderr)
			}
		})
	}
	out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), t.TempDir(), "status")
	if code != 127 || !strings.Contains(out, "git") {
		t.Fatalf("missing executable: exit %d: %s", code, out)
	}
}

func TestE2EAuthStatusActiveAccount(t *testing.T) {
	bin, dir := buildGG(t), t.TempDir()
	writeFakeCLI(t, dir, "gh", fakeCLIConfig{Stdout: `{"hosts":{"GitHub.COM":[{"login":"old"},{"login":"active","active":true}]}}`})
	out, code := runGGWithPath(t, bin, t.TempDir(), t.TempDir(), dir, "auth", "status")
	if code != 0 || !strings.Contains(out, "active") || strings.Contains(out, "old") {
		t.Fatalf("exit %d: %s", code, out)
	}
}
