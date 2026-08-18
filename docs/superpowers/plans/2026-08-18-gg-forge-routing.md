# gg forge 라우팅 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 저장소 URL의 host를 보고 `gh`/`glab`/`tea`/`git`으로 라우팅하는 Go 단일 실행 파일 `gg`를 만든다.

**Architecture:** 얇은 stdlib 변환기. `command.go`가 공통 문법을 파싱하고, `route.go`가 remote/URL/provider를 판별하고, `config.go`가 host별 선택을 저장하고, `main.go`가 자식 process를 stdio 상속으로 실행한다. forge API 호출 없음.

**Tech Stack:** Go stdlib만 사용 (외부 dependency 0개). 자식 CLI: `gh`, `glab`, `tea`, `git`.

**Spec:** `docs/superpowers/specs/2026-08-18-gg-forge-routing-design.md`

## Global Constraints

- 외부 Go dependency 금지. `go.mod`에 require 항목이 없어야 한다.
- module 이름 `gg`, `go 1.22` (신규 API 사용 금지, 1.22 floor 유지).
- binary 이름 `gg`.
- 설정 경로 fallback: `$GG_HOME` → `$XDG_CONFIG_HOME/gg` → `~/.gg`, 파일 이름 `config.json`.
- 설정에는 host → `gh|glab|tea` 만 저장. token/URL/username 저장 금지.
- exit code: `0` 성공, `1` 실행 전 오류, `2` 사용법 오류, `127` 실행 파일 없음, 자식 실행 후에는 자식 exit code 그대로.
- 자식 process는 stdin/stdout/stderr 상속. 출력 가공 금지.
- commit message: type 접두어는 영어 소문자(`feat:`, `fix:`, `test:`, `docs:`, `chore:`), 요약은 한국어.
- 개발 PC(Windows)에 Go가 없으면 먼저 설치: `winget install GoLang.Go` (새 터미널에서 `go version` 확인).
- flag 변환표는 spec의 검증된 표를 그대로 따른다 (GitLab list: `--closed`/`--all`/`--per-page`).

## 파일 구조

- `go.mod` — module 선언만.
- `main.go` — entrypoint, usage 문자열, `plan()` 조립, 자식 실행, exit code.
- `route.go` — `ParseRepoURL`, `CurrentRemoteURL`, `DetectProvider`, 로그인 probe, `teaLoginName`.
- `command.go` — `Request`, `ParseRequest`, `Translate`, `Invocation`, `UsageError`.
- `config.go` — `Config`, `ConfigDir/ConfigPath`, `LoadConfig`, `SaveProvider`.
- `route_test.go`, `command_test.go`, `config_test.go`, `e2e_test.go`.

모든 코드는 `package main` 하나에 둔다. package 분리 금지 (spec).

---

### Task 1: Go module과 URL parser

**Files:**
- Create: `go.mod`
- Create: `route.go`
- Test: `route_test.go`

**Interfaces:**
- Consumes: 없음 (첫 task)
- Produces:
  - `type Provider string` / 상수 `GH`, `GLab`, `Tea`
  - `type RepoURL struct { Host, Owner, Name string }`
  - `func (r RepoURL) Slug() string` — `owner/repo`
  - `func (r RepoURL) HTTPS() string` — `https://host/owner/repo`
  - `func ParseRepoURL(raw string) (RepoURL, error)`

- [ ] **Step 1: module 생성**

```bash
cd P:/OSS/gg
go mod init gg
```

`go.mod` 내용 확인 (버전 줄은 `go 1.22`로 맞춘다):

```text
module gg

go 1.22
```

- [ ] **Step 2: 실패하는 테스트 작성** — `route_test.go`

```go
package main

import "testing"

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
```

- [ ] **Step 3: 실패 확인**

Run: `go test ./... -run TestParseRepoURL -v`
Expected: FAIL — `undefined: RepoURL`, `undefined: ParseRepoURL` compile error

- [ ] **Step 4: 최소 구현** — `route.go`

```go
package main

import (
	"fmt"
	"net/url"
	"strings"
)

type Provider string

const (
	GH   Provider = "gh"
	GLab Provider = "glab"
	Tea  Provider = "tea"
)

// RepoURL은 저장소 URL에서 얻은 forge 좌표다. Host는 lowercase, port 없음.
type RepoURL struct {
	Host  string
	Owner string // GitLab namespace는 "grp/sub"처럼 /를 포함할 수 있다
	Name  string // 끝의 .git 제거
}

func (r RepoURL) Slug() string  { return r.Owner + "/" + r.Name }
func (r RepoURL) HTTPS() string { return "https://" + r.Host + "/" + r.Slug() }

// ParseRepoURL은 https/ssh/SCP 형식 저장소 URL을 파싱한다.
func ParseRepoURL(raw string) (RepoURL, error) {
	raw = strings.TrimSpace(raw)
	var host, path string
	switch {
	case strings.HasPrefix(raw, "https://"), strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "ssh://"):
		u, err := url.Parse(raw)
		if err != nil {
			return RepoURL{}, fmt.Errorf("invalid repository URL %q: %v", raw, err)
		}
		host = u.Hostname()
		path = u.Path
	case raw == "", strings.Contains(raw, "://"):
		return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
	default: // SCP 형식: [user@]host:owner/repo
		i := strings.Index(raw, ":")
		if i < 0 {
			return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
		}
		host = raw[:i]
		if at := strings.Index(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		path = raw[i+1:]
	}
	host = strings.ToLower(host)
	var segs []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if host == "" || len(segs) < 2 {
		return RepoURL{}, fmt.Errorf("repository URL %q must look like host/owner/repo", raw)
	}
	name := strings.TrimSuffix(segs[len(segs)-1], ".git")
	if name == "" {
		return RepoURL{}, fmt.Errorf("repository URL %q has an empty repository name", raw)
	}
	return RepoURL{Host: host, Owner: strings.Join(segs[:len(segs)-1], "/"), Name: name}, nil
}
```

