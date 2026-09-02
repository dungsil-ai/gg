package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
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
	return runGGWithHome(t, bin, fakeDir, workDir, t.TempDir(), args...)
}

// runGGStreams는 실제 gg process의 stdout과 stderr를 따로 읽는다.
func runGGStreams(t *testing.T, bin, workDir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := ggCommand(t, bin, "", workDir, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := "stdout: " + stdout.String() + "\nstderr: " + stderr.String()
	return stdout.String(), stderr.String(), processExitCode(t, err, output)
}

func ggCommand(t *testing.T, bin, fakeDir, workDir string, args ...string) *exec.Cmd {
	t.Helper()
	return ggCommandWithHomeAndPath(bin, workDir, t.TempDir(), pathWithFakeBin(fakeDir), args...)
}

func runGGWithHome(t *testing.T, bin, fakeDir, workDir, ggHome string, args ...string) (string, int) {
	t.Helper()
	return runGGWithPath(t, bin, workDir, ggHome, pathWithFakeBin(fakeDir), args...)
}

func pathWithFakeBin(fakeDir string) string {
	pathValue := os.Getenv("PATH")
	if fakeDir != "" {
		pathValue = fakeDir + string(os.PathListSeparator) + pathValue
	}
	return pathValue
}

func runGGWithoutProviderCLIs(t *testing.T, bin, workDir, ggHome string, args ...string) (string, int) {
	t.Helper()
	return runGGWithPath(t, bin, workDir, ggHome, t.TempDir(), args...)
}

func runGGWithPath(t *testing.T, bin, workDir, ggHome, pathValue string, args ...string) (string, int) {
	t.Helper()
	cmd := ggCommandWithHomeAndPath(bin, workDir, ggHome, pathValue, args...)
	out, err := cmd.CombinedOutput()
	return string(out), processExitCode(t, err, string(out))
}

func ggCommandWithHomeAndPath(bin, workDir, ggHome, pathValue string, args ...string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"PATH="+pathValue,
		"GG_HOME="+ggHome,
	)
	return cmd
}

func processExitCode(t *testing.T, err error, output string) int {
	t.Helper()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	t.Fatalf("실행 실패: %v\n%s", err, output)
	return 0
}

