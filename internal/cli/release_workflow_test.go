package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type releaseWorkflow struct {
	On          map[string]workflowEvent `yaml:"on"`
	Permissions map[string]string        `yaml:"permissions"`
	Jobs        map[string]workflowJob   `yaml:"jobs"`
}

type workflowEvent struct {
	Branches []string `yaml:"branches"`
	Tags     []string `yaml:"tags"`
	Paths    []string `yaml:"paths"`
	Inputs   map[string]struct {
		Required bool   `yaml:"required"`
		Type     string `yaml:"type"`
	} `yaml:"inputs"`
}

type workflowJob struct {
	If          string            `yaml:"if"`
	RunsOn      string            `yaml:"runs-on"`
	Permissions map[string]string `yaml:"permissions"`
	Strategy    struct {
		Matrix struct {
			OS []string `yaml:"os"`
		} `yaml:"matrix"`
	} `yaml:"strategy"`
	Steps []workflowStep `yaml:"steps"`
}

type workflowStep struct {
	ID              string            `yaml:"id"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	Shell           string            `yaml:"shell"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	Env             map[string]string `yaml:"env"`
	With            map[string]string `yaml:"with"`
}

func readReleaseWorkflow(t *testing.T, name string) releaseWorkflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var workflow releaseWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("%s YAML 파싱 실패: %v", name, err)
	}
	return workflow
}

func TestReleaseWorkflowGate(t *testing.T) {
	ci := readReleaseWorkflow(t, "ci.yml")
	release := readReleaseWorkflow(t, "release.yml")
	if !slices.Contains(ci.On["push"].Branches, "main") || len(ci.On["push"].Tags) != 0 {
		t.Errorf("CI push 이벤트 = %+v; main 검증이 필요하며 tag 게시를 담당하지 않아야 합니다", ci.On["push"])
	}
	if _, ok := ci.On["pull_request"]; !ok {
		t.Error("CI에 pull_request 검증이 없습니다")
	}
	if _, ok := ci.Jobs["release"]; ok {
		t.Error("CI가 release 잡을 중복으로 실행합니다")
	}
	for _, runner := range []string{"ubuntu-latest", "windows-latest"} {
		if !slices.Contains(ci.Jobs["verify"].Strategy.Matrix.OS, runner) {
			t.Errorf("CI 검증 매트릭스에 %s가 없습니다", runner)
		}
	}
	for _, command := range []string{"go vet ./...", "go test ./..."} {
		if !slices.ContainsFunc(ci.Jobs["verify"].Steps, func(step workflowStep) bool { return strings.Contains(step.Run, command) }) {
			t.Errorf("CI에 %q 검증이 없습니다", command)
		}
	}
	if !slices.Contains(release.On["push"].Branches, "main") || !slices.Contains(release.On["push"].Tags, "v*") {
		t.Errorf("Release push 이벤트 = %+v; main 병합과 v* tag를 처리해야 합니다", release.On["push"])
	}
	if input := release.On["workflow_dispatch"].Inputs["tag"]; !input.Required || input.Type != "string" {
		t.Errorf("수동 릴리즈 tag 입력 = %+v; 필수 string 입력이어야 합니다", input)
	}
	if _, ok := release.On["pull_request"]; ok {
		t.Error("PR 이벤트가 릴리즈 작업을 직접 실행합니다")
	}
	if release.Permissions["contents"] != "read" {
		t.Errorf("workflow 기본 contents 권한 = %q; want read", release.Permissions["contents"])
	}
	// Job conditions are the event contract; step labels and layout are not.
	for _, tc := range []struct{ job, condition string }{
		{"prepare-release-pr", "github.event_name == 'workflow_dispatch'"},
		{"tag-release", "github.event_name == 'push' && github.ref_type == 'branch' && github.ref == format('refs/heads/{0}', github.event.repository.default_branch)"},
		{"release", "github.event_name == 'push' && github.ref_type == 'tag'"},
	} {
		job, ok := release.Jobs[tc.job]
		if !ok {
			t.Errorf("%s 잡이 없습니다", tc.job)
			continue
		}
		if strings.Join(strings.Fields(job.If), " ") != tc.condition {
			t.Errorf("%s 이벤트 조건 = %q; want %q", tc.job, job.If, tc.condition)
		}
		if job.Permissions["contents"] != "write" {
			t.Errorf("%s contents 권한 = %q; want write", tc.job, job.Permissions["contents"])
		}
		if !slices.ContainsFunc(job.Steps, func(step workflowStep) bool {
			return strings.HasPrefix(step.Uses, "actions/checkout@") && step.With["fetch-depth"] == "0"
		}) {
			t.Errorf("%s 잡이 tag/부모 commit 검증에 필요한 전체 Git 기록을 가져오지 않습니다", tc.job)
		}
	}
	if release.Jobs["prepare-release-pr"].Permissions["pull-requests"] != "write" {
		t.Error("릴리즈 PR 준비 잡에 pull-requests: write 권한이 없습니다")
	}
}

