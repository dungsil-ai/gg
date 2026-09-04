package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestParseRequestRepoLifecycle(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "repo fork", args: []string{"repo", "fork"},
			want: Request{Resource: "repo", Action: "fork"}},
		{name: "fork 생략형과 repo 문맥", args: []string{"fork", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "repo", Action: "fork", RepoFlag: "https://github.com/o/r"}},
		{name: "repo delete 확인 생략", args: []string{"repo", "delete", "--yes"},
			want: Request{Resource: "repo", Action: "delete", Yes: true}},
		{name: "delete 생략형", args: []string{"delete"},
			want: Request{Resource: "repo", Action: "delete"}},
		{name: "repo edit 설명", args: []string{"repo", "edit", "--description", "d"},
			want: Request{Resource: "repo", Action: "edit", Description: "d"}},
		{name: "repo edit 가시성", args: []string{"repo", "edit", "--public"},
			want: Request{Resource: "repo", Action: "edit", Public: true}},
		{name: "repo rename 새 이름", args: []string{"repo", "rename", "newname", "--yes"},
			want: Request{Resource: "repo", Action: "rename", Name: "newname", Yes: true}},
		{name: "rename 생략형", args: []string{"rename", "newname"},
			want: Request{Resource: "repo", Action: "rename", Name: "newname"}},
		{name: "repo sync 전체 flag", args: []string{"repo", "sync", "--branch", "main", "--source", "o/up", "--force"},
			want: Request{Resource: "repo", Action: "sync", Branch: "main", Source: "o/up", Force: true}},
		{name: "sync 생략형", args: []string{"sync"},
			want: Request{Resource: "repo", Action: "sync"}},
		{name: "repo set-default", args: []string{"repo", "set-default"},
			want: Request{Resource: "repo", Action: "set-default"}},
		{name: "repo set-default unset", args: []string{"repo", "set-default", "--unset"},
			want: Request{Resource: "repo", Action: "set-default", Unset: true}},
		{name: "set-default 생략형 view", args: []string{"set-default", "--view"},
			want: Request{Resource: "repo", Action: "set-default", View: true}},
		{name: "repo set-default 문맥 flag", args: []string{"repo", "set-default", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "repo", Action: "set-default", RepoFlag: "https://github.com/o/r"}},
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
}

func TestParseRequestRepoLifecycleErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "edit은 가시성 둘 다", args: []string{"repo", "edit", "--public", "--private"},
			want: "repo edit needs at most one of --public or --private"},
		{name: "edit은 바뀜 없이", args: []string{"repo", "edit"},
			want: "repo edit needs at least one of --description, --public, or --private"},
		{name: "rename은 이름 하나만", args: []string{"repo", "rename", "a", "b"},
			want: "usage: gg repo rename [<new-name>]"},
		{name: "set-default는 unset과 view 동시", args: []string{"repo", "set-default", "--unset", "--view"},
			want: "repo set-default needs at most one of --unset or --view"},
		{name: "set-default는 explain 미지원", args: []string{"repo", "set-default", "--explain"},
			want: "--explain is not supported for repo set-default"},
		{name: "set-default unset도 explain 미지원", args: []string{"repo", "set-default", "--unset", "--explain"},
			want: "--explain is not supported for repo set-default"},
		{name: "sync 모르는 flag", args: []string{"repo", "sync", "--nope"},
			want: "unknown flag --nope"},
	}
	for _, c := range cases {
		_, err := ParseRequest(c.args)
		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("%s: UsageError 기대, got %v", c.name, err)
			continue
		}
		if !strings.Contains(usage.Msg, c.want) {
			t.Errorf("%s 오류 = %q, want %q 포함", c.name, usage.Msg, c.want)
		}
	}
}