- [ ] **Step 5: 통과 확인**

Run: `go test ./... -v`
Expected: PASS (2 tests)

- [ ] **Step 6: Commit**

```bash
git add go.mod route.go route_test.go
git commit -m "feat: 저장소 URL parser 추가"
```

---

### Task 2: 설정 파일

**Files:**
- Create: `config.go`
- Test: `config_test.go`

**Interfaces:**
- Consumes: `Provider` (Task 1)
- Produces:
  - `type Config struct { Hosts map[string]string }`
  - `func ConfigDir() string`, `func ConfigPath() string`
  - `func LoadConfig() (Config, error)` — 파일 없음 = 빈 설정 + nil error
  - `func SaveProvider(host string, p Provider) error` — atomic write

- [ ] **Step 1: 실패하는 테스트 작성** — `config_test.go`

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDirFallback(t *testing.T) {
	t.Setenv("GG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	t.Setenv("GG_HOME", `C:\gghome`)
	if got := ConfigDir(); got != `C:\gghome` {
		t.Errorf("GG_HOME 우선이어야 함, got %q", got)
	}

	t.Setenv("GG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("x", "cfg"))
	if got := ConfigDir(); got != filepath.Join("x", "cfg", "gg") {
		t.Errorf("XDG fallback = %q", got)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	if got := ConfigDir(); got != filepath.Join(home, ".gg") {
		t.Errorf("home fallback = %q", got)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("빈 설정이어야 함: %v", err)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
}

func TestSaveAndReload(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	if err := SaveProvider("git.example.com", GLab); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts["git.example.com"] != "glab" {
		t.Errorf("Hosts = %v", cfg.Hosts)
	}
	data, _ := os.ReadFile(ConfigPath())
	if !strings.Contains(string(data), `"hosts"`) {
		t.Errorf("json 형식이 아님: %s", data)
	}
}

func TestBrokenConfigIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GG_HOME", dir)
	bad := []byte("{broken json")
	if err := os.WriteFile(filepath.Join(dir, "config.json"), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("손상 파일은 error여야 함")
	}
	if err := SaveProvider("h", GH); err == nil {
		t.Fatal("손상 파일 위 저장은 실패해야 함")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(after) != string(bad) {
		t.Error("손상 파일이 덮어써짐")
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./... -run TestConfig -v`
Expected: FAIL — `undefined: ConfigDir` compile error

- [ ] **Step 3: 최소 구현** — `config.go`

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config는 host별 provider 선택만 담는다. token 저장 금지.
type Config struct {
	Hosts map[string]string `json:"hosts"`
}

// ConfigDir: $GG_HOME → $XDG_CONFIG_HOME/gg → ~/.gg
func ConfigDir() string {
	if d := os.Getenv("GG_HOME"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "gg")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gg")
}

func ConfigPath() string { return filepath.Join(ConfigDir(), "config.json") }

func LoadConfig() (Config, error) {
	cfg := Config{Hosts: map[string]string{}}
	data, err := os.ReadFile(ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("broken config %s: %v", ConfigPath(), err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]string{}
	}
	return cfg, nil
}

// SaveProvider는 temp 파일 + rename으로 원자적으로 저장한다.
func SaveProvider(host string, p Provider) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err // 손상 파일은 절대 덮어쓰지 않는다
	}
	cfg.Hosts[host] = string(p)
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(ConfigDir(), "config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), ConfigPath())
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./... -v`
Expected: PASS (기존 + 신규 4 tests)

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: host별 provider 설정 저장 추가"
```

---

### Task 3: 공통 명령 parser

**Files:**
- Create: `command.go`
- Test: `command_test.go`

**Interfaces:**
- Consumes: 없음
- Produces:
  - `type UsageError struct{ Msg string }` + `func (e UsageError) Error() string`
  - `func usageErr(m string) error`
  - `type Request struct` (아래 필드 그대로)
  - `func ParseRequest(args []string) (Request, error)`

- [ ] **Step 1: 실패하는 테스트 작성** — `command_test.go`

```go
package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseRequest(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "repo 생략 list", args: []string{"list", "--limit", "5"},
			want: Request{Resource: "repo", Action: "list", Limit: "5"}},
		{name: "repo 명시 list", args: []string{"repo", "list"},
			want: Request{Resource: "repo", Action: "list"}},
		{name: "전역 repo flag", args: []string{"--repo", "https://github.com/o/r", "view"},
			want: Request{Resource: "repo", Action: "view", RepoFlag: "https://github.com/o/r"}},
		{name: "후행 repo flag", args: []string{"issue", "list", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "issue", Action: "list", RepoFlag: "https://github.com/o/r"}},
		{name: "issue view", args: []string{"issue", "view", "42"},
			want: Request{Resource: "issue", Action: "view", Number: "42"}},
		{name: "issue create", args: []string{"issue", "create", "--title", "t", "--body", "b"},
			want: Request{Resource: "issue", Action: "create", Title: "t", Body: "b"}},
		{name: "pr list state", args: []string{"pr", "list", "--state", "all", "--limit", "3"},
			want: Request{Resource: "pr", Action: "list", State: "all", Limit: "3"}},
		{name: "pr create full", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"},
			want: Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true}},
		{name: "repo create", args: []string{"--repo", "https://gitea.com/o/r", "create", "--private", "--description", "d"},
			want: Request{Resource: "repo", Action: "create", RepoFlag: "https://gitea.com/o/r", Private: true, Description: "d"}},
		{name: "clone dir", args: []string{"clone", "https://github.com/o/r", "dst"},
			want: Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", CloneDir: "dst"}},
		{name: "pull 전달", args: []string{"pull", "--rebase", "origin", "main"},
			want: Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase", "origin", "main"}}},
		{name: "push 전달", args: []string{"repo", "push", "--force-with-lease"},
			want: Request{Resource: "repo", Action: "push", GitArgs: []string{"--force-with-lease"}}},
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

func TestParseRequestErrors(t *testing.T) {
	bad := [][]string{
		{},                                   // 명령 없음
		{"unknown"},                          // 알 수 없는 자원
		{"issue"},                            // action 없음
		{"issue", "close", "1"},              // 지원 안 하는 action
		{"issue", "view"},                    // number 없음
		{"issue", "view", "1", "2"},          // 인자 초과
		{"issue", "list", "--wat"},           // 알 수 없는 flag
		{"pr", "list", "--state", "merged"},  // 지원 안 하는 state
		{"pr", "create", "--title"},          // 값 없는 flag
		{"clone"},                            // URL 없음
		{"clone", "u", "d", "x"},             // 인자 초과
		{"create", "--public"},               // --repo 없는 repo create
		{"create", "--repo", "https://x.com/o/r"},                 // 공개 범위 없음
		{"create", "--repo", "https://x.com/o/r", "--public", "--private"}, // 둘 다 지정
		{"list", "extra"},                    // list에 positional
	}
	for _, args := range bad {
		_, err := ParseRequest(args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./... -run TestParseRequest -v`
Expected: FAIL — `undefined: ParseRequest` compile error

- [ ] **Step 3: 최소 구현** — `command.go`

```go
package main

import (
	"strings"
)

// UsageError는 exit code 2로 이어지는 사용법 오류다.
type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }
func usageErr(m string) error      { return UsageError{Msg: m} }

// Request는 파싱된 공통 명령이다.
type Request struct {
	Resource string // "repo" | "issue" | "pr"
	Action   string // list | view | create | clone | pull | push
	RepoFlag string // --repo 값
	Number   string // issue/pr view 대상
	CloneURL string
	CloneDir string
	GitArgs  []string // pull/push는 검사 없이 git으로 전달

	Title, Body, Base, Head, State, Limit, Description string
	Draft, Public, Private                             bool
}

var repoActions = map[string]bool{
	"list": true, "view": true, "create": true,
	"clone": true, "pull": true, "push": true,
}

func ParseRequest(args []string) (Request, error) {
	var req Request
	for len(args) >= 2 && args[0] == "--repo" {
		req.RepoFlag = args[1]
		args = args[2:]
	}
	if len(args) == 1 && args[0] == "--repo" {
		return req, usageErr("--repo needs a URL")
	}
	if len(args) == 0 {
		return req, usageErr("missing command")
	}
	head, rest := args[0], args[1:]
	switch {
	case head == "issue" || head == "pr":
		req.Resource = head
		if len(rest) == 0 {
			return req, usageErr(head + " needs an action: list, view, create")
		}
		req.Action, rest = rest[0], rest[1:]
		if req.Action != "list" && req.Action != "view" && req.Action != "create" {
			return req, usageErr(head + " does not support " + req.Action)
		}
	case head == "repo":
		req.Resource = "repo"
		if len(rest) == 0 || !repoActions[rest[0]] {
			return req, usageErr("repo needs an action: list, view, create, clone, pull, push")
		}
		req.Action, rest = rest[0], rest[1:]
	case repoActions[head]: // gg list == gg repo list
		req.Resource, req.Action = "repo", head
	default:
		return req, usageErr("unknown command " + head)
	}
	return req, parseRest(&req, rest)
}

// flagLoop은 허용된 flag만 소비하고 positional 인자를 돌려준다.
func flagLoop(req *Request, args []string, strs map[string]*string, bools map[string]*bool) ([]string, error) {
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		if a == "--repo" {
			if i+1 >= len(args) {
				return nil, usageErr("--repo needs a URL")
			}
			req.RepoFlag = args[i+1]
			i++
			continue
		}
		if p, ok := bools[a]; ok {
			*p = true
			continue
		}
		if p, ok := strs[a]; ok {
			if i+1 >= len(args) {
				return nil, usageErr(a + " needs a value")
			}
			*p = args[i+1]
			i++
			continue
		}
		return nil, usageErr("unknown flag " + a)
	}
	return pos, nil
}

func parseRest(req *Request, args []string) error {
	switch req.Resource + " " + req.Action {
	case "repo pull", "repo push":
		req.GitArgs = args
		return nil
	case "repo clone":
		pos, err := flagLoop(req, args, nil, nil)
		if err != nil {
			return err
		}
		if len(pos) < 1 || len(pos) > 2 {
			return usageErr("usage: gg clone <URL> [DIR]")
		}
		req.CloneURL = pos[0]
		if len(pos) == 2 {
			req.CloneDir = pos[1]
		}
		return nil
	case "repo list":
		return noPositional(req, args, map[string]*string{"--limit": &req.Limit}, nil)
	case "repo view":
		return noPositional(req, args, nil, nil)
	case "repo create":
		err := noPositional(req, args,
			map[string]*string{"--description": &req.Description},
			map[string]*bool{"--public": &req.Public, "--private": &req.Private})
		if err != nil {
			return err
		}
		if req.RepoFlag == "" {
			return usageErr("repo create needs --repo <new-repository-URL>")
		}
		if req.Public == req.Private {
			return usageErr("repo create needs exactly one of --public or --private")
		}
		return nil
	case "issue list", "pr list":
		if err := noPositional(req, args, map[string]*string{
			"--state": &req.State, "--limit": &req.Limit,
		}, nil); err != nil {
			return err
		}
		switch req.State {
		case "", "open", "closed", "all":
			return nil
		}
		return usageErr("--state must be open, closed, or all")
	case "issue view", "pr view":
		pos, err := flagLoop(req, args, nil, nil)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return usageErr("usage: gg " + req.Resource + " view <number>")
		}
		req.Number = pos[0]
		return nil
	case "issue create":
		return noPositional(req, args, map[string]*string{
			"--title": &req.Title, "--body": &req.Body,
		}, nil)
	case "pr create":
		return noPositional(req, args, map[string]*string{
			"--title": &req.Title, "--body": &req.Body,
			"--base": &req.Base, "--head": &req.Head,
		}, map[string]*bool{"--draft": &req.Draft})
	}
	return usageErr("unknown command")
}

func noPositional(req *Request, args []string, strs map[string]*string, bools map[string]*bool) error {
	pos, err := flagLoop(req, args, strs, bools)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return usageErr("unexpected argument " + pos[0])
	}
	return nil
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add command.go command_test.go
git commit -m "feat: 공통 명령 문법 parser 추가"
```

---

### Task 4: provider별 argv 변환

**Files:**
- Modify: `command.go` (파일 끝에 추가)
- Test: `command_test.go` (파일 끝에 추가)

**Interfaces:**
- Consumes: `Request`(Task 3), `RepoURL`/`Provider`(Task 1)
- Produces:
  - `type Invocation struct { Bin string; Args []string; Env []string }`
  - `func Translate(req Request, r RepoURL, p Provider, teaLogin string) (Invocation, error)`
  - 규칙: `pull`/`push`는 Translate에 오지 않는다 (main에서 git 직행).

- [ ] **Step 1: 실패하는 테스트 작성** — `command_test.go`에 추가

```go
func TestTranslate(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	ghe := RepoURL{Host: "ghe.corp.com", Owner: "o", Name: "r"}
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
		// ---- GitHub ----
		{name: "gh issue list",
			req:  Request{Resource: "issue", Action: "list", State: "all", Limit: "5"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"issue", "list", "-R", "github.com/o/r", "--state", "all", "--limit", "5"}}},
		{name: "gh pr view",
			req:  Request{Resource: "pr", Action: "view", Number: "7"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "view", "7", "-R", "github.com/o/r"}}},
		{name: "gh pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"pr", "create", "-R", "github.com/o/r", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"}}},
		{name: "gh repo list on GHE",
			req:  Request{Resource: "repo", Action: "list", Limit: "3"},
			repo: ghe, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "list", "--limit", "3"}, Env: []string{"GH_HOST=ghe.corp.com"}}},
		{name: "gh repo view",
			req:  Request{Resource: "repo", Action: "view"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "view", "https://github.com/o/r"}}},
		{name: "gh repo create",
			req:  Request{Resource: "repo", Action: "create", Public: true, Description: "d"},
			repo: ghe, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "create", "o/r", "--public", "--description", "d"}, Env: []string{"GH_HOST=ghe.corp.com"}}},
		{name: "gh clone",
			req:  Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", CloneDir: "dst"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"repo", "clone", "https://github.com/o/r", "dst"}}},

		// ---- GitLab ----
		{name: "glab issue list closed",
			req:  Request{Resource: "issue", Action: "list", State: "closed", Limit: "5"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "list", "--repo", "https://git.example.com/grp/sub/p", "--closed", "--per-page", "5"}}},
		{name: "glab pr list all",
			req:  Request{Resource: "pr", Action: "list", State: "all"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "list", "--repo", "https://git.example.com/grp/sub/p", "--all"}}},
		{name: "glab pr list open은 flag 없음",
			req:  Request{Resource: "pr", Action: "list", State: "open"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "list", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"mr", "create", "--repo", "https://git.example.com/grp/sub/p", "--title", "t", "--description", "b", "--target-branch", "main", "--source-branch", "f", "--draft"}}},
		{name: "glab issue view",
			req:  Request{Resource: "issue", Action: "view", Number: "9"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"issue", "view", "9", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab repo list",
			req:  Request{Resource: "repo", Action: "list", Limit: "7"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "list", "--per-page", "7"}, Env: []string{"GITLAB_HOST=git.example.com"}}},
		{name: "glab repo create",
			req:  Request{Resource: "repo", Action: "create", Private: true, Description: "d"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"repo", "create", "grp/sub/p", "--private", "--description", "d"}, Env: []string{"GITLAB_HOST=git.example.com"}}},

		// ---- Gitea ----
		{name: "tea issue list",
			req:  Request{Resource: "issue", Action: "list", State: "all", Limit: "5"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"issues", "list", "--login", "corp", "--repo", "o/r", "--state", "all", "--limit", "5"}}},
		{name: "tea pr view",
			req:  Request{Resource: "pr", Action: "view", Number: "3"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"pulls", "3", "--login", "corp", "--repo", "o/r"}}},
		{name: "tea pr create",
			req:  Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"pulls", "create", "--login", "corp", "--repo", "o/r", "--title", "t", "--description", "b", "--base", "main", "--head", "f", "--draft"}}},
		{name: "tea repo view",
			req:  Request{Resource: "repo", Action: "view"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "o/r", "--login", "corp"}}},
		{name: "tea repo create public은 --private 없음",
			req:  Request{Resource: "repo", Action: "create", Public: true, Description: "d"},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--description", "d"}}},
		{name: "tea repo create private",
			req:  Request{Resource: "repo", Action: "create", Private: true},
			repo: te, p: Tea, tea: "corp",
			want: Invocation{Bin: "tea", Args: []string{"repos", "create", "--login", "corp", "--owner", "o", "--name", "r", "--private"}}},
		{name: "tea clone은 login 불필요",
			req:  Request{Resource: "repo", Action: "clone", CloneURL: "https://gitea.example.com/o/r"},
			repo: te, p: Tea,
			want: Invocation{Bin: "tea", Args: []string{"clone", "https://gitea.example.com/o/r"}}},
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
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./... -run TestTranslate -v`
Expected: FAIL — `undefined: Invocation`, `undefined: Translate`

- [ ] **Step 3: 최소 구현** — `command.go` 끝에 추가

```go
// Invocation은 실행할 자식 process다.
type Invocation struct {
	Bin  string
	Args []string
	Env  []string // os.Environ()에 덧붙일 KEY=VALUE
}

func Translate(req Request, r RepoURL, p Provider, teaLogin string) (Invocation, error) {
	switch p {
	case GH:
		return ghInvocation(req, r), nil
	case GLab:
		return glabInvocation(req, r), nil
	case Tea:
		return teaInvocation(req, r, teaLogin), nil
	}
	return Invocation{}, usageErr("unknown provider " + string(p))
}

func appendKV(args []string, flag, val string) []string {
	if val == "" {
		return args
	}
	return append(args, flag, val)
}

func ghInvocation(req Request, r RepoURL) Invocation {
	inv := Invocation{Bin: "gh"}
	if r.Host != "github.com" {
		inv.Env = []string{"GH_HOST=" + r.Host}
	}
	target := []string{"-R", r.Host + "/" + r.Slug()}
	if r.Host == "github.com" {
		target = []string{"-R", r.Slug()}
	}
	res := map[string]string{"repo": "repo", "issue": "issue", "pr": "pr"}[req.Resource]
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV([]string{"repo", "list"}, "--limit", req.Limit)
	case "repo view":
		inv.Args = []string{"repo", "view", r.HTTPS()}
		inv.Env = nil // URL이 host를 지정하므로 GH_HOST 불필요
	case "repo create":
		inv.Args = []string{"repo", "create", r.Slug(), visFlag(req)}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
	case "repo clone":
		inv.Args = []string{"repo", "clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
		inv.Env = nil
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		inv.Args = appendKV(inv.Args, "--state", req.State)
		inv.Args = appendKV(inv.Args, "--limit", req.Limit)
		inv.Env = nil
	case "issue view", "pr view":
		inv.Args = append([]string{res, "view", req.Number}, target...)
		inv.Env = nil
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--body", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--base", req.Base)
			inv.Args = appendKV(inv.Args, "--head", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
		inv.Env = nil
	}
	return inv
}

func visFlag(req Request) string {
	if req.Private {
		return "--private"
	}
	return "--public"
}

func glabInvocation(req Request, r RepoURL) Invocation {
	inv := Invocation{Bin: "glab"}
	target := []string{"--repo", r.HTTPS()}
	res := map[string]string{"repo": "repo", "issue": "issue", "pr": "mr"}[req.Resource]
	stateFlags := map[string]string{"closed": "--closed", "all": "--all"}
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV([]string{"repo", "list"}, "--per-page", req.Limit)
		inv.Env = []string{"GITLAB_HOST=" + r.Host}
	case "repo view":
		inv.Args = []string{"repo", "view", r.HTTPS()}
	case "repo create":
		inv.Args = []string{"repo", "create", r.Slug(), visFlag(req)}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
		inv.Env = []string{"GITLAB_HOST=" + r.Host}
	case "repo clone":
		inv.Args = []string{"repo", "clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		if f := stateFlags[req.State]; f != "" {
			inv.Args = append(inv.Args, f)
		}
		inv.Args = appendKV(inv.Args, "--per-page", req.Limit)
	case "issue view", "pr view":
		inv.Args = append([]string{res, "view", req.Number}, target...)
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--description", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--target-branch", req.Base)
			inv.Args = appendKV(inv.Args, "--source-branch", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
	}
	return inv
}

