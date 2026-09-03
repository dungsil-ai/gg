package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseRequestCI(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "ci list", args: []string{"ci", "list"},
			want: Request{Resource: "ci", Action: "list"}},
		{name: "ci list flag", args: []string{"ci", "list", "--limit", "5", "--branch", "main"},
			want: Request{Resource: "ci", Action: "list", Limit: "5", Branch: "main"}},
		{name: "ci view id", args: []string{"ci", "view", "123"},
			want: Request{Resource: "ci", Action: "view", Number: "123"}},
		{name: "ci view 생략", args: []string{"ci", "view"},
			want: Request{Resource: "ci", Action: "view"}},
		{name: "ci watch id", args: []string{"ci", "watch", "123"},
			want: Request{Resource: "ci", Action: "watch", Number: "123"}},
		{name: "ci watch 생략", args: []string{"ci", "watch"},
			want: Request{Resource: "ci", Action: "watch"}},
		{name: "ci retry", args: []string{"ci", "retry", "123"},
			want: Request{Resource: "ci", Action: "retry", Number: "123"}},
		{name: "ci cancel", args: []string{"ci", "cancel", "123"},
			want: Request{Resource: "ci", Action: "cancel", Number: "123"}},
		{name: "ci list repo 문맥", args: []string{"ci", "list", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "ci", Action: "list", RepoFlag: "https://github.com/o/r"}},
		{name: "ci list remote 문맥", args: []string{"ci", "list", "--remote", "upstream"},
			want: Request{Resource: "ci", Action: "list", RemoteFlag: "upstream"}},
		{name: "ci retry explain", args: []string{"ci", "retry", "123", "--explain"},
			want: Request{Resource: "ci", Action: "retry", Number: "123", Explain: true}},
	}
	for _, c := range cases {
		got, err := ParseRequest(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}

	bad := []struct {
		args []string
		want string
	}{
		{args: []string{"ci"}, want: "ci needs an action: list, view, watch, retry, cancel"},
		{args: []string{"ci", "status"}, want: "ci does not support status"},
		{args: []string{"ci", "list", "extra"}, want: "unexpected argument extra"},
		{args: []string{"ci", "view", "1", "2"}, want: "usage: gg ci view [<id>]"},
		{args: []string{"ci", "watch", "1", "2"}, want: "usage: gg ci watch [<id>]"},
		{args: []string{"ci", "retry"}, want: "usage: gg ci retry <id>"},
		{args: []string{"ci", "retry", "1", "2"}, want: "usage: gg ci retry <id>"},
		{args: []string{"ci", "cancel"}, want: "usage: gg ci cancel <id>"},
		{args: []string{"ci", "list", "--wat"}, want: "unknown flag --wat"},
	}
	for _, c := range bad {
		_, err := ParseRequest(c.args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", c.args, err)
			continue
		}
		if ue.Msg != c.want {
			t.Errorf("ParseRequest(%v) = %q, want %q", c.args, ue.Msg, c.want)
		}
	}
}

func TestParseRequestActionsAliasUsesCanonicalCommand(t *testing.T) {
	tests := []struct {
		name      string
		canonical []string
		alias     []string
	}{
		{
			name:      "list with flags",
			canonical: []string{"ci", "list", "--limit", "3", "--branch", "main"},
			alias:     []string{"actions", "list", "--limit", "3", "--branch", "main"},
		},
		{
			name:      "view with repo context",
			canonical: []string{"ci", "view", "123", "--repo", "https://github.com/o/r"},
			alias:     []string{"actions", "view", "123", "--repo", "https://github.com/o/r"},
		},
		{
			name:      "retry with remote context",
			canonical: []string{"ci", "retry", "123", "--remote", "upstream"},
			alias:     []string{"actions", "retry", "123", "--remote", "upstream"},
		},
		{
			name:      "explain before alias",
			canonical: []string{"--explain", "ci", "list"},
			alias:     []string{"--explain", "actions", "list"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := ParseRequest(tt.canonical)
			if err != nil {
				t.Fatalf("ParseRequest(%v): %v", tt.canonical, err)
			}
			got, err := ParseRequest(tt.alias)
			if err != nil {
				t.Fatalf("ParseRequest(%v): %v", tt.alias, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ParseRequest(%v) = %+v, want ParseRequest(%v) = %+v", tt.alias, got, tt.canonical, want)
			}
		})
	}
}

func TestTranslateCI(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}

	cases := []struct {
		name string
		req  Request
		repo RepoURL
		p    Provider
		want Invocation
	}{
		{name: "gh ci list",
			req:  Request{Resource: "ci", Action: "list", Limit: "5"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "list", "-R", "github.com/o/r", "--limit", "5"}}},
		{name: "gh ci list branch",
			req:  Request{Resource: "ci", Action: "list", Branch: "main"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "list", "-R", "github.com/o/r", "--branch", "main"}}},
		{name: "gh ci view id",
			req:  Request{Resource: "ci", Action: "view", Number: "123"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "view", "123", "-R", "github.com/o/r"}}},
		{name: "gh ci view 최근 run",
			req:  Request{Resource: "ci", Action: "view"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "view", "-R", "github.com/o/r"}}},
		{name: "gh ci watch",
			req:  Request{Resource: "ci", Action: "watch", Number: "123"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "watch", "123", "-R", "github.com/o/r"}}},
		{name: "gh ci retry는 rerun",
			req:  Request{Resource: "ci", Action: "retry", Number: "123"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "rerun", "123", "-R", "github.com/o/r"}}},
		{name: "gh ci cancel",
			req:  Request{Resource: "ci", Action: "cancel", Number: "123"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"run", "cancel", "123", "-R", "github.com/o/r"}}},
		{name: "glab ci list",
			req:  Request{Resource: "ci", Action: "list", Limit: "5"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "list", "--repo", "https://git.example.com/grp/sub/p", "--per-page", "5"}}},
		{name: "glab ci list branch는 ref",
			req:  Request{Resource: "ci", Action: "list", Branch: "main"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "list", "--repo", "https://git.example.com/grp/sub/p", "--ref", "main"}}},
		{name: "glab ci view는 get",
			req:  Request{Resource: "ci", Action: "view", Number: "123"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "get", "--pipeline-id", "123", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab ci watch는 trace",
			req:  Request{Resource: "ci", Action: "watch", Number: "123"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "trace", "123", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab ci retry",
			req:  Request{Resource: "ci", Action: "retry", Number: "123"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "retry", "123", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab ci cancel",
			req:  Request{Resource: "ci", Action: "cancel", Number: "123"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"ci", "cancel", "pipeline", "123", "--repo", "https://git.example.com/grp/sub/p"}}},
	}
	for _, c := range cases {
		got, err := Translate(c.req, c.repo, c.p, "")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestTranslateCITeaUnsupported(t *testing.T) {
	_, err := Translate(
		Request{Resource: "ci", Action: "list"},
		RepoURL{Host: "gitea.com", Owner: "o", Name: "r"},
		Tea,
		"corp",
	)
	var ue UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("Translate(ci list, tea): UsageError 기대, got %v", err)
	}
	if ue.Msg != "ci is not supported for tea" {
		t.Errorf("Tea 오류 = %q", ue.Msg)
	}
}

func TestPlanCITeaSkipsLogin(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{})

	_, err := plan(Request{Resource: "ci", Action: "list", RepoFlag: "https://gitea.com/o/r"})
	var ue UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("plan(ci list, tea): UsageError 기대, got %v", err)
	}
	if ue.Msg != "ci is not supported for tea" {
		t.Errorf("Tea 오류 = %q", ue.Msg)
	}
}