func TestE2EProviderSettingListEmpty(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	ggHome := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	for _, name := range []string{"gh", "glab", "tea"} {
		writeFakeBin(t, fakeDir, name, logFile)
	}

	out, code := runGGWithHome(t, bin, fakeDir, t.TempDir(), ggHome, "config", "list")
	if code != 0 || out != "No provider settings.\n" {
		t.Fatalf("exit %d, output %q", code, out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Fatalf("provider CLI should not run, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(ggHome, "config.json.lock")); !os.IsNotExist(err) {
		t.Fatalf("empty config list should not create a lock file, stat error = %v", err)
	}
}

func TestE2EProviderSettingSetReplacesWithoutPromptAndListsSorted(t *testing.T) {
	bin := buildGG(t)
	ggHome := t.TempDir()
	workDir := t.TempDir()

	commands := [][]string{
		{"config", "set", "Git.B.Example:8443", "glab"},
		{"config", "set", "git.b.example", "glab"},
		{"config", "set", "git.b.example:443", "tea"},
		{"config", "set", "A.Example", "gh"},
	}
	for _, args := range commands {
		out, code := runGGWithoutProviderCLIs(t, bin, workDir, ggHome, args...)
		if code != 0 || out != "" {
			t.Fatalf("gg %v: exit %d, output %q", args, code, out)
		}
	}

	out, code := runGGWithoutProviderCLIs(t, bin, workDir, ggHome, "config", "list")
	if code != 0 {
		t.Fatalf("list: exit %d, output %q", code, out)
	}
	wantFields := []string{"HOST", "PROVIDER", "a.example", "gh", "git.b.example", "tea"}
	if got := strings.Fields(out); !slices.Equal(got, wantFields) {
		t.Fatalf("list fields = %q, want %q", got, wantFields)
	}

	data, err := os.ReadFile(filepath.Join(ggHome, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	wantHosts := map[string]string{"a.example": "gh", "git.b.example": "tea"}
	if !maps.Equal(saved.Hosts, wantHosts) {
		t.Fatalf("saved hosts = %v, want %v", saved.Hosts, wantHosts)
	}
}

func TestE2EProviderSettingUnsetIsIdempotentAndSavesResult(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	ggHome := t.TempDir()
	workDir := t.TempDir()

	for _, args := range [][]string{
		{"config", "unset", "gitlab.com"},
		{"config", "unset", "Missing.Example:443"},
		{"config", "set", "Git.Example", "glab"},
		{"config", "unset", "git.example:8443"},
		{"config", "unset", "git.example"},
	} {
		out, code := runGGWithHome(t, bin, fakeDir, workDir, ggHome, args...)
		if code != 0 || out != "" {
			t.Fatalf("gg %v: exit %d, output %q", args, code, out)
		}
	}

	out, code := runGGWithHome(t, bin, fakeDir, workDir, ggHome, "config", "list")
	if code != 0 || out != "No provider settings.\n" {
		t.Fatalf("list: exit %d, output %q", code, out)
	}
	data, err := os.ReadFile(filepath.Join(ggHome, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Hosts) != 0 {
		t.Fatalf("saved hosts = %v, want empty", saved.Hosts)
	}
}

func TestE2EProviderSettingRejectsInvalidAndDefaultDomainChanges(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	for _, name := range []string{"gh", "glab", "tea"} {
		writeFakeBin(t, fakeDir, name, logFile)
	}
	ggHome := t.TempDir()
	workDir := t.TempDir()

	commands := [][]string{
		{"config", "set", "https://git.example.com", "gh"},
		{"config", "set", "git.example.com/group/repo", "gh"},
		{"config", "set", "git.example.com", "github"},
		{"config", "set", "GitHub.com:443", "gh"},
	}
	for _, args := range commands {
		out, code := runGGWithHome(t, bin, fakeDir, workDir, ggHome, args...)
		if code != 2 || !strings.Contains(out, "gg:") {
			t.Fatalf("gg %v: exit %d, output %q", args, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(ggHome, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("invalid commands should not create config, stat error = %v", err)
	}
	if got := readLog(t, logFile); got != "" {
		t.Fatalf("provider CLI should not run, got %q", got)
	}
}

func TestE2EProviderSettingMutationsPreserveBrokenConfig(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	brokenConfigs := []struct {
		name    string
		content string
	}{
		{name: "malformed JSON", content: `{"hosts":{"git.example.com":"gh"}`},
		{name: "secret fields", content: `{"hosts":{},"token":"secret","login":"user","repository":"https://git.example.com/o/r"}`},
	}
	for _, brokenConfig := range brokenConfigs {
		t.Run(brokenConfig.name, func(t *testing.T) {
			ggHome := t.TempDir()
			configPath := filepath.Join(ggHome, "config.json")
			broken := []byte(brokenConfig.content)
			if err := os.WriteFile(configPath, broken, 0o600); err != nil {
				t.Fatal(err)
			}

			for _, args := range [][]string{
				{"config", "set", "other.example", "tea"},
				{"config", "unset", "git.example.com"},
			} {
				out, code := runGGWithHome(t, bin, fakeDir, t.TempDir(), ggHome, args...)
				if code != 1 || !strings.Contains(out, "broken config") {
					t.Fatalf("gg %v: exit %d, output %q", args, code, out)
				}
				got, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(got, broken) {
					t.Fatalf("gg %v changed broken config to %q", args, got)
				}
			}
		})
	}
}

func TestE2EProviderSettingConcurrentSetsKeepEveryHost(t *testing.T) {
	bin := buildGG(t)
	ggHome := t.TempDir()
	emptyPath := t.TempDir()
	workDir := t.TempDir()
	const count = 32
	const baseCount = 20000
	baseHosts := make(map[string]string, baseCount)
	for i := 0; i < baseCount; i++ {
		baseHosts[fmt.Sprintf("base-%04d.example", i)] = "tea"
	}
	baseData, err := json.Marshal(Config{Hosts: baseHosts})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ggHome, "config.json"), baseData, 0o600); err != nil {
		t.Fatal(err)
	}

	commands := make([]*exec.Cmd, 0, count)
	for i := 0; i < count; i++ {
		host := fmt.Sprintf("git-%02d.example", i)
		cmd := exec.Command(bin, "config", "set", host, "gh")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "PATH="+emptyPath, "GG_HOME="+ggHome)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("concurrent set failed: %v", err)
		}
	}

	data, err := os.ReadFile(filepath.Join(ggHome, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Hosts) != baseCount+count {
		t.Fatalf("saved %d hosts, want %d", len(saved.Hosts), baseCount+count)
	}
}

func TestE2EProviderSettingListWaitsForMutationLock(t *testing.T) {
	bin := buildGG(t)
	ggHome := t.TempDir()
	configPath := filepath.Join(ggHome, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"hosts":{"git.example.com":"gh"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	lock := flock.New(configPath + ".lock")
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			_ = lock.Unlock()
		}
	}()

	cmd := exec.Command(bin, "config", "list")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "PATH="+t.TempDir(), "GG_HOME="+ggHome)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		t.Fatalf("config list finished while mutation lock was held: %v, output %q", err, output.String())
	case <-time.After(750 * time.Millisecond):
	}

	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	locked = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("config list failed after mutation lock was released: %v, output %q", err, output.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("config list did not finish after mutation lock was released")
	}
	if got := strings.Fields(output.String()); !slices.Equal(got, []string{"HOST", "PROVIDER", "git.example.com", "gh"}) {
		t.Fatalf("list fields = %q", got)
	}
}

func TestE2EProviderSettingIPv6RoundTrip(t *testing.T) {
	bin := buildGG(t)
	ggHome := t.TempDir()
	workDir := t.TempDir()

	out, code := runGGWithoutProviderCLIs(t, bin, workDir, ggHome, "config", "set", "[2001:DB8::1]:2222", "gh")
	if code != 0 || out != "" {
		t.Fatalf("set: exit %d, output %q", code, out)
	}
	out, code = runGGWithoutProviderCLIs(t, bin, workDir, ggHome, "config", "list")
	if code != 0 || !slices.Equal(strings.Fields(out), []string{"HOST", "PROVIDER", "2001:db8::1", "gh"}) {
		t.Fatalf("list: exit %d, output %q", code, out)
	}
	out, code = runGGWithoutProviderCLIs(t, bin, workDir, ggHome, "config", "unset", "2001:db8::1")
	if code != 0 || out != "" {
		t.Fatalf("unset: exit %d, output %q", code, out)
	}
}

func readLog(t *testing.T, logFile string) string {
	t.Helper()
	data, _ := os.ReadFile(logFile)
	return strings.TrimSpace(string(data))
}

func setupFakeGH(t *testing.T) (bin, fakeDir, logFile string) {
	t.Helper()
	bin = buildGG(t)
	fakeDir = t.TempDir()
	logFile = filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	return bin, fakeDir, logFile
}

// 실제 git으로 remote를 가진 임시 저장소를 만든다.
func tempRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "remote", "add", "origin", remoteURL)
	return dir
}

func assertGGHelp(t *testing.T, bin string, args, wants []string) {
	t.Helper()
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), args...)
	if code != 0 {
		t.Errorf("gg %v exit = %d, want 0", args, code)
	}
	if stderr != "" {
		t.Errorf("gg %v stderr = %q, want empty", args, stderr)
	}
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Errorf("gg %v stdout에 %q 없음:\n%s", args, want, stdout)
		}
	}
}