// Git for Windows supplies a native Bash. A PATH entry pointing to WSL's
// bash.exe cannot execute scripts against the Windows test fixture directly.
func releaseWorkflowBash(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		path, err := exec.LookPath("bash")
		if err != nil {
			t.Fatalf("릴리즈 workflow 검증에 Bash가 필요합니다: %v", err)
		}
		return path
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(git))
	roots := []string{root}
	if base := strings.ToLower(filepath.Base(root)); base == "mingw64" || base == "mingw32" {
		roots = append(roots, filepath.Dir(root))
	}
	for _, root := range roots {
		for _, candidate := range []string{filepath.Join(root, "bin", "bash.exe"), filepath.Join(root, "usr", "bin", "bash.exe")} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	t.Fatalf("Git for Windows의 Bash를 찾을 수 없습니다 (git: %s)", git)
	return ""
}

type releaseFixture struct {
	dir         string
	env         []string
	values      map[string]string
	release     string
	immutable   string
	buildFailed bool
}

func newReleaseFixture(t *testing.T) *releaseFixture {
	t.Helper()
	dir := t.TempDir()
	f := &releaseFixture{
		dir: dir,
		env: append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "gitconfig"),
			"GIT_AUTHOR_NAME=Release Test", "GIT_AUTHOR_EMAIL=release@example.invalid",
			"GIT_COMMITTER_NAME=Release Test", "GIT_COMMITTER_EMAIL=release@example.invalid",
			"GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=file"),
		values: map[string]string{
			"github.ref_name": "v0.1.0", "github.event.repository.default_branch": "main",
			"github.repository": "test/gg", "github.workspace": releaseBashPath(dir),
			"github.token": "test-publish-token", "secrets.RELEASE_ADMIN_TOKEN": "test-admin-token",
		},
		release: "absent", immutable: "true",
	}
	f.git(t, "init", "--initial-branch=main")
	f.git(t, "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "initial")
	f.git(t, "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "release: v0.1.0")
	return f
}

func (f *releaseFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = f.dir, f.env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func (f *releaseFixture) remote(t *testing.T) {
	t.Helper()
	// Every origin is a local bare repository. Workflow Git commands never use a
	// real hosting service, including the stale-tip and failed-lookup cases.
	origin := filepath.Join(t.TempDir(), "origin.git")
	f.git(t, "init", "--bare", origin)
	f.git(t, "remote", "add", "origin", filepath.ToSlash(origin))
	f.git(t, "push", "origin", "HEAD:refs/heads/main")
}

var workflowExpression = regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)

func releaseBashPath(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" && len(path) > 2 && path[1] == ':' {
		return "/" + strings.ToLower(path[:1]) + path[2:]
	}
	return path
}

func (f *releaseFixture) expand(t *testing.T, value string) string {
	t.Helper()
	return workflowExpression.ReplaceAllStringFunc(value, func(expression string) string {
		key := strings.TrimSpace(workflowExpression.FindStringSubmatch(expression)[1])
		result, ok := f.values[key]
		if !ok {
			t.Fatalf("workflow 테스트에서 지원하지 않는 표현식: %s", expression)
		}
		return result
	})
}

