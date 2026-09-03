package cli

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
	out, err := exec.Command("go", "build", "-o", bin, "../..").CombinedOutput()
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

// buildGitPassthroughProbe는 전달받은 argv를 JSON Lines로 기록하고 표준
// 스트림을 중계 검증용으로 되돌린 뒤 종료 코드 23을 반환하는 fake git을 만든다.
func buildGitPassthroughProbe(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "main.go")
	const source = `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	if logPath := os.Getenv("GG_GIT_LOG"); logPath != "" {
		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(99)
		}
		if err := json.NewEncoder(file).Encode(os.Args[1:]); err != nil {
			file.Close()
			fmt.Fprint(os.Stderr, err)
			os.Exit(99)
		}
		file.Close()
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(99)
	}
	fmt.Printf("stdout:%s", input)
	fmt.Fprintf(os.Stderr, "stderr:%s", input)
	os.Exit(23)
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "git")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, sourcePath).CombinedOutput()
	if err != nil {
		t.Fatalf("fake git build 실패: %v\n%s", err, out)
	}
	return bin
}

func readGitPassthroughCalls(t *testing.T, logFile string) [][]string {
	t.Helper()
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var args []string
		if err := json.Unmarshal([]byte(line), &args); err != nil {
			t.Fatalf("fake git argv decode 실패: %v", err)
		}
		calls = append(calls, args)
	}
	return calls
}

// writeFakeVersionBin은 --version 출력과 종료 코드를 정한 fake 실행 파일을 만든다.
func writeFakeVersionBin(t *testing.T, dir, name, stdout, stderr string, exitCode int) {
	t.Helper()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name+".cmd")
		body = "@echo off\r\n" +
			"if not \"%~1\"==\"--version\" exit /b 99\r\n" +
			"if not \"%~2\"==\"\" exit /b 99\r\n"
		if stdout != "" {
			for _, line := range strings.Split(stdout, "\n") {
				body += "echo " + line + "\r\n"
			}
		}
		if stderr != "" {
			for _, line := range strings.Split(stderr, "\n") {
				body += "echo " + line + " 1>&2\r\n"
			}
		}
		body += fmt.Sprintf("exit /b %d\r\n", exitCode)
	} else {
		path = filepath.Join(dir, name)
		body = "#!/bin/sh\n" +
			"if [ \"$#\" -ne 1 ] || [ \"$1\" != \"--version\" ]; then\n" +
			"  exit 99\n" +
			"fi\n"
		if stdout != "" {
			for _, line := range strings.Split(stdout, "\n") {
				body += "printf '%s\\n' '" + line + "'\n"
			}
		}
		if stderr != "" {
			for _, line := range strings.Split(stderr, "\n") {
				body += "printf '%s\\n' '" + line + "' >&2\n"
			}
		}
		body += fmt.Sprintf("exit %d\n", exitCode)
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
		assertGGHelp(t, bin, args, []string{"Usage:", "gg [flags] <command>", "gg <supported-git-command> [git args]", "gg repo --help", "Commands:", "commit", "issue", "pr", "alias: mr", "config", "--repo", "--remote", "--version         gg 버전만 표시", "-v, -verison      단독 사용 시 gg와 설치된 git, gh, glab, tea 버전을 표시"})
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
	for _, args := range [][]string{
		{"config", "--help"}, {"issue", "--help"}, {"issue", "list", "--help"}, {"pr", "create", "--help"},
		{"repo", "--help"}, {"repo", "list", "--help"}, {"repo", "commit", "--help"}, {"pr", "merge", "--help"},
	} {
		assertGGHelpOmits(t, bin, args, "-h, --help")
	}
}

func TestE2ENestedHelp(t *testing.T) {
	bin := buildGG(t)
	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"config", "--help"}, []string{"Usage:", "list", "List Provider 설정", "set", "unset", "Flags:", "--help"}},
		{[]string{"issue", "--help"}, []string{"Usage:", "list", "view", "create", "comment", "close", "reopen", "--repo", "--remote", "--help"}},
		{[]string{"issue", "list", "--help"}, []string{"Usage:", "--state", "--limit", "--repo", "--remote", "--help"}},
		{[]string{"issue", "comment", "--help"}, []string{"Usage:", "comment <number>", "--body", "--repo", "--remote", "--help"}},
		{[]string{"label", "--help"}, []string{"Usage:", "list", "create", "--repo", "--remote", "--help"}},
		{[]string{"label", "list", "--help"}, []string{"Usage:", "--limit", "--repo", "--remote", "--help"}},
		{[]string{"label", "create", "--help"}, []string{"Usage:", "--name", "--color", "--description", "--repo", "--remote", "--help"}},
		{[]string{"issue", "close", "--help"}, []string{"Usage:", "close <number>", "--repo", "--remote", "--help"}},
		{[]string{"issue", "reopen", "--help"}, []string{"Usage:", "reopen <number>", "--repo", "--remote", "--help"}},
		{[]string{"pr", "create", "--help"}, []string{"Usage:", "--title", "--body", "--base", "--head", "--draft", "--repo", "--remote", "--help"}},
	}
	for _, tt := range tests {
		assertGGHelp(t, bin, tt.args, tt.want)
	}
}

// TestE2EAllActionHelp는 gg가 소유한 resource와 action의 --help가 stdout으로
// 나오고 usage와 정의된 flag를 모두 표시하는지 본다.
func TestE2EAllActionHelp(t *testing.T) {
	bin := buildGG(t)
	for _, name := range commandOrder {
		rd := commandDefs[name]

		stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), name, "--help")
		if code != 0 || stderr != "" {
			t.Errorf("gg %s --help = stderr %q, exit %d", name, stderr, code)
		}
		for _, want := range []string{rd.desc, rd.usage, "Commands:", "Flags:"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("gg %s --help stdout에 %q 없음:\n%s", name, want, stdout)
			}
		}

		for i := range rd.actions {
			ad := &rd.actions[i]
			if name == "repo" && isGitPassthroughAction(ad.name) {
				continue
			}
			stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), name, ad.name, "--help")
			if code != 0 || stderr != "" {
				t.Errorf("gg %s %s --help = stderr %q, exit %d", name, ad.name, stderr, code)
				continue
			}
			wants := []string{ad.summary + ".", ad.usage, "Flags:", "--help"}
			for _, f := range actionFlags(ad) {
				if f.arg != "" {
					wants = append(wants, f.name+" "+f.arg)
				} else {
					wants = append(wants, f.name)
				}
			}
			for _, want := range wants {
				if !strings.Contains(stdout, want) {
					t.Errorf("gg %s %s --help stdout에 %q 없음:\n%s", name, ad.name, want, stdout)
				}
			}
		}
	}
}

// TestE2ERepoOmittedHelpMatchesRepoForm은 repo를 생략한 명령의 help가
// gg repo <action> --help와 같은 출력을 내는지 본다.
func TestE2ERepoOmittedHelpMatchesRepoForm(t *testing.T) {
	bin := buildGG(t)
	for alias := range helpAliases {
		omitted, _, code := runGGStreams(t, bin, t.TempDir(), alias, "--help")
		prefixed, _, code2 := runGGStreams(t, bin, t.TempDir(), "repo", alias, "--help")
		if code != 0 || code2 != 0 {
			t.Fatalf("gg %s --help exits %d, gg repo %s --help exits %d", alias, code, alias, code2)
		}
		if omitted != prefixed {
			t.Errorf("gg %s --help와 gg repo %s --help 출력이 다름:\n%q\n%q", alias, alias, omitted, prefixed)
		}
	}
}

// TestE2ECommitAliasPassesHelpToGit는 repo 생략 commit의 --help가
// gg help가 아니라 git으로 전달되는지 본다.
func TestE2ECommitAliasPassesHelpToGit(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "git", logFile)

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "commit", "--help")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if got := readLog(t, logFile); !strings.Contains(got, "git commit --no-gpg-sign --help") {
		t.Errorf("git argv = %q, want git commit --no-gpg-sign --help", got)
	}
	if strings.Contains(out, "Usage:") {
		t.Errorf("gg commit --help가 gg help를 출력함:\n%s", out)
	}
}

// TestE2EPullPushHelpDoesNotRunGit은 repo 생략 pull/push의 --help가
// git을 실행하지 않고 gg의 단계별 help를 stdout에 출력하는지 본다.
func TestE2EPullPushHelpDoesNotRunGit(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "git", logFile)

	for _, args := range [][]string{{"pull", "--help"}, {"repo", "pull", "--help"}, {"push", "--help"}} {
		stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), args...)
		if code != 0 || stderr != "" {
			t.Fatalf("gg %v = stderr %q, exit %d", args, stderr, code)
		}
		for _, want := range []string{"Usage:", "Flags:", "--help"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("gg %v stdout에 %q 없음:\n%s", args, want, stdout)
			}
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("git should not run, got %q", got)
	}
}

// TestE2EHelpAfterContextFlags는 저장소 문맥 flag 뒤에 --help가 와도
// 해당 명령의 help가 나오는지 본다.
func TestE2EHelpAfterContextFlags(t *testing.T) {
	bin := buildGG(t)
	for _, args := range [][]string{
		{"--repo", "https://github.com/o/r", "issue", "list", "--help"},
		{"--explain", "pr", "--help"},
		{"--remote", "origin", "list", "--help"},
	} {
		stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), args...)
		if code != 0 || stderr != "" {
			t.Fatalf("gg %v = stderr %q, exit %d", args, stderr, code)
		}
		if !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "Flags:") {
			t.Errorf("gg %v stdout에 help 없음:\n%s", args, stdout)
		}
	}
}

// TestE2EUnknownFlagUsesStderrAndExitTwo는 알 수 없는 flag의
// 오류 출력과 exit code 2 계약을 본다.
func TestE2EUnknownFlagUsesStderrAndExitTwo(t *testing.T) {
	bin := buildGG(t)
	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "issue", "list", "--wat")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"gg: unknown flag --wat", "Usage:"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr에 %q 없음:\n%s", want, stderr)
		}
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
	build := exec.Command("go", "build", "-ldflags", "-X main.version=v0.1.0", "-o", releaseBin, "../..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("release build 실패: %v\n%s", err, out)
	}

	builds := []struct {
		name    string
		bin     string
		version string
	}{
		{"local", localBin, "dev"},
		{"release", releaseBin, "v0.1.0"},
	}
	for _, build := range builds {
		for _, args := range [][]string{{"version"}, {"--version"}} {
			stdout, stderr, code := runGGStreams(t, build.bin, t.TempDir(), args...)
			want := "gg " + build.version + "\n"
			if code != 0 || stdout != want || stderr != "" {
				t.Errorf("%s %v = stdout %q, stderr %q, exit %d; want stdout %q, empty stderr, exit 0", build.name, args, stdout, stderr, code, want)
			}
		}
	}

	runAggregate := func(t *testing.T, bin, pathValue string, args ...string) (string, string, int) {
		t.Helper()
		cmd := ggCommandWithHomeAndPath(bin, t.TempDir(), t.TempDir(), pathValue, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		output := "stdout: " + stdout.String() + "\nstderr: " + stderr.String()
		return stdout.String(), stderr.String(), processExitCode(t, err, output)
	}

	versionCases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name: "all successful tools show stdout in order",
			setup: func(t *testing.T, dir string) {
				writeFakeVersionBin(t, dir, "git", "git version 2.47.0\ngit build metadata", "git warning", 0)
				writeFakeVersionBin(t, dir, "gh", "gh version 2.62.0\ngh build metadata", "gh warning", 0)
				writeFakeVersionBin(t, dir, "glab", "glab version 1.49.0", "", 0)
				writeFakeVersionBin(t, dir, "tea", "tea version 0.9.2", "", 0)
			},
			want: "git version 2.47.0\ngit build metadata\ngh version 2.62.0\ngh build metadata\nglab version 1.49.0\ntea version 0.9.2\n",
		},
		{
			name: "missing and failing tools are skipped while later tools run",
			setup: func(t *testing.T, dir string) {
				writeFakeVersionBin(t, dir, "git", "git version 2.47.0", "", 0)
				writeFakeVersionBin(t, dir, "glab", "glab version 1.49.0", "glab failed", 1)
				writeFakeVersionBin(t, dir, "tea", "tea version 0.9.2", "", 0)
			},
			want: "git version 2.47.0\ntea version 0.9.2\n",
		},
		{
			name: "successful stderr-only tool is skipped",
			setup: func(t *testing.T, dir string) {
				writeFakeVersionBin(t, dir, "git", "git version 2.47.0", "", 0)
				writeFakeVersionBin(t, dir, "gh", "", "gh warning", 0)
				writeFakeVersionBin(t, dir, "glab", "glab version 1.49.0", "", 0)
				writeFakeVersionBin(t, dir, "tea", "tea version 0.9.2", "", 0)
			},
			want: "git version 2.47.0\nglab version 1.49.0\ntea version 0.9.2\n",
		},
		{
			name: "all tools missing",
			want: "",
		},
	}
	for _, build := range builds {
		for _, versionCase := range versionCases {
			for _, args := range [][]string{{"-verison"}, {"-v"}} {
				t.Run(build.name+"/"+versionCase.name+"/"+args[0], func(t *testing.T) {
					fakeDir := t.TempDir()
					if versionCase.setup != nil {
						versionCase.setup(t, fakeDir)
					}
					stdout, stderr, code := runAggregate(t, build.bin, fakeDir, args...)
					want := "gg " + build.version + "\n" + versionCase.want
					if code != 0 || stdout != want || stderr != "" {
						t.Errorf("%s %v = stdout %q, stderr %q, exit %d; want stdout %q, empty stderr, exit 0", versionCase.name, args, stdout, stderr, code, want)
					}
				})
			}
		}
	}

	stdout, stderr, code := runGGStreams(t, localBin, t.TempDir(), "-v", "extra")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "gg: unknown command -v") {
		t.Errorf("-v extra = stdout %q, stderr %q, exit %d; want usage error", stdout, stderr, code)
	}

	gitDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, gitDir, "git", logFile)
	out, code := runGG(t, localBin, gitDir, t.TempDir(), "commit", "-v")
	if code != 0 || out != "" {
		t.Errorf("gg commit -v = output %q, exit %d; want empty output, exit 0", out, code)
	}
	if got := readLog(t, logFile); got != "git commit --no-gpg-sign -v" {
		t.Errorf("git argv = %q, want %q", got, "git commit --no-gpg-sign -v")
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

func TestE2EPRCommandAliasMatchesCanonicalCommand(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	t.Run("provider invocation", func(t *testing.T) {
		canonicalArgs := []string{"pr", "list", "--state", "closed", "--limit", "3"}
		aliasArgs := []string{"mr", "list", "--state", "closed", "--limit", "3"}
		canonicalOut, canonicalCode := runGG(t, bin, fakeDir, repo, canonicalArgs...)
		if canonicalCode != 0 {
			t.Fatalf("gg %v: exit %d, output %q", canonicalArgs, canonicalCode, canonicalOut)
		}
		canonicalLog := readLog(t, logFile)
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		aliasOut, aliasCode := runGG(t, bin, fakeDir, repo, aliasArgs...)
		aliasLog := readLog(t, logFile)
		if aliasOut != canonicalOut || aliasCode != canonicalCode || aliasLog != canonicalLog {
			t.Errorf("gg %v = output %q, exit %d, invocation %q; want gg %v = output %q, exit %d, invocation %q", aliasArgs, aliasOut, aliasCode, aliasLog, canonicalArgs, canonicalOut, canonicalCode, canonicalLog)
		}
	})

	t.Run("usage error", func(t *testing.T) {
		cases := []struct {
			name          string
			canonicalArgs []string
			aliasArgs     []string
		}{
			{
				name:          "missing action",
				canonicalArgs: []string{"pr"},
				aliasArgs:     []string{"mr"},
			},
			{
				name:          "unsupported action",
				canonicalArgs: []string{"pr", "unsupported"},
				aliasArgs:     []string{"mr", "unsupported"},
			},
			{
				name:          "unknown flag",
				canonicalArgs: []string{"pr", "list", "--unknown"},
				aliasArgs:     []string{"mr", "list", "--unknown"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				canonicalOut, canonicalErr, canonicalCode := runGGStreams(t, bin, repo, tc.canonicalArgs...)
				aliasOut, aliasErr, aliasCode := runGGStreams(t, bin, repo, tc.aliasArgs...)
				if aliasOut != canonicalOut || aliasErr != canonicalErr || aliasCode != canonicalCode {
					t.Errorf("gg %v = stdout %q, stderr %q, exit %d; want gg %v = stdout %q, stderr %q, exit %d", tc.aliasArgs, aliasOut, aliasErr, aliasCode, tc.canonicalArgs, canonicalOut, canonicalErr, canonicalCode)
				}
			})
		}
	})

	t.Run("nested help", func(t *testing.T) {
		actions := []string{"create", "status", "ready", "merge"}
		for _, action := range actions {
			t.Run(action, func(t *testing.T) {
				canonicalOut, canonicalErr, canonicalCode := runGGStreams(t, bin, repo, "pr", action, "--help")
				aliasOut, aliasErr, aliasCode := runGGStreams(t, bin, repo, "mr", action, "--help")
				if aliasOut != canonicalOut || aliasErr != canonicalErr || aliasCode != canonicalCode {
					t.Errorf("gg mr %s --help = stdout %q, stderr %q, exit %d; want gg pr %s --help = stdout %q, stderr %q, exit %d", action, aliasOut, aliasErr, aliasCode, action, canonicalOut, canonicalErr, canonicalCode)
				}
			})
		}
	})

	t.Run("top-level help omits duplicate alias", func(t *testing.T) {
		stdout, stderr, code := runGGStreams(t, bin, repo, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("gg --help = stdout %q, stderr %q, exit %d", stdout, stderr, code)
		}
		if strings.Contains(stdout, "\n  mr ") {
			t.Errorf("gg --help should only list canonical commands:\n%s", stdout)
		}
	})
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

func TestE2EPullPushPassThroughToGit(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "git", logFile)

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"pull", "--rebase", "origin", "main"}, want: "git pull --rebase origin main"},
		{args: []string{"push", "--force-with-lease", "origin", "main"}, want: "git push --force-with-lease origin main"},
	} {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, t.TempDir(), test.args...)
		if code != 0 {
			t.Fatalf("gg %v exits %d: %s", test.args, code, out)
		}
		if got := readLog(t, logFile); !strings.Contains(got, test.want) {
			t.Errorf("gg %v git argv = %q, want %q", test.args, got, test.want)
		}
	}
}

func TestE2EGitPassthroughRoutesAllForms(t *testing.T) {
	bin := buildGG(t)
	gitBin := buildGitPassthroughProbe(t)
	logFile := filepath.Join(t.TempDir(), "git-args.jsonl")
	t.Setenv("GG_GIT_LOG", logFile)
	rawArgs := []string{"--", "-starts-with-hyphen", "two words", "", "--repo", "https://example.invalid/o/r"}
	workDir := t.TempDir()
	var wantCalls [][]string

	for _, action := range gitPassthroughActionNames {
		for _, form := range [][]string{nil, {"repo"}} {
			for _, suffix := range [][]string{rawArgs, {"--help"}} {
				args := append(append([]string{}, form...), action)
				args = append(args, suffix...)
				if _, code := runGG(t, bin, filepath.Dir(gitBin), workDir, args...); code != 23 {
					t.Errorf("gg %v exit = %d, want 23", args, code)
				}
				wantCalls = append(wantCalls, append([]string{action}, suffix...))
			}
		}
	}

	calls := readGitPassthroughCalls(t, logFile)
	if len(calls) != len(wantCalls) {
		t.Fatalf("git call count = %d, want %d", len(calls), len(wantCalls))
	}
	for i := range wantCalls {
		if !slices.Equal(calls[i], wantCalls[i]) {
			t.Errorf("git call %d = %q, want %q", i, calls[i], wantCalls[i])
		}
	}
}

func TestE2EPassthroughContextFlagPositions(t *testing.T) {
	bin := buildGG(t)
	gitBin := buildGitPassthroughProbe(t)
	logFile := filepath.Join(t.TempDir(), "git-args.jsonl")
	t.Setenv("GG_GIT_LOG", logFile)
	workDir := t.TempDir()
	contextFlags := []struct {
		name string
		args []string
	}{
		{name: "--repo", args: []string{"--repo", "https://example.invalid/o/r"}},
		{name: "--remote", args: []string{"--remote", "git-owned-remote"}},
		{name: "--explain", args: []string{"--explain"}},
	}

	for _, action := range []string{"status", "mergetool", "commit", "pull", "push"} {
		for _, form := range [][]string{nil, {"repo"}} {
			for _, contextFlag := range contextFlags {
				for _, suffix := range [][]string{nil, {"--help"}} {
					testPath := append(append([]string{}, form...), action, contextFlag.name)
					testPath = append(testPath, suffix...)
					t.Run(strings.Join(testPath, " "), func(t *testing.T) {
						if err := os.WriteFile(logFile, nil, 0o600); err != nil {
							t.Fatal(err)
						}
						leading := append(append([]string{}, contextFlag.args...), form...)
						leading = append(leading, action)
						leading = append(leading, suffix...)
						if out, code := runGG(t, bin, filepath.Dir(gitBin), workDir, leading...); code != 2 {
							t.Errorf("gg %v exit = %d, want 2: %s", leading, code, out)
						}
						if calls := readGitPassthroughCalls(t, logFile); len(calls) != 0 {
							t.Errorf("gg %v should not run git, calls = %q", leading, calls)
						}

						if err := os.WriteFile(logFile, nil, 0o600); err != nil {
							t.Fatal(err)
						}
						trailing := append(append([]string{}, form...), action)
						trailing = append(trailing, contextFlag.args...)
						trailing = append(trailing, suffix...)
						if out, code := runGG(t, bin, filepath.Dir(gitBin), workDir, trailing...); code != 23 {
							t.Errorf("gg %v exit = %d, want 23: %s", trailing, code, out)
						}
						want := []string{action}
						if action == "commit" {
							want = append(want, "--no-gpg-sign")
						}
						want = append(want, contextFlag.args...)
						want = append(want, suffix...)
						if calls := readGitPassthroughCalls(t, logFile); len(calls) != 1 || !slices.Equal(calls[0], want) {
							t.Errorf("gg %v git calls = %q, want %q", trailing, calls, [][]string{want})
						}
					})
				}
			}
		}
	}
}

func TestE2EGitPassthroughRelaysChildStreams(t *testing.T) {
	bin := buildGG(t)
	gitBin := buildGitPassthroughProbe(t)
	logFile := filepath.Join(t.TempDir(), "git-args.jsonl")
	t.Setenv("GG_GIT_LOG", logFile)

	cmd := ggCommand(t, bin, filepath.Dir(gitBin), t.TempDir(), "mergetool", "--", "-interactive")
	cmd.Stdin = strings.NewReader("typed input\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if code := processExitCode(t, err, "stdout: "+stdout.String()+"\nstderr: "+stderr.String()); code != 23 {
		t.Errorf("exit = %d, want 23", code)
	}
	if got, want := stdout.String(), "stdout:typed input\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "stderr:typed input\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
	if calls := readGitPassthroughCalls(t, logFile); len(calls) != 1 || !slices.Equal(calls[0], []string{"mergetool", "--", "-interactive"}) {
		t.Errorf("git calls = %q, want [[mergetool -- -interactive]]", calls)
	}
}

func TestE2EGitPassthroughDoesNotShadowGGResourceHelp(t *testing.T) {
	bin := buildGG(t)
	gitBin := buildGitPassthroughProbe(t)
	logFile := filepath.Join(t.TempDir(), "git-args.jsonl")
	t.Setenv("GG_GIT_LOG", logFile)

	for _, args := range [][]string{{"repo", "--help"}, {"issue", "--help"}, {"pr", "--help"}, {"config", "--help"}} {
		cmd := ggCommand(t, bin, filepath.Dir(gitBin), t.TempDir(), args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if code := processExitCode(t, cmd.Run(), "stdout: "+stdout.String()+"\nstderr: "+stderr.String()); code != 0 {
			t.Errorf("gg %v exit = %d, want 0", args, code)
		}
		if !strings.Contains(stdout.String(), "Usage:") || stderr.Len() != 0 {
			t.Errorf("gg %v help = stdout %q, stderr %q", args, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Errorf("resource help should not run git, stat error = %v", err)
	}

	cmd := ggCommand(t, bin, filepath.Dir(gitBin), t.TempDir(), "diff", "--help")
	if code := processExitCode(t, cmd.Run(), "gg diff --help"); code != 23 {
		t.Errorf("gg diff --help exit = %d, want 23", code)
	}
	if calls := readGitPassthroughCalls(t, logFile); len(calls) != 1 || !slices.Equal(calls[0], []string{"diff", "--help"}) {
		t.Errorf("git calls = %q, want [[diff --help]]", calls)
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

	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"-m", "msg"}, "git commit --no-gpg-sign -m msg"},
		{[]string{"-v"}, "git commit --no-gpg-sign -v"},
	} {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), append([]string{"commit"}, tt.args...)...)
		if code != 0 {
			t.Fatalf("gg commit %v exit %d: %s", tt.args, code, out)
		}
		if got := readLog(t, logFile); !strings.Contains(got, tt.want) {
			t.Errorf("gg commit %v git argv = %q, want %q", tt.args, got, tt.want)
		}
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

func TestE2EExplainIssueListBothFlagOrders(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := [][]string{
		{"--explain", "issue", "list"},
		{"issue", "list", "--explain"},
	}

	for _, args := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d, output: %s", args, code, out)
		}
		wants := []string{
			"저장소 문맥: https://github.com/o/r",
			"Provider: gh",
			"CLI: gh",
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Errorf("gg %v output missing %q:\n%s", args, want, out)
			}
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got: %q", args, got)
		}
	}
}

func TestE2EExplainWithRepoFlag(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)

	cases := [][]string{
		{"--explain", "--repo", "https://gitlab.com/o/r", "issue", "list"},
		{"issue", "list", "--repo", "https://gitlab.com/o/r", "--explain"},
		{"--explain", "repo", "view", "--repo", "git@gitlab.com:o/r.git"},
	}

	for _, args := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, t.TempDir(), args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d, output: %s", args, code, out)
		}
		wants := []string{
			"저장소 문맥: https://gitlab.com/o/r",
			"Provider: glab",
			"CLI: glab",
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Errorf("gg %v output missing %q:\n%s", args, want, out)
			}
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got: %q", args, got)
		}
	}
}

func TestE2EExplainWithRemoteFlag(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	cases := [][]string{
		{"--explain", "--remote", "upstream", "pr", "list"},
		{"pr", "list", "--remote", "upstream", "--explain"},
	}

	for _, args := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d, output: %s", args, code, out)
		}
		wants := []string{
			"저장소 문맥: https://github.com/o/upstream",
			"Provider: gh",
			"CLI: gh",
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Errorf("gg %v output missing %q:\n%s", args, want, out)
			}
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got: %q", args, got)
		}
	}
}

func TestE2EExplainAutomaticRemoteSelection(t *testing.T) {
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
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if !strings.Contains(out, "저장소 문맥: https://github.com/o/upstream") || !strings.Contains(out, "Provider: gh") || !strings.Contains(out, "CLI: gh") {
			t.Errorf("output unexpected: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got: %q", got)
		}
	})

	t.Run("origin", func(t *testing.T) {
		repo := tempRepo(t, "https://github.com/o/origin.git")
		gitIn(t, repo, "remote", "add", "other", "https://github.com/o/other.git")

		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if !strings.Contains(out, "저장소 문맥: https://github.com/o/origin") || !strings.Contains(out, "Provider: gh") || !strings.Contains(out, "CLI: gh") {
			t.Errorf("output unexpected: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got: %q", got)
		}
	})

	t.Run("single remote", func(t *testing.T) {
		repo := t.TempDir()
		gitIn(t, repo, "init", "-q")
		gitIn(t, repo, "remote", "add", "solo", "https://github.com/o/solo.git")

		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "issue", "list")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if !strings.Contains(out, "저장소 문맥: https://github.com/o/solo") || !strings.Contains(out, "Provider: gh") || !strings.Contains(out, "CLI: gh") {
			t.Errorf("output unexpected: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got: %q", got)
		}
	})
}

func TestE2EExplainErrorsPreserved(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	t.Run("missing remote", func(t *testing.T) {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "--remote", "missing", "issue", "list")
		if code != 1 {
			t.Fatalf("exit = %d, want 1: %s", code, out)
		}
		if !strings.Contains(out, `remote "missing" not found`) {
			t.Errorf("expected missing remote message in output: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got %q", got)
		}
	})

	t.Run("flag conflict", func(t *testing.T) {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "--repo", "https://github.com/o/r", "--remote", "upstream", "issue", "list")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "cannot be used together") {
			t.Errorf("expected conflict message in output: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got %q", got)
		}
	})

	t.Run("missing command", func(t *testing.T) {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "missing command") {
			t.Errorf("expected missing command message in output: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got %q", got)
		}
	})

	t.Run("unsupported config command", func(t *testing.T) {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, "--explain", "config", "list")
		if code != 2 {
			t.Fatalf("exit = %d, want 2: %s", code, out)
		}
		if !strings.Contains(out, "--explain is not supported for config list") {
			t.Errorf("expected unsupported message in output: %s", out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("child should not run, got %q", got)
		}
	})
}

func TestE2EExplainDoesNotLeakUserInputs(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	if err := os.WriteFile(logFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	out, code := runGG(t, bin, fakeDir, repo, "--explain", "issue", "create", "--title", "SECRET_TITLE", "--body", "SECRET_TOKEN_12345")
	if code != 0 {
		t.Fatalf("exit = %d: %s", code, out)
	}
	if strings.Contains(out, "SECRET_TITLE") || strings.Contains(out, "SECRET_TOKEN_12345") {
		t.Errorf("explain output leaked user inputs:\n%s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("child should not run, got: %q", got)
	}
}

func TestE2EGitHubIssueCommentCloseReopen(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepoWithUpstream(t)

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"issue", "comment", "18", "--body", "fixed"}, "gh issue comment 18 --body fixed -R github.com/o/origin"},
		{[]string{"issue", "close", "18"}, "gh issue close 18 -R github.com/o/origin"},
		{[]string{"issue", "reopen", "18"}, "gh issue reopen 18 -R github.com/o/origin"},
		{[]string{"issue", "comment", "18", "--body", "upstream-note", "--remote", "upstream"}, "gh issue comment 18 --body upstream-note -R github.com/o/upstream"},
		{[]string{"issue", "close", "18", "--remote", "upstream"}, "gh issue close 18 -R github.com/o/upstream"},
		{[]string{"issue", "reopen", "18", "--remote", "upstream"}, "gh issue reopen 18 -R github.com/o/upstream"},
		{[]string{"--repo", "https://github.com/custom/repo", "issue", "comment", "18", "--body", "repo-flag"}, "gh issue comment 18 --body repo-flag -R github.com/custom/repo"},
		{[]string{"--repo", "https://github.com/custom/repo", "issue", "close", "18"}, "gh issue close 18 -R github.com/custom/repo"},
		{[]string{"--repo", "https://github.com/custom/repo", "issue", "reopen", "18"}, "gh issue reopen 18 -R github.com/custom/repo"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		got := readLog(t, logFile)
		if got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EGitLabIssueCommentCloseReopen(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"issue", "comment", "18", "--body", "fixed"}, "glab issue note 18 --message fixed --repo https://gitlab.com/o/r"},
		{[]string{"issue", "close", "18"}, "glab issue close 18 --repo https://gitlab.com/o/r"},
		{[]string{"issue", "reopen", "18"}, "glab issue reopen 18 --repo https://gitlab.com/o/r"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		got := readLog(t, logFile)
		if got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EGitLabLabelListCreate(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"label", "list"}, "glab label list --repo https://gitlab.com/o/r"},
		{[]string{"label", "list", "--limit", "3"}, "glab label list --repo https://gitlab.com/o/r --per-page 3"},
		{[]string{"label", "create", "--name", "bug"}, "glab label create --repo https://gitlab.com/o/r --name bug"},
		{[]string{"label", "create", "--name", "bug", "--color", "#FF0000", "--description", "broken"},
			"glab label create --repo https://gitlab.com/o/r --name bug --color #FF0000 --description broken"},
		{[]string{"--repo", "https://gitlab.com/custom/repo", "label", "create", "--name", "bug"},
			"glab label create --repo https://gitlab.com/custom/repo --name bug"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		got := readLog(t, logFile)
		if got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestE2EGitHubLabelUnsupported(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"label", "list"},
		{"label", "create", "--name", "bug"},
	} {
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "is not supported for gh") {
			t.Errorf("gg %v output에 미지원 오류 없음: %s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got %q", args, got)
		}
	}
}

func TestE2EGiteaLabelUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "label", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2: %s", code, out)
	}
	if !strings.Contains(out, "label list is not supported for tea") {
		t.Errorf("output에 미지원 오류 없음: %s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea should not run, got %q", got)
	}
}

func TestE2EGiteaIssueCommentCloseReopen(t *testing.T) {
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
		args []string
		want string
	}{
		{[]string{"issue", "comment", "18", "--body", "fixed"}, "tea comment 18 fixed --login pub --repo o/r"},
		{[]string{"issue", "close", "18"}, "tea issues close 18 --login pub --repo o/r"},
		{[]string{"issue", "reopen", "18"}, "tea issues reopen 18 --login pub --repo o/r"},
	}

	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		got := readLog(t, logFile)
		if got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}