// TestE2ECIInvocations는 gg ci의 실제 gh/glab argv를 본다.
func TestE2ECIInvocations(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	writeFakeBin(t, fakeDir, "glab", logFile)
	ghRepo := tempRepo(t, "https://github.com/o/r.git")
	glabRepo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		dir  string
		want string
	}{
		{[]string{"ci", "list"}, ghRepo, "gh run list -R github.com/o/r"},
		{[]string{"ci", "list", "--limit", "5"}, ghRepo, "gh run list -R github.com/o/r --limit 5"},
		{[]string{"ci", "list", "--branch", "main"}, ghRepo, "gh run list -R github.com/o/r --branch main"},
		{[]string{"ci", "view", "123"}, ghRepo, "gh run view 123 -R github.com/o/r"},
		{[]string{"ci", "watch", "123"}, ghRepo, "gh run watch 123 -R github.com/o/r"},
		{[]string{"ci", "retry", "123"}, ghRepo, "gh run rerun 123 -R github.com/o/r"},
		{[]string{"ci", "cancel", "123"}, ghRepo, "gh run cancel 123 -R github.com/o/r"},
		{[]string{"ci", "list"}, glabRepo, "glab ci list --repo https://gitlab.com/o/r"},
		{[]string{"ci", "list", "--branch", "main", "--limit", "5"}, glabRepo, "glab ci list --repo https://gitlab.com/o/r --ref main --per-page 5"},
		{[]string{"ci", "view", "123"}, glabRepo, "glab ci get --pipeline-id 123 --repo https://gitlab.com/o/r"},
		{[]string{"ci", "watch", "123"}, glabRepo, "glab ci trace 123 --repo https://gitlab.com/o/r"},
		{[]string{"ci", "retry", "123"}, glabRepo, "glab ci retry 123 --repo https://gitlab.com/o/r"},
		{[]string{"ci", "cancel", "123"}, glabRepo, "glab ci cancel pipeline 123 --repo https://gitlab.com/o/r"},
		// alias: actions는 ci와 같은 invocation을 낸다
		{[]string{"actions", "list", "--limit", "3"}, ghRepo, "gh run list -R github.com/o/r --limit 3"},
		{[]string{"actions", "cancel", "123"}, glabRepo, "glab ci cancel pipeline 123 --repo https://gitlab.com/o/r"},
	}
	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, tc.dir, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestE2ECIAliasHelpMatchesCanonical은 gg actions --help가 gg ci --help와
// 같은 출력을 내는지 본다.
func TestE2ECIAliasHelpMatchesCanonical(t *testing.T) {
	bin := buildGG(t)
	aliasOut, aliasErr, aliasCode := runGGStreams(t, bin, t.TempDir(), "actions", "--help")
	canonOut, canonErr, canonCode := runGGStreams(t, bin, t.TempDir(), "ci", "--help")
	if aliasCode != 0 || canonCode != 0 {
		t.Fatalf("help exit = %d(alias), %d(canon)", aliasCode, canonCode)
	}
	if aliasErr != "" || canonErr != "" {
		t.Fatalf("help stderr = %q(alias), %q(canon)", aliasErr, canonErr)
	}
	if aliasOut != canonOut {
		t.Errorf("gg actions --help가 gg ci --help와 다름:\n%q\n%q", aliasOut, canonOut)
	}
	if !strings.Contains(aliasOut, "watch") || !strings.Contains(aliasOut, "cancel") {
		t.Errorf("gg ci --help에 action 목록 없음:\n%s", aliasOut)
	}
}

// TestE2ECIGiteaUnsupported는 gitea remote에서 gg ci가 tea를 실행하지 않고
// usage error로 끝나는지 본다.
func TestE2ECIGiteaUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "ci", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2: %s", code, out)
	}
	if !strings.Contains(out, "ci is not supported for tea") {
		t.Errorf("output: %s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea should not run, got %q", got)
	}
}
