package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ciViewTestRepo는 커밋이 하나 있는 GitHub 임시 저장소를 만든다. 현재 branch
// 조회가 rev-parse HEAD에 의존하므로 커밋이 필요하다.
func ciViewTestRepo(t *testing.T) string {
	t.Helper()
	repo := tempRepo(t, "https://github.com/o/r.git")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@e.com")
	gitIn(t, repo, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-qm", "first")
	return repo
}

func TestE2ECIViewLatestOnCurrentBranch(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHWithID(t, fakeDir, logFile, "1234567890")
	repo := ciViewTestRepo(t)
	branch := strings.TrimSpace(gitIn(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))

	out, code := runGG(t, bin, fakeDir, repo, "ci", "view")
	if code != 0 {
		t.Fatalf("gg ci view: exit %d: %s", code, out)
	}
	lines := readLogLines(t, logFile)
	if len(lines) != 2 {
		t.Fatalf("gh 호출 수 = %d, want 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	wantList := wantCall("gh", "run", "list", "-R", "github.com/o/r",
		"--branch", branch, "--limit", "1", "--json", "databaseId", "--jq", ".[0].databaseId")
	if lines[0] != wantList {
		t.Errorf("run list argv = %s, want %s", lines[0], wantList)
	}
	wantView := wantCall("gh", "run", "view", "1234567890", "-R", "github.com/o/r")
	if lines[1] != wantView {
		t.Errorf("run view argv = %s, want %s", lines[1], wantView)
	}
}

func TestE2ECIViewWithIDSkipsLookup(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeGHWithID(t, fakeDir, logFile, "1234567890")
	repo := ciViewTestRepo(t)

	out, code := runGG(t, bin, fakeDir, repo, "ci", "view", "42")
	if code != 0 {
		t.Fatalf("gg ci view 42: exit %d: %s", code, out)
	}
	lines := readLogLines(t, logFile)
	if len(lines) != 1 {
		t.Fatalf("gh 호출 수 = %d, want 1:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if want := wantCall("gh", "run", "view", "42", "-R", "github.com/o/r"); lines[0] != want {
		t.Errorf("run view argv = %s, want %s", lines[0], want)
	}
}

func TestE2ECIViewWithoutRunsFails(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeCLI(t, fakeDir, "gh", fakeCLIConfig{LogFile: logFile, ExitCode: 1})
	repo := ciViewTestRepo(t)

	stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, "ci", "view")
	if code != 1 {
		t.Errorf("exit = %d, want 1 (stdout: %s)", code, stdout)
	}
	if !strings.Contains(stderr, "cannot resolve the latest run on branch") {
		t.Errorf("stderr = %q", stderr)
	}
	lines := readLogLines(t, logFile)
	if len(lines) != 1 {
		t.Errorf("gh 호출 = %v, run list 한 번이어야 한다", lines)
	}
}