func TestE2ETopLevelHelp(t *testing.T) {
	bin := buildGG(t)
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		assertGGHelp(t, bin, args, []string{"Usage:", "Commands:", "commit", "issue", "pr", "config", "--repo", "--remote"})
	}
}

func TestE2ETopLevelHelpListsSupportedNestedHelpForms(t *testing.T) {
	bin := buildGG(t)
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg --help = stderr %q, exit %d; want empty stderr, exit 0", stderr, code)
	}
	for _, want := range []string{"gg config --help", "gg issue --help", "gg issue list --help", "gg pr create --help"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("gg --help stdout에 %q 없음:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "gg <command> --help") {
		t.Errorf("gg --help stdout advertises unsupported generic command help:\n%s", stdout)
	}
}

func assertGGHelpOmits(t *testing.T, bin string, args []string, unsupported string) {
	t.Helper()
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), args...)
	if code != 0 || stderr != "" {
		t.Fatalf("gg %v = stderr %q, exit %d; want empty stderr, exit 0", args, stderr, code)
	}
	if strings.Contains(stdout, unsupported) {
		t.Errorf("gg %v stdout advertises unsupported %s:\n%s", args, unsupported, stdout)
	}
}

func TestE2EConfigHelpDoesNotAdvertiseRemoteFlag(t *testing.T) {
	bin := buildGG(t)
	assertGGHelpOmits(t, bin, []string{"config", "--help"}, "--remote")
}