func (f *releaseFixture) run(t *testing.T, bash string, job workflowJob) (string, error) {
	t.Helper()
	bin := filepath.Join(f.dir, "fake-bin")
	calls := filepath.Join(f.dir, "calls")
	for _, dir := range []string{bin, calls} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, script := range map[string]string{"gh": releaseFakeGH, "go": releaseFakeGo} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var output strings.Builder
	for index, step := range job.Steps {
		if step.Run == "" {
			continue // checkout and setup-go are supplied by the isolated fixture.
		}
		if step.If != "" || step.ContinueOnError || (step.Shell != "" && step.Shell != "bash") {
			t.Fatalf("workflow 실행 조건의 테스트 지원을 갱신해야 합니다: step %d: %+v", index, step)
		}
		outputPath := filepath.Join(f.dir, fmt.Sprintf("output-%d", index))
		script := "export PATH=\"$GG_WORKFLOW_BIN:$PATH\"\n" + f.expand(t, step.Run)
		// GitHub's unspecified Linux shell uses bash -e. Only an explicit
		// shell: bash enables pipefail; adding it here would hide pipeline bugs.
		shellArgs := []string{"-e"}
		if step.Shell == "bash" {
			shellArgs = []string{"--noprofile", "--norc", "-e", "-o", "pipefail"}
		}
		cmd := exec.Command(bash, append(shellArgs, "-c", script)...)
		cmd.Dir = f.dir
		cmd.Env = append(slices.Clone(f.env),
			"GG_WORKFLOW_BIN="+releaseBashPath(bin), "GG_WORKFLOW_CALLS="+releaseBashPath(calls),
			"GG_WORKFLOW_RELEASE="+f.release, "GG_WORKFLOW_IMMUTABLE="+f.immutable,
			fmt.Sprintf("GG_WORKFLOW_BUILD_FAILED=%t", f.buildFailed),
			"GITHUB_OUTPUT="+releaseBashPath(outputPath))
		for key, value := range step.Env {
			cmd.Env = append(cmd.Env, key+"="+f.expand(t, value))
		}
		data, err := cmd.CombinedOutput()
		output.Write(data)
		if err != nil {
			return output.String(), fmt.Errorf("workflow step %d: %w", index, err)
		}
		if data, err := os.ReadFile(outputPath); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				key, value, ok := strings.Cut(line, "=")
				if !ok || step.ID == "" {
					t.Fatalf("잘못된 workflow 출력: %q (step ID %q)", line, step.ID)
				}
				f.values["steps."+step.ID+".outputs."+key] = value
			}
		}
	}
	return output.String(), nil
}

func TestReleaseWorkflowPublishing(t *testing.T) {
	bash := releaseWorkflowBash(t)
	job := readReleaseWorkflow(t, "release.yml").Jobs["release"]
	// Keep Git and Bash process fan-out bounded on CI runners.
	slots := make(chan struct{}, 2)
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *releaseFixture)
		valid  bool
	}{
		{name: "valid annotated release", valid: true},
		{name: "invalid version", mutate: func(t *testing.T, f *releaseFixture) { f.values["github.ref_name"] = "vbroken" }},
		{name: "missing tag", mutate: func(t *testing.T, f *releaseFixture) { f.git(t, "tag", "-d", "v0.1.0") }},
		{name: "lightweight tag", mutate: func(t *testing.T, f *releaseFixture) {
			f.git(t, "tag", "-d", "v0.1.0")
			f.git(t, "tag", "v0.1.0")
		}},
		{name: "tag points elsewhere", mutate: func(t *testing.T, f *releaseFixture) {
			f.git(t, "tag", "-fa", "v0.1.0", "HEAD~1", "-m", "previous commit")
		}},
		{name: "missing default branch", mutate: func(t *testing.T, f *releaseFixture) { f.values["github.event.repository.default_branch"] = "" }},
		{name: "default branch lookup fails", mutate: func(t *testing.T, f *releaseFixture) {
			f.git(t, "remote", "set-url", "origin", filepath.Join(f.dir, "missing-origin"))
		}},
		{name: "stale default branch tip", mutate: func(t *testing.T, f *releaseFixture) {
			f.git(t, "push", "--force", "origin", "HEAD~1:refs/heads/main")
		}},
		{name: "wrong release subject", mutate: func(t *testing.T, f *releaseFixture) {
			f.git(t, "commit", "--amend", "--allow-empty", "-m", "ordinary commit")
			f.git(t, "tag", "-fa", "v0.1.0", "-m", "release")
			f.git(t, "push", "--force", "origin", "HEAD:refs/heads/main")
		}},
		{name: "release changes tree", mutate: func(t *testing.T, f *releaseFixture) {
			if err := os.WriteFile(filepath.Join(f.dir, "changed"), []byte("new content"), 0o644); err != nil {
				t.Fatal(err)
			}
			f.git(t, "add", "changed")
			f.git(t, "commit", "--amend", "--no-edit")
			f.git(t, "tag", "-fa", "v0.1.0", "-m", "release")
			f.git(t, "push", "--force", "origin", "HEAD:refs/heads/main")
		}},
		{name: "release has no parent", mutate: func(t *testing.T, f *releaseFixture) {
			tree := f.git(t, "rev-parse", "HEAD^{tree}")
			commit := f.git(t, "commit-tree", tree, "-m", "release: v0.1.0")
			f.git(t, "reset", "--hard", commit)
			f.git(t, "tag", "-fa", "v0.1.0", "-m", "release")
			f.git(t, "push", "--force", "origin", "HEAD:refs/heads/main")
		}},
		{name: "release exists", mutate: func(t *testing.T, f *releaseFixture) { f.release = "exists" }},
		{name: "release API fails", mutate: func(t *testing.T, f *releaseFixture) { f.release = "error" }},
		{name: "admin token missing", mutate: func(t *testing.T, f *releaseFixture) { f.values["secrets.RELEASE_ADMIN_TOKEN"] = "" }},
		{name: "immutable API fails", mutate: func(t *testing.T, f *releaseFixture) { f.immutable = "error" }},
		{name: "immutable disabled", mutate: func(t *testing.T, f *releaseFixture) { f.immutable = "false" }},
		{name: "immutable unknown", mutate: func(t *testing.T, f *releaseFixture) { f.immutable = "null" }},
		{name: "build fails", mutate: func(t *testing.T, f *releaseFixture) { f.buildFailed = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			slots <- struct{}{}
			t.Cleanup(func() { <-slots })
			f := newReleaseFixture(t)
			f.remote(t)
			f.git(t, "-c", "tag.gpgsign=false", "tag", "-a", "v0.1.0", "-m", "Release v0.1.0")
			if tc.mutate != nil {
				tc.mutate(t, f)
			}
			output, err := f.run(t, bash, job)
			if (err == nil) != tc.valid {
				t.Errorf("workflow 결과 = %v; 성공 기대 = %t\n%s", err, tc.valid, output)
			}
			calls, err := os.ReadDir(filepath.Join(f.dir, "calls"))
			if err != nil {
				t.Fatal(err)
			}
			var publishes [][]string
			for _, call := range calls {
				data, err := os.ReadFile(filepath.Join(f.dir, "calls", call.Name()))
				if err != nil {
					t.Fatal(err)
				}
				args := strings.Split(string(bytes.TrimSuffix(data, []byte{0})), "\x00")
				if len(args) >= 2 && args[0] == "release" && args[1] == "create" {
					publishes = append(publishes, args)
				}
			}
			if !tc.valid {
				if len(publishes) != 0 {
					t.Errorf("실패한 릴리즈가 게시를 호출했습니다: %q", publishes)
				}
				return
			}
			want := [][]string{{"release", "create", "v0.1.0", "dist/fixture.tar.gz", "dist/fixture.tar.gz.sha256", "--title", "v0.1.0", "--generate-notes", "--verify-tag"}}
			if !reflect.DeepEqual(publishes, want) {
				t.Errorf("게시 호출 = %q; want %q", publishes, want)
			}
		})
	}
}