func TestTranslateRepoLifecycle(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}
	te := RepoURL{Host: "gitea.example.com", Owner: "o", Name: "r"}

	cases := []struct {
		name string
		req  Request
		repo RepoURL
		p    Provider
		tea  string
		want Invocation
	}{
		{name: "gh fork",
			req: Request{Resource: "repo", Action: "fork"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "fork", "https://github.com/o/r"}}},
		{name: "glab fork는 slug를 쓴다",
			req: Request{Resource: "repo", Action: "fork"}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "fork", "grp/sub/p"}}},
		{name: "tea fork",
			req: Request{Resource: "repo", Action: "fork"}, repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "fork", "--login", "corp", "--repo", "o/r"}}},
		{name: "gh delete는 --yes 없이 확인을 묻는다",
			req: Request{Resource: "repo", Action: "delete"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "delete", "https://github.com/o/r"}}},
		{name: "gh delete --yes",
			req: Request{Resource: "repo", Action: "delete", Yes: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "delete", "https://github.com/o/r", "--yes"}}},
		{name: "glab delete",
			req: Request{Resource: "repo", Action: "delete", Yes: true}, repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "delete", "grp/sub/p", "--yes"}}},
		{name: "tea delete는 --force로 확인을 건너뛴다",
			req: Request{Resource: "repo", Action: "delete", Yes: true}, repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "delete", "--login", "corp", "--owner", "o", "--name", "r", "--force"}}},
		{name: "tea delete는 확인 프롬프트를 중계한다",
			req: Request{Resource: "repo", Action: "delete"}, repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "delete", "--login", "corp", "--owner", "o", "--name", "r"}}},
		{name: "gh edit 설명",
			req: Request{Resource: "repo", Action: "edit", Description: "d"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "edit", "https://github.com/o/r", "--description", "d"}}},
		{name: "gh edit private은 영향 수락 flag을 붙인다",
			req: Request{Resource: "repo", Action: "edit", Private: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "edit", "https://github.com/o/r", "--visibility", "private", "--accept-visibility-change-consequences"}}},
		{name: "gh edit public과 설명 함께",
			req: Request{Resource: "repo", Action: "edit", Public: true, Description: "d"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "edit", "https://github.com/o/r", "--description", "d", "--visibility", "public", "--accept-visibility-change-consequences"}}},
		{name: "gh rename",
			req: Request{Resource: "repo", Action: "rename", Name: "newname", Yes: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "rename", "newname", "-R", "github.com/o/r", "--yes"}}},
		{name: "gh rename은 이름을 비우면 gh가 묻는다",
			req: Request{Resource: "repo", Action: "rename"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "rename", "-R", "github.com/o/r"}}},
		{name: "gh sync 전체 flag",
			req: Request{Resource: "repo", Action: "sync", Branch: "main", Source: "o/up", Force: true}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "sync", "https://github.com/o/r", "--branch", "main", "--source", "o/up", "--force"}}},
		{name: "gh set-default",
			req: Request{Resource: "repo", Action: "set-default"}, repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "set-default", "https://github.com/o/r"}}},
	}
	for _, c := range cases {
		got, err := Translate(c.req, c.repo, c.p, c.tea)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestTranslateRepoLifecycleUnsupported(t *testing.T) {
	repo := RepoURL{Host: "git.example.com", Owner: "o", Name: "r"}
	for _, p := range []Provider{GLab, Tea} {
		for _, action := range []string{"edit", "rename", "sync", "set-default"} {
			_, err := Translate(Request{Resource: "repo", Action: action}, repo, p, "corp")
			var usage UsageError
			if !errors.As(err, &usage) {
				t.Errorf("%s %s: UsageError 기대, got %v", p, action, err)
				continue
			}
			if want := "repo does not support " + action; usage.Msg != want {
				t.Errorf("%s %s 오류 = %q, want %q", p, action, usage.Msg, want)
			}
		}
	}
}

func TestResolvePlanRepoSetDefaultBypass(t *testing.T) {
	t.Run("unset과 view는 git 저장소 밖에서도 gh로 바로 간다", func(t *testing.T) {
		for _, tc := range []struct {
			req  Request
			want []string
		}{
			{Request{Resource: "repo", Action: "set-default", Unset: true},
				[]string{"repo", "set-default", "--unset"}},
			{Request{Resource: "repo", Action: "set-default", View: true},
				[]string{"repo", "set-default", "--view"}},
		} {
			ep, err := resolvePlan(tc.req)
			if err != nil {
				t.Fatalf("resolvePlan(%+v): %v", tc.req, err)
			}
			want := Invocation{Bin: "gh", Args: tc.want}
			if !reflect.DeepEqual(ep.inv, want) {
				t.Errorf("resolvePlan(%+v) = %+v, want %+v", tc.req, ep.inv, want)
			}
		}
	})
	t.Run("기본 모드는 git 저장소가 필요하다", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, err := resolvePlan(Request{Resource: "repo", Action: "set-default"}); err == nil {
			t.Error("git 저장소 밖 set-default가 오류 없이 통과됨")
		}
	})
	t.Run("기본 모드는 문맥 저장소를 넘긴다", func(t *testing.T) {
		t.Chdir(tempRepo(t, "https://github.com/o/r.git"))
		t.Setenv("GG_HOME", t.TempDir())
		ep, err := resolvePlan(Request{Resource: "repo", Action: "set-default"})
		if err != nil {
			t.Fatalf("resolvePlan: %v", err)
		}
		want := Invocation{Bin: "gh", Args: []string{"repo", "set-default", "https://github.com/o/r"}}
		if !reflect.DeepEqual(ep.inv, want) {
			t.Errorf("resolvePlan = %+v, want %+v", ep.inv, want)
		}
	})
}