func teaInvocation(req Request, r RepoURL, login string) Invocation {
	inv := Invocation{Bin: "tea"}
	auth := []string{"--login", login}
	target := append(append([]string{}, auth...), "--repo", r.Slug())
	res := map[string]string{"repo": "repos", "issue": "issues", "pr": "pulls"}[req.Resource]
	switch req.Resource + " " + req.Action {
	case "repo list":
		inv.Args = appendKV(append([]string{"repos", "list"}, auth...), "--limit", req.Limit)
	case "repo view":
		inv.Args = append([]string{"repos", r.Slug()}, auth...)
	case "repo create":
		inv.Args = append([]string{"repos", "create"}, auth...)
		inv.Args = append(inv.Args, "--owner", r.Owner, "--name", r.Name)
		if req.Private {
			inv.Args = append(inv.Args, "--private")
		}
		inv.Args = appendKV(inv.Args, "--description", req.Description)
	case "repo clone":
		inv.Args = []string{"clone", req.CloneURL}
		if req.CloneDir != "" {
			inv.Args = append(inv.Args, req.CloneDir)
		}
	case "issue list", "pr list":
		inv.Args = append([]string{res, "list"}, target...)
		inv.Args = appendKV(inv.Args, "--state", req.State)
		inv.Args = appendKV(inv.Args, "--limit", req.Limit)
	case "issue view", "pr view":
		inv.Args = append([]string{res, req.Number}, target...)
	case "issue create", "pr create":
		inv.Args = append([]string{res, "create"}, target...)
		inv.Args = appendKV(inv.Args, "--title", req.Title)
		inv.Args = appendKV(inv.Args, "--description", req.Body)
		if req.Resource == "pr" {
			inv.Args = appendKV(inv.Args, "--base", req.Base)
			inv.Args = appendKV(inv.Args, "--head", req.Head)
			if req.Draft {
				inv.Args = append(inv.Args, "--draft")
			}
		}
	}
	return inv
}
```

주의: `gh repo create`는 github.com이 아닐 때 `GH_HOST` env가 유지되어야 한다.
`ghInvocation`은 처음에 Env를 설정하고 URL 기반 case에서만 `nil`로 되돌린다.
테스트 기대값과 정확히 일치하는지 case별로 확인한다.

- [ ] **Step 4: 통과 확인**

Run: `go test ./... -v`
Expected: PASS (Translate 20 cases 포함)

- [ ] **Step 5: Commit**

```bash
git add command.go command_test.go
git commit -m "feat: provider별 argv 변환 추가"
```

---

### Task 5: remote 선택과 provider 판별

**Files:**
- Modify: `route.go` (파일 끝에 추가)
- Test: `route_test.go` (파일 끝에 추가)

**Interfaces:**
- Consumes: `Config`/`SaveProvider`(Task 2), `Provider`(Task 1)
- Produces:
  - `var runOut func(name string, args ...string) (string, error)` — 테스트 교체점
  - `var lookPath = exec.LookPath` — 테스트 교체점
  - `var stdin io.Reader = os.Stdin` — 테스트 교체점
  - `func CurrentRemoteURL() (string, error)`
  - `func DetectProvider(host string, cfg *Config, interactive bool) (Provider, error)`
  - `func teaLoginName(host string) string` — 없으면 `""`
  - `func stdinIsTerminal() bool`

- [ ] **Step 1: 실패하는 테스트 작성** — `route_test.go`에 추가

```go
import 블록에 추가: "fmt", "strings"

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
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./... -run 'TestCurrentRemote|TestDetect|TestTeaLogin' -v`
Expected: FAIL — `undefined: runOut` 등 compile error

- [ ] **Step 3: 최소 구현** — `route.go`에 추가 (import에 `encoding/json`, `errors`, `io`, `os`, `os/exec` 추가)

```go
// 테스트 교체점: 자식 실행/탐색/입력.
var (
	runOut = func(name string, args ...string) (string, error) {
		out, err := exec.Command(name, args...).Output()
		return strings.TrimSpace(string(out)), err
	}
	lookPath           = exec.LookPath
	stdin    io.Reader = os.Stdin
)