const releaseFakeGH = `#!/usr/bin/env bash
set -eu
printf '%s\0' "$@" > "$(mktemp "$GG_WORKFLOW_CALLS/gh.XXXXXX")"
if [[ "$1" == api && "$2" == repos/test/gg/releases/tags/* ]]; then
  [[ "$GH_TOKEN" == test-publish-token ]] || exit 91
  case "$GG_WORKFLOW_RELEASE" in
    absent) echo 'gh: Not Found (HTTP 404)' >&2; exit 1 ;;
    exists) echo '{}' ;;
    error) echo 'gh: Forbidden (HTTP 403)' >&2; exit 1 ;;
    *) exit 92 ;;
  esac
elif [[ "$1" == api && "$2" == --method && "$3" == GET && "$4" == repos/test/gg/immutable-releases && "$5" == --jq && "$6" == .enabled ]]; then
  [[ "$GH_TOKEN" == test-admin-token ]] || exit 93
  if [[ "$GG_WORKFLOW_IMMUTABLE" == error ]]; then
    echo 'gh: service unavailable (HTTP 503)' >&2
    exit 1
  fi
  echo "$GG_WORKFLOW_IMMUTABLE"
elif [[ "$1" == release && "$2" == create ]]; then
  [[ "$GH_TOKEN" == test-publish-token ]] || exit 94
else
  echo "Unexpected gh operation: $*" >&2
  exit 95
fi
`

// Packaging is verified separately with real binaries in TestBuildAndPackageRelease.
const releaseFakeGo = `#!/usr/bin/env bash
set -eu
[[ "$1" == test ]] || exit 96
shift
run_filter=''
package=''
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -v) shift ;;
    -run) [[ "$#" -ge 2 ]] || exit 96; run_filter="$2"; shift 2 ;;
    -run=*) run_filter="${1#-run=}"; shift ;;
    ./internal/cli) package="$1"; shift ;;
    *) echo "Unexpected go test argument: $1" >&2; exit 96 ;;
  esac
done
[[ "$run_filter" == '^TestBuildAndPackageRelease$' && "$package" == ./internal/cli ]] || exit 96
[[ "$GG_RELEASE_VERSION" == v0.1.0 && "$GG_RELEASE_OUT_DIR" == */dist ]] || exit 97
[[ "$GG_WORKFLOW_BUILD_FAILED" == false ]] || exit 1
mkdir -p "$GG_RELEASE_OUT_DIR"
touch "$GG_RELEASE_OUT_DIR/fixture.tar.gz" "$GG_RELEASE_OUT_DIR/fixture.tar.gz.sha256"
`
