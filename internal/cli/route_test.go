package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		in    string
		want  RepoURL
		isErr bool
	}{
		{in: "https://github.com/cli/cli.git", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "https://GitHub.com/cli/cli", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "http://git.example.com/o/r", want: RepoURL{Host: "git.example.com", Owner: "o", Name: "r"}},
		{in: "https://gitlab.com/grp/sub/proj.git", want: RepoURL{Host: "gitlab.com", Owner: "grp/sub", Name: "proj"}},
		{in: "ssh://git@git.example.com:2222/o/r.git", want: RepoURL{Host: "git.example.com", Owner: "o", Name: "r"}},
		{in: "git@github.com:cli/cli.git", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "git@gitea.com:gitea/tea", want: RepoURL{Host: "gitea.com", Owner: "gitea", Name: "tea"}},
		{in: "", isErr: true},
		{in: "ftp://x.com/a/b", isErr: true},
		{in: "https://github.com/onlyowner", isErr: true},
		{in: "https:///a/b", isErr: true},
		{in: "plain-text", isErr: true},
	}
	for _, c := range cases {
		got, err := ParseRepoURL(c.in)
		if c.isErr {
			if err == nil {
				t.Errorf("ParseRepoURL(%q): error 기대, got %+v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRepoURL(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseRepoURL(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestRepoURLHelpers(t *testing.T) {
	r := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "proj"}
	if r.Slug() != "grp/sub/proj" {
		t.Errorf("Slug() = %q", r.Slug())
	}
	if r.HTTPS() != "https://git.example.com/grp/sub/proj" {
		t.Errorf("HTTPS() = %q", r.HTTPS())
	}
}

// fakeExec는 "명령 인자..." 문자열 key로 응답을 돌려준다.
func fakeExec(t *testing.T, responses map[string]string) {
	t.Helper()
	origRun, origLook := runOut, lookPath
	t.Cleanup(func() { runOut, lookPath = origRun, origLook })
	runOut = func(name string, args ...string) (string, error) {
		key := name + " " + strings.Join(args, " ")
		if out, ok := responses[key]; ok {
			return out, nil
		}
		return "", fmt.Errorf("fake: no response for %q", key)
	}
	lookPath = func(bin string) (string, error) {
		if _, ok := responses["BIN "+bin]; ok {
			return bin, nil
		}
		return "", fmt.Errorf("fake: %s not installed", bin)
	}
}

func TestCurrentRemoteURL(t *testing.T) {
	// 1) branch upstream remote 우선
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD":     "main",
		"git config --get branch.main.remote": "up",
		"git remote get-url up":               "https://github.com/o/r.git",
	})
	if u, err := CurrentRemoteURL(); err != nil || u != "https://github.com/o/r.git" {
		t.Errorf("upstream: %q %v", u, err)
	}

	// 2) origin fallback
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD": "main",
		"git remote get-url origin":       "git@gitlab.com:g/p.git",
	})
	if u, err := CurrentRemoteURL(); err != nil || u != "git@gitlab.com:g/p.git" {
		t.Errorf("origin: %q %v", u, err)
	}

	// 3) 유일 remote fallback
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD": "main",
		"git remote":                      "fork",
		"git remote get-url fork":         "https://gitea.com/o/r",
	})
	if u, err := CurrentRemoteURL(); err != nil || u != "https://gitea.com/o/r" {
		t.Errorf("single: %q %v", u, err)
	}

	// 4) 여러 remote, upstream/origin 없음 → 오류
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD": "main",
		"git remote":                      "a\nb",
	})
	if _, err := CurrentRemoteURL(); err == nil {
		t.Error("여러 remote는 error여야 함")
	}
}

func TestDetectProviderDefaultDomains(t *testing.T) {
	cfg := Config{Hosts: map[string]string{}}
	for host, want := range map[string]Provider{
		"github.com": GH, "gitlab.com": GLab, "gitea.com": Tea,
	} {
		got, err := DetectProvider(host, &cfg, false)
		if err != nil || got != want {
			t.Errorf("%s = %v (%v), want %v", host, got, err, want)
		}
	}
}

func TestDetectProviderSaved(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"BIN glab": "",
		"glab auth status --hostname git.example.com": "",
	})
	cfg := Config{Hosts: map[string]string{"git.example.com": "glab"}}
	got, err := DetectProvider("git.example.com", &cfg, false)
	if err != nil || got != GLab {
		t.Errorf("saved = %v (%v)", got, err)
	}
}

func TestDetectProviderSavedGoneRedetects(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	// 저장값은 glab이지만 glab 미설치 → gh 후보 하나로 재판별
	fakeExec(t, map[string]string{
		"BIN gh": "",
		"gh auth status --hostname git.example.com --json hosts": `{"hosts":{"git.example.com":[{}]}}`,
	})
	cfg := Config{Hosts: map[string]string{"git.example.com": "glab"}}
	got, err := DetectProvider("git.example.com", &cfg, false)
	if err != nil || got != GH {
		t.Errorf("redetect = %v (%v)", got, err)
	}
}

func TestDetectProviderNoCandidates(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{})
	cfg := Config{Hosts: map[string]string{}}
	_, err := DetectProvider("git.example.com", &cfg, false)
	if err == nil || !strings.Contains(err.Error(), "auth login") {
		t.Errorf("로그인 안내 error 기대, got %v", err)
	}
}

func TestDetectProviderMultipleNonInteractive(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"BIN gh":   "",
		"BIN glab": "",
		"gh auth status --hostname git.example.com --json hosts": `{"hosts":{"git.example.com":[{}]}}`,
		"glab auth status --hostname git.example.com":            "",
	})
	cfg := Config{Hosts: map[string]string{}}
	_, err := DetectProvider("git.example.com", &cfg, false)
	if err == nil || !strings.Contains(err.Error(), ConfigPath()) {
		t.Errorf("설정 경로 포함 error 기대, got %v", err)
	}
}

func TestDetectProviderInteractiveChoiceSaves(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"BIN gh":   "",
		"BIN glab": "",
		"gh auth status --hostname git.example.com --json hosts": `{"hosts":{"git.example.com":[{}]}}`,
		"glab auth status --hostname git.example.com":            "",
	})
	origStdin := stdin
	t.Cleanup(func() { stdin = origStdin })
	stdin = strings.NewReader("2\n")
	cfg := Config{Hosts: map[string]string{}}
	got, err := DetectProvider("git.example.com", &cfg, true)
	if err != nil || got != GLab {
		t.Fatalf("choice = %v (%v)", got, err)
	}
	saved, err := LoadConfig()
	if err != nil || saved.Hosts["git.example.com"] != "glab" {
		t.Errorf("저장 안 됨: %v %v", saved.Hosts, err)
	}
}

func TestTeaLoginName(t *testing.T) {
	fakeExec(t, map[string]string{
		"tea logins list --output json": `[{"name":"corp","url":"https://gitea.example.com"},{"name":"pub","url":"https://gitea.com"}]`,
	})
	if got := teaLoginName("gitea.example.com"); got != "corp" {
		t.Errorf("teaLoginName = %q", got)
	}
	if got := teaLoginName("unknown.com"); got != "" {
		t.Errorf("없는 host = %q", got)
	}
}