// CurrentRemoteURL: branch upstream → origin → 유일 remote 순서.
func CurrentRemoteURL() (string, error) {
	// commit이 없는 저장소에서는 rev-parse가 실패한다. upstream 단계만
	// 건너뛰고 origin/유일 remote fallback은 계속 시도한다.
	if branch, err := runOut("git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil && branch != "" {
		if remote, _ := runOut("git", "config", "--get", "branch."+branch+".remote"); remote != "" {
			if u, _ := runOut("git", "remote", "get-url", remote); u != "" {
				return u, nil
			}
		}
	}
	if u, _ := runOut("git", "remote", "get-url", "origin"); u != "" {
		return u, nil
	}
	remotes, err := runOut("git", "remote")
	if err != nil {
		return "", errors.New("not a git repository (use --repo <URL>)")
	}
	names := strings.Fields(remotes)
	if len(names) == 1 {
		if u, _ := runOut("git", "remote", "get-url", names[0]); u != "" {
			return u, nil
		}
	}
	return "", errors.New("cannot pick a remote (use --repo <URL>)")
}

var defaultProviders = map[string]Provider{
	"github.com": GH,
	"gitlab.com": GLab,
	"gitea.com":  Tea,
}

func hasBin(p Provider) bool { _, err := lookPath(string(p)); return err == nil }