func TestE2ERepoLifecycleInvocations(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"fork", []string{"repo", "fork"}, "gh repo fork https://github.com/o/r"},
		{"delete", []string{"repo", "delete", "--yes"}, "gh repo delete https://github.com/o/r --yes"},
		{"edit 설명", []string{"repo", "edit", "--description", "hello"}, "gh repo edit https://github.com/o/r --description hello"},
		{"edit 가시성", []string{"repo", "edit", "--private"}, "gh repo edit https://github.com/o/r --visibility private --accept-visibility-change-consequences"},
		{"rename", []string{"repo", "rename", "newname", "--yes"}, "gh repo rename newname -R github.com/o/r --yes"},
		{"sync", []string{"repo", "sync", "--source", "o/up", "--force"}, "gh repo sync https://github.com/o/r --source o/up --force"},
		{"set-default", []string{"repo", "set-default"}, "gh repo set-default https://github.com/o/r"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(logFile, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			out, code := runGG(t, bin, fakeDir, repo, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		})
	}

	t.Run("fork 생략형은 repo 접두 형태와 같다", func(t *testing.T) {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, code := runGG(t, bin, fakeDir, repo, "repo", "fork"); code != 0 {
			t.Fatalf("gg repo fork: exit %d", code)
		}
		canonical := readLog(t, logFile)
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, code := runGG(t, bin, fakeDir, repo, "fork"); code != 0 {
			t.Fatalf("gg fork: exit %d", code)
		}
		if got := readLog(t, logFile); got != canonical {
			t.Errorf("gg fork argv = %q, want %q", got, canonical)
		}
	})

	t.Run("set-default unset과 view는 git 저장소 밖에서 동작한다", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			want string
		}{
			{[]string{"repo", "set-default", "--unset"}, "gh repo set-default --unset"},
			{[]string{"repo", "set-default", "--view"}, "gh repo set-default --view"},
		} {
			if err := os.WriteFile(logFile, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			out, code := runGG(t, bin, fakeDir, t.TempDir(), tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		}
	})

	t.Run("set-default는 git 저장소 밖에서 오류다", func(t *testing.T) {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), "repo", "set-default")
		if code != 1 {
			t.Fatalf("exit = %d, want 1: %s", code, out)
		}
		if !strings.Contains(out, "not a git repository") {
			t.Errorf("output에 오류 없음: %s", out)
		}
	})
}

func TestE2ERepoLifecycleUnsupportedForGitLab(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"repo", "edit", "--description", "d"}, "repo does not support edit"},
		{[]string{"repo", "rename", "newname"}, "repo does not support rename"},
		{[]string{"repo", "sync", "--force"}, "repo does not support sync"},
		{[]string{"repo", "set-default"}, "repo does not support set-default"},
	} {
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", tc.args, code, out)
		}
		if !strings.Contains(out, tc.want) {
			t.Errorf("gg %v output에 미지원 오류 없음: %s", tc.args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got %q", tc.args, got)
		}
	}
}

func TestE2ERepoForkDeleteForGitea(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	var scriptPath, body string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(fakeDir, "tea.cmd")
		body = "@echo off\r\nif \"%1\"==\"logins\" if \"%2\"==\"list\" (\r\n  echo [{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]\r\n  exit /b 0\r\n)\r\necho tea %* >> \"" + logFile + "\"\r\nexit /b 0\r\n"
	} else {
		scriptPath = filepath.Join(fakeDir, "tea")
		body = "#!/bin/sh\nif [ \"$1\" = \"logins\" ] && [ \"$2\" = \"list\" ]; then\n  echo '[{\"name\":\"pub\",\"url\":\"https://gitea.com\"}]'\n  exit 0\nfi\necho \"tea $@\" >> \"" + logFile + "\"\nexit 0\n"
	}
	if err := os.WriteFile(scriptPath, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"fork", []string{"repo", "fork"}, "tea repos fork --login pub --repo o/r"},
		{"delete", []string{"repo", "delete", "--yes"}, "tea repos delete --login pub --owner o --name r --force"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(logFile, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			out, code := runGG(t, bin, fakeDir, repo, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}