func TestE2ENestedHelpDoesNotAdvertiseUnsupportedShortFlag(t *testing.T) {
	bin := buildGG(t)
	for _, args := range [][]string{{"config", "--help"}, {"issue", "--help"}, {"issue", "list", "--help"}, {"pr", "create", "--help"}} {
		assertGGHelpOmits(t, bin, args, "-h, --help")
	}
}

func TestE2ENestedHelp(t *testing.T) {
	bin := buildGG(t)
	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"config", "--help"}, []string{"Usage:", "config list", "config set", "config unset", "Flags:", "--help"}},
		{[]string{"issue", "--help"}, []string{"Usage:", "list", "view", "create", "--repo", "--remote", "--help"}},
		{[]string{"issue", "list", "--help"}, []string{"Usage:", "--state", "--limit", "--repo", "--remote", "--help"}},
		{[]string{"pr", "create", "--help"}, []string{"Usage:", "--title", "--body", "--base", "--head", "--draft", "--repo", "--remote", "--help"}},
	}
	for _, tt := range tests {
		assertGGHelp(t, bin, tt.args, tt.want)
	}
}

func TestE2EUnknownCommandUsesStderrAndExitTwo(t *testing.T) {
	bin := buildGG(t)
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "unknown")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "gg: unknown command unknown") {
		t.Errorf("stderr에 unknown command 오류 없음:\n%s", stderr)
	}
}

func TestE2EVersion(t *testing.T) {
	localBin := buildGG(t)
	releaseBin := filepath.Join(t.TempDir(), "gg")
	if runtime.GOOS == "windows" {
		releaseBin += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags", "-X main.version=v0.1.0", "-o", releaseBin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("release build 실패: %v\n%s", err, out)
	}

	for _, tt := range []struct {
		name string
		bin  string
		want string
	}{
		{"local version", localBin, "gg dev\n"},
		{"release version", releaseBin, "gg v0.1.0\n"},
	} {
		for _, args := range [][]string{{"version"}, {"--version"}} {
			stdout, stderr, code := runGGStreams(t, tt.bin, t.TempDir(), args...)
			if code != 0 || stdout != tt.want || stderr != "" {
				t.Errorf("%s %v = stdout %q, stderr %q, exit %d; want stdout %q, empty stderr, exit 0", tt.name, args, stdout, stderr, code, tt.want)
			}
		}
	}

	stdout, stderr, code := runGGStreams(t, localBin, t.TempDir(), "-v")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "unknown command -v") {
		t.Errorf("-v = stdout %q, stderr %q, exit %d; want usage error", stdout, stderr, code)
	}
}

func tempRepoWithUpstream(t *testing.T) string {
	t.Helper()
	repo := tempRepo(t, "https://github.com/o/origin.git")
	gitIn(t, repo, "remote", "add", "upstream", "https://github.com/o/upstream.git")
	return repo
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
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
	if !strings.Contains(got, "gh issue list -R github.com/o/r --limit 3") {
		t.Errorf("gh argv = %q", got)
	}
}

func TestE2ERepositoryContextRemoteBeforeAndAfterCommand(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	for _, args := range [][]string{
		{"--remote", "upstream", "issue", "list"},
		{"issue", "list", "--remote", "upstream"},
	} {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", args, code, out)
		}
		if got := readLog(t, logFile); got != "gh issue list -R github.com/o/upstream" {
			t.Errorf("gg %v argv = %q", args, got)
		}
	}
}