func ghHasLogin(host string) bool {
	out, err := runOut("gh", "auth", "status", "--hostname", host, "--json", "hosts")
	if err != nil {
		return false
	}
	var v struct {
		Hosts map[string]json.RawMessage `json:"hosts"`
	}
	if json.Unmarshal([]byte(out), &v) != nil {
		return false
	}
	for h := range v.Hosts {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func glabHasLogin(host string) bool {
	_, err := runOut("glab", "auth", "status", "--hostname", host)
	return err == nil
}

// teaLoginName은 host에 맞는 tea login 이름을 돌려준다. 없으면 "".
func teaLoginName(host string) string {
	out, err := runOut("tea", "logins", "list", "--output", "json")
	if err != nil {
		return ""
	}
	var logins []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if json.Unmarshal([]byte(out), &logins) != nil {
		return ""
	}
	for _, l := range logins {
		if u, err := url.Parse(l.URL); err == nil && strings.EqualFold(u.Hostname(), host) {
			return l.Name
		}
	}
	return ""
}

func hasLogin(p Provider, host string) bool {
	switch p {
	case GH:
		return ghHasLogin(host)
	case GLab:
		return glabHasLogin(host)
	case Tea:
		return teaLoginName(host) != ""
	}
	return false
}

// DetectProvider: 기본 domain → 저장값 → 로그인 후보 → 선택/오류.
func DetectProvider(host string, cfg *Config, interactive bool) (Provider, error) {
	if p, ok := defaultProviders[host]; ok {
		return p, nil
	}
	if saved := Provider(cfg.Hosts[host]); saved == GH || saved == GLab || saved == Tea {
		if hasBin(saved) && hasLogin(saved, host) {
			return saved, nil
		}
	}
	var cands []Provider
	for _, p := range []Provider{GH, GLab, Tea} {
		if hasBin(p) && hasLogin(p, host) {
			cands = append(cands, p)
		}
	}
	switch len(cands) {
	case 0:
		return "", fmt.Errorf(
			"no logged-in CLI for %s\nrun one of:\n  gh auth login --hostname %s\n  glab auth login --hostname %s\n  tea login add",
			host, host, host)
	case 1:
		return cands[0], nil
	}
	if !interactive {
		names := make([]string, len(cands))
		for i, p := range cands {
			names[i] = string(p)
		}
		return "", fmt.Errorf(
			"multiple providers match %s: %s\nset it once in %s:\n  {\"hosts\": {\"%s\": \"%s\"}}",
			host, strings.Join(names, ", "), ConfigPath(), host, names[0])
	}
	p, err := chooseProvider(host, cands)
	if err != nil {
		return "", err
	}
	if err := SaveProvider(host, p); err != nil {
		return "", err
	}
	return p, nil
}

func chooseProvider(host string, cands []Provider) (Provider, error) {
	fmt.Fprintf(os.Stderr, "Multiple providers match %s:\n", host)
	for i, p := range cands {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, p)
	}
	fmt.Fprint(os.Stderr, "Choose provider: ")
	var n int
	if _, err := fmt.Fscanln(stdin, &n); err != nil || n < 1 || n > len(cands) {
		return "", errors.New("invalid choice")
	}
	return cands[n-1], nil
}

func stdinIsTerminal() bool {
	f, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./... -v`
Expected: PASS (전체)

- [ ] **Step 5: Commit**

```bash
git add route.go route_test.go
git commit -m "feat: remote 선택과 provider 판별 추가"
```

---

### Task 6: main 조립과 자식 실행

**Files:**
- Create: `main.go`
- Test: `command_test.go`에 `plan()` 테스트 추가

**Interfaces:**
- Consumes: 앞의 모든 함수
- Produces:
  - `func plan(req Request) (Invocation, error)` — 라우팅 결정 전체
  - `func run(args []string) int` — exit code 결정
  - `func execChild(inv Invocation) int`

- [ ] **Step 1: 실패하는 테스트 작성** — `command_test.go`에 추가

`command_test.go` import 블록에 `"strings"`를 추가한다 (`strings.Contains` 사용).

```go
func TestPlanPullGoesToGit(t *testing.T) {
	req := Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase"}}
	inv, err := plan(req)
	if err != nil {
		t.Fatal(err)
	}
	want := Invocation{Bin: "git", Args: []string{"pull", "--rebase"}}
	if !reflect.DeepEqual(inv, want) {
		t.Errorf("plan = %+v, want %+v", inv, want)
	}
}

func TestPlanUsesRemoteWhenNoRepoFlag(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"git rev-parse --abbrev-ref HEAD": "main",
		"git remote get-url origin":       "git@github.com:o/r.git",
	})
	inv, err := plan(Request{Resource: "issue", Action: "list"})
	if err != nil {
		t.Fatal(err)
	}
	want := Invocation{Bin: "gh", Args: []string{"issue", "list", "-R", "o/r"}}
	if !reflect.DeepEqual(inv, want) {
		t.Errorf("plan = %+v, want %+v", inv, want)
	}
}

