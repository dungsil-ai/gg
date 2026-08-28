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
	if !strings.Contains(got, "gh issue list -R github.com/o/r --limit 3") {
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