func TestE2EAutomaticRepositoryContextSelectionOrder(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	t.Run("branch upstream", func(t *testing.T) {
		repo := tempRepoWithUpstream(t)
		gitIn(t, repo, "config", "user.name", "gg test")
		gitIn(t, repo, "config", "user.email", "gg-test@example.com")
		gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "test")
		branch := gitIn(t, repo, "branch", "--show-current")
		gitIn(t, repo, "config", "branch."+branch+".remote", "upstream")

		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh issue list -R github.com/o/upstream" {
			t.Errorf("argv = %q", got)
		}
	})

	t.Run("origin", func(t *testing.T) {
		repo := tempRepo(t, "https://github.com/o/origin.git")
		gitIn(t, repo, "remote", "add", "other", "https://github.com/o/other.git")

		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh issue list -R github.com/o/origin" {
			t.Errorf("argv = %q", got)
		}
	})

	t.Run("single remote", func(t *testing.T) {
		repo := t.TempDir()
		gitIn(t, repo, "init", "-q")
		gitIn(t, repo, "remote", "add", "solo", "https://github.com/o/solo.git")

		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); got != "gh issue list -R github.com/o/solo" {
			t.Errorf("argv = %q", got)
		}
	})
}

func TestE2ERepositoryContextShowsAvailableNamesForMissingRemote(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	out, code := runGG(t, bin, fakeDir, repo, "--remote", "missing", "issue", "list")
	if code != 1 {
		t.Fatalf("exit = %d, want 1: %s", code, out)
	}
	for _, want := range []string{`remote "missing" not found`, "origin", "upstream"} {
		if !strings.Contains(out, want) {
			t.Errorf("output에 %q 필요: %s", want, out)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("child command should not run, got %q", got)
	}
}

func TestE2ERepositoryContextFlagConflictIsUsageError(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	out, code := runGG(t, bin, fakeDir, t.TempDir(),
		"--repo", "https://github.com/o/r", "--remote", "upstream", "issue", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2: %s", code, out)
	}
	for _, want := range []string{"cannot be used together", "Usage:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output에 %q 필요: %s", want, out)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("child command should not run, got %q", got)
	}
}

func TestE2EEmptyRemoteNameInRepositoryContextIsUsageError(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/origin.git")

	for _, args := range [][]string{
		{"--remote", "", "issue", "list"},
		{"issue", "list", "--remote", ""},
	} {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "--remote needs a name") {
			t.Errorf("gg %v output: %s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got %q", args, got)
		}
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

func TestE2ECommitAlwaysDisablesSigning(t *testing.T) {
	bin := buildGG(t)
	repo := tempRepo(t, "https://github.com/o/r.git")
	gitIn(t, repo, "config", "user.name", "gg test")
	gitIn(t, repo, "config", "user.email", "gg-test@example.com")
	gitIn(t, repo, "config", "commit.gpgSign", "true")
	gitIn(t, repo, "config", "gpg.program", filepath.Join(t.TempDir(), "missing-gpg"))

	out, code := runGG(t, bin, "", repo, "commit", "--allow-empty", "-m", "unsigned")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if got := gitIn(t, repo, "log", "-1", "--format=%s"); got != "unsigned" {
		t.Errorf("commit subject = %q, want unsigned", got)
	}
}

func TestE2ECommitPassesThroughToGitWithNoGpgSign(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "git", logFile)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "commit", "-m", "msg")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "git commit --no-gpg-sign -m msg") {
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

func TestE2ECloneHTTPBlockedByDefault(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "clone", "http://github.com/o/r.git")
	if code == 0 {
		t.Fatalf("HTTP clone should be blocked: %s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Fatalf("child command should not run, got log: %q", got)
	}
}

func TestE2ECloneHTTPAllowedWithWarning(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "clone", "http://github.com/o/r.git", "--allow-insecure-http")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "warning: allowing insecure HTTP clone") {
		t.Fatalf("warning expected, got: %s", out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "gh repo clone http://github.com/o/r.git") {
		t.Fatalf("gh argv = %q", got)
	}
}

func TestE2ECloneKeepsSSHNonStandardPort(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "clone", "ssh://git@github.com:2222/o/r.git")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "gh repo clone ssh://git@github.com:2222/o/r.git") {
		t.Fatalf("gh argv = %q", got)
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