func TestPlanTeaNeedsLogin(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"tea logins list --output json": `[]`,
	})
	_, err := plan(Request{Resource: "issue", Action: "list",
		RepoFlag: "https://gitea.com/o/r"})
	if err == nil || !strings.Contains(err.Error(), "tea login add") {
		t.Errorf("tea login 안내 기대, got %v", err)
	}
}

func TestRunExitCodes(t *testing.T) {
	if code := run([]string{"unknown"}); code != 2 {
		t.Errorf("usage error = %d, want 2", code)
	}
	fakeExec(t, map[string]string{}) // git 실패
	if code := run([]string{"view"}); code != 1 {
		t.Errorf("route error = %d, want 1", code)
	}
}
```

주의: `TestPlanUsesRemoteWhenNoRepoFlag`의 fake는 `git config --get branch.main.remote`
응답이 없으므로 upstream 단계는 실패하고 origin fallback을 탄다. 의도된 동작이다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./... -run 'TestPlan|TestRunExit' -v`
Expected: FAIL — `undefined: plan`, `undefined: run`

- [ ] **Step 3: 최소 구현** — `main.go`

```go
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

const usage = `usage:
  gg [--repo <URL>] <command>

repository (repo 생략 가능):
  gg list [--limit N]
  gg view
  gg --repo <URL> create (--public|--private) [--description TEXT]
  gg clone <URL> [DIR]
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
```

- [ ] **Step 4: 통과 확인 + build**

Run: `go test ./... -v && go build -o gg.exe .`
Expected: PASS + `gg.exe` 생성. `./gg.exe`만 치면 usage와 exit 2.

- [ ] **Step 5: `.gitignore` 추가 후 commit**

`.gitignore`:

```text
gg
gg.exe
```

```bash
git add main.go command_test.go .gitignore
git commit -m "feat: gg entrypoint와 자식 실행 추가"
```

---

### Task 7: end-to-end smoke 테스트와 cross-build

**Files:**
- Test: `e2e_test.go`

**Interfaces:**
- Consumes: 완성된 `gg` binary (테스트가 직접 build)
- Produces: 실제 process 수준 검증. fake `git`/`gh`/`glab` 실행 파일이 받은 argv를 파일로 기록한다.

- [ ] **Step 1: e2e 테스트 작성** — `e2e_test.go`

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildGG는 gg를 임시 폴더에 build한다.
func buildGG(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gg")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build 실패: %v\n%s", err, out)
	}
	return bin
}

// writeFakeBin은 argv를 LOG 파일에 기록하는 fake 실행 파일을 만든다.
func writeFakeBin(t *testing.T, dir, name, logFile string) {
	t.Helper()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name+".cmd")
		body = "@echo off\r\necho " + name + " %* >> \"" + logFile + "\"\r\nexit /b 0\r\n"
	} else {
		path = filepath.Join(dir, name)
		body = "#!/bin/sh\necho \"" + name + " $@\" >> \"" + logFile + "\"\nexit 0\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// runGG는 fake PATH + 임시 GG_HOME으로 gg를 실행한다.
func runGG(t *testing.T, bin, fakeDir, workDir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GG_HOME="+t.TempDir(),
	)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("실행 실패: %v\n%s", err, out)
	}
	return string(out), code
}

func readLog(t *testing.T, logFile string) string {
	t.Helper()
	data, _ := os.ReadFile(logFile)
	return strings.TrimSpace(string(data))
}

// 실제 git으로 remote를 가진 임시 저장소를 만든다.
func tempRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", remoteURL},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestE2EGitHubIssueList(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "issue", "list", "--limit", "3")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "gh issue list -R o/r --limit 3") {
		t.Errorf("gh argv = %q", got)
	}
}

func TestE2EPullPassesThroughToGit(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "git", logFile)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "pull", "--rebase", "origin", "main")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "git pull --rebase origin main") {
		t.Errorf("git argv = %q", got)
	}
}

func TestE2EChildExitCodePassthrough(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	// exit 7로 끝나는 fake gh
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(fakeDir, "gh.cmd")
		body = "@echo off\r\nexit /b 7\r\n"
	} else {
		path = filepath.Join(fakeDir, "gh")
		body = "#!/bin/sh\nexit 7\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://github.com/o/r.git")

	_, code := runGG(t, bin, fakeDir, repo, "issue", "list")
	if code != 7 {
		t.Errorf("exit = %d, want 7", code)
	}
}

func TestE2ESavedConfigRoutesWithoutPrompt(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://git.example.com/g/p.git")

	ggHome := t.TempDir()
	cfg := `{"hosts":{"git.example.com":"glab"}}`
	if err := os.WriteFile(filepath.Join(ggHome, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "issue", "list")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GG_HOME="+ggHome,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("exit: %v\n%s", err, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "glab issue list --repo https://git.example.com/g/p") {
		t.Errorf("glab argv = %q", got)
	}
}
```

주의: `TestE2ESavedConfigRoutesWithoutPrompt`에서 fake glab은
`glab auth status --hostname ...` probe도 exit 0으로 응답하므로 저장값 검증을 통과한다.
probe 호출과 실제 명령이 같은 log 파일에 쌓이므로 `Contains`로 확인한다.

- [ ] **Step 2: e2e 실행**

Run: `go test ./... -run TestE2E -v`
Expected: PASS (4 tests). Windows에서 `.cmd` fake가 실행되는지 확인.

- [ ] **Step 3: 전체 검증**

```bash
go vet ./...
go test ./...
go build ./...
```

Expected: 모두 성공, warning 없음.

- [ ] **Step 4: cross-build 확인** (PowerShell)

```powershell
$env:GOOS="linux";   go build -o out/gg-linux .
$env:GOOS="darwin";  go build -o out/gg-darwin .
$env:GOOS="windows"; go build -o out/gg.exe .
Remove-Item Env:GOOS
```

Expected: 세 binary 모두 생성. `out/`은 commit하지 않는다 (`.gitignore`에 `out/` 추가).

- [ ] **Step 5: 수동 smoke 테스트** (실제 CLI 사용)

```bash
# 이 저장소는 remote가 없으므로 --repo로 확인
./gg.exe --repo https://github.com/cli/cli issue list --limit 3
./gg.exe pull
```

Expected: 첫 명령은 gh 로그인 상태에 따라 gh의 실제 출력 또는 gh 로그인 오류를 그대로 표시.
둘째 명령은 `git pull` 결과(이 저장소는 remote 없음 오류)를 그대로 표시.

- [ ] **Step 6: Commit**

```bash
git add e2e_test.go .gitignore
git commit -m "test: end-to-end 라우팅 smoke 테스트 추가"
```

---

## Self-Review 체크

1. **Spec coverage:** 명령 문법(Task 3), remote 선택(Task 5), URL parser(Task 1), provider 판별과 대화형 선택 저장(Task 5), 설정 fallback/atomic write(Task 2), argv 변환표(Task 4), git pull/push 직행과 stdio/exit code 계약(Task 6), e2e/cross-build(Task 7). exit code 0/1/2/127 매핑은 Task 6.
2. **Placeholder:** 모든 step에 실제 코드 포함. TBD 없음.
3. **Type consistency:** `Request`/`Invocation`/`RepoURL`/`Provider`/`Config` 필드와 함수 서명이 task 간 동일.
