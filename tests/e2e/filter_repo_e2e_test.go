package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EFilterRepoReplaceTextFromFile(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{
		"a.txt":    "secret password=hunter2\npreserve me\n",
		"note.txt": "public secret password=abc123\n",
	})
	f := filepath.Join(t.TempDir(), "replacements.txt")
	content := "# comment\n\nsecret==>***\npassword=\\S+==>password=?\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--replace-text", f, "--force"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		object string
		want   string
	}{
		{"HEAD^:a.txt", "*** password=?\npreserve me"},
		{"HEAD^:note.txt", "public *** password=?"},
		{"HEAD:a.txt", "second"},
		{"HEAD:note.txt", "public *** password=?"},
	} {
		if got := gitIn(t, dir, "show", c.object); got != c.want {
			t.Errorf("%s = %q, want %q", c.object, got, c.want)
		}
	}
	for name, want := range map[string]string{
		"a.txt":    "second\n",
		"note.txt": "public *** password=?\n",
	} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("working tree %s = %q, want %q", name, got, want)
		}
	}
}

// 메모장·PowerShell 등이 넣는 UTF-8 BOM이 파일 첫 규칙을 조용히 무력화하지
// 않아야 한다.
func TestE2EFilterRepoReplaceTextFromFileWithBOM(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "secret password=hunter2\n"})
	f := filepath.Join(t.TempDir(), "replacements.txt")
	content := "\xef\xbb\xbfsecret==>***\npassword=\\S+==>password=?\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--replace-text", f, "--force"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "show", "HEAD:a.txt"); got != "second" {
		t.Errorf("BOM 뒤 첫 규칙 적용 기대, got %q", got)
	}
	if got := gitIn(t, dir, "show", "HEAD^:a.txt"); got != "*** password=?" {
		t.Errorf("BOM 뒤 두 번째 규칙 적용 기대, got %q", got)
	}
}

// filterRepoTestRepo는 실제 git 저장소를 만들고 두 커밋을 쌓는다.
func filterRepoTestRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "config", "user.name", "Old Name")
	gitIn(t, dir, "config", "user.email", "old@example.com")
	// 테스트 환경의 전역 커밋 서명(gpg/ssh)이 끼지 않도록 끈다.
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	// 작업 파일의 바이트 비교가 전역 줄바꿈 변환 설정에 영향을 받지 않게 한다.
	gitIn(t, dir, "config", "core.autocrlf", "false")
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "first")
	second := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(second, []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "second")
	return dir
}

func runFilterRepoIn(t *testing.T, dir string, args []string) error {
	t.Helper()
	_, err := runFilterRepoInOutput(t, dir, args)
	return err
}

func runFilterRepoInOutput(t *testing.T, dir string, args []string) (string, error) {
	t.Helper()
	stdout, stderr, code := runGGStreams(t, buildGG(t), dir, args...)
	if code != 0 {
		return stdout, fmt.Errorf("gg %v: exit %d: %s", args, code, stderr)
	}
	return stdout, nil
}

func TestE2EFilterRepoRemovesPath(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "log", "--oneline", "--all"); len(strings.Split(strings.TrimSpace(got), "\n")) != 2 {
		t.Errorf("커밋 2개 유지 기대, got:\n%s", got)
	}
	if got := gitIn(t, dir, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(got, "secret.txt") {
		t.Errorf("secret.txt 제거 기대, got %q", got)
	}
	if got := gitIn(t, dir, "log", "--all", "--oneline", "--", "secret.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("히스토리에서 secret.txt 제거 기대, got %q", got)
	}
	if out := gitIn(t, dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("작업 트리 clean 기대, got %q", out)
	}
}

func TestE2EFilterRepoKeepRenameReplaceMailmap(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "docs/note.txt": "pw=hunter2\n"})
	mailmap := filepath.Join(t.TempDir(), "mailmap")
	if err := os.WriteFile(mailmap, []byte("New Name <new@example.com> <old@example.com>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "docs", "--path-rename", "docs:manual", "--replace-text", `hunter2==>***`, "--mailmap", mailmap, "--force"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "ls-tree", "-r", "--name-only", "HEAD"); got != "manual/note.txt" {
		t.Errorf("HEAD files = %q, want manual/note.txt", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "manual", "note.txt"))
	if err != nil || !strings.Contains(string(data), "***") || strings.Contains(string(data), "hunter2") {
		t.Errorf("replace 기대, got %q, err %v", data, err)
	}
	if got := gitIn(t, dir, "log", "--format=%an <%ae>", "--all"); strings.Contains(got, "old@example.com") {
		t.Errorf("mailmap 기대, got %q", got)
	}
}

func TestE2EFilterRepoDryRunChangesNothing(t *testing.T) {
	for _, c := range []struct {
		name        string
		path        string
		rename      string
		replacement string
		want        string
	}{
		{
			name: "변경 미리보기", path: "secret.txt", rename: "docs:manual", replacement: "hunter2==>***",
			want: "dry-run: 2 commits, 4 blobs (1 would change), 4 identities (0 would change), 2 paths dropped, 2 renamed\n",
		},
		{
			name: "일치하는 대상 없음", path: "missing.txt", rename: "missing:other", replacement: "absent==>***",
			want: "dry-run: 2 commits, 4 blobs (0 would change), 4 identities (0 would change), 0 paths dropped, 0 renamed\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := filterRepoTestRepo(t, map[string]string{
				"a.txt": "keep\n", "secret.txt": "topsecret\n", "docs/note.txt": "password=hunter2\n",
			})
			gitIn(t, dir, "branch", "saved", "HEAD^")
			gitIn(t, dir, "tag", "before-filter", "HEAD^")
			beforeHead := gitIn(t, dir, "rev-parse", "HEAD")
			beforeRefs := gitIn(t, dir, "show-ref")
			beforeTree := gitIn(t, dir, "ls-tree", "-r", "HEAD")
			beforeFiles := map[string][]byte{}
			for _, name := range []string{"a.txt", "secret.txt", "docs/note.txt"} {
				data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				beforeFiles[name] = data
			}

			out, err := runFilterRepoInOutput(t, dir, []string{"repo", "filter-repo", "--path", c.path, "--invert-paths", "--dry-run", "--path-rename", c.rename, "--replace-text", c.replacement})
			if err != nil {
				t.Fatal(err)
			}
			if out != c.want {
				t.Errorf("dry-run preview = %q, want %q", out, c.want)
			}
			if got := gitIn(t, dir, "rev-parse", "HEAD"); got != beforeHead {
				t.Error("dry-run: HEAD 변경 없음 기대")
			}
			if got := gitIn(t, dir, "show-ref"); got != beforeRefs {
				t.Errorf("dry-run: refs 변경 없음 기대, got %q, want %q", got, beforeRefs)
			}
			if got := gitIn(t, dir, "ls-tree", "-r", "HEAD"); got != beforeTree {
				t.Errorf("dry-run: tree 변경 없음 기대, got %q, want %q", got, beforeTree)
			}
			for name, want := range beforeFiles {
				got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("dry-run: %s 변경 없음 기대, got %q, want %q", name, got, want)
				}
			}
			if got := gitIn(t, dir, "status", "--porcelain"); got != "" {
				t.Errorf("dry-run: 작업 트리 clean 기대, got %q", got)
			}
		})
	}
}

func TestE2EFilterRepoSafety(t *testing.T) {
	t.Run("force 없이 거부", func(t *testing.T) {
		dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n"})
		before := gitIn(t, dir, "rev-parse", "HEAD")
		err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "a.txt", "--invert-paths"})
		if err == nil || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("--force 안내 기대, got %v", err)
		}
		if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
			t.Error("거부 시 HEAD 변경 없음 기대")
		}
	})
	t.Run("dirty 작업 트리 거부", func(t *testing.T) {
		dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n"})
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dirty\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "a.txt", "--invert-paths", "--force"})
		if err == nil || !strings.Contains(err.Error(), "dirty") {
			t.Fatalf("dirty 안내 기대, got %v", err)
		}
	})
	t.Run("저장소 밖 거부", func(t *testing.T) {
		err := runFilterRepoIn(t, t.TempDir(), []string{"repo", "filter-repo", "--path", "a.txt", "--invert-paths", "--force"})
		if err == nil || !strings.Contains(err.Error(), "not a git repository") {
			t.Fatalf("저장소 안내 기대, got %v", err)
		}
	})
}

func TestE2EFilterRepoPreservesMergesAndTags(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "config", "user.name", "T")
	gitIn(t, dir, "config", "user.email", "t@e.com")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "base")
	gitIn(t, dir, "checkout", "-qb", "side")
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "side")
	gitIn(t, dir, "checkout", "-q", "-")
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "main")
	gitIn(t, dir, "merge", "-q", "--no-ff", "side", "-m", "merge side")
	gitIn(t, dir, "tag", "-a", "v1", "-m", "release one")

	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "b.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	// 머지 토폴로지: 머지 커밋의 부모가 2개여야 한다.
	parents := gitIn(t, dir, "log", "-1", "--format=%P", "HEAD")
	if len(strings.Fields(parents)) != 2 {
		t.Errorf("merge parents = %q, want 2", parents)
	}
	if tags := gitIn(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "v1" {
		t.Errorf("tags = %q, want v1", tags)
	}
	if got := gitIn(t, dir, "log", "--all", "--oneline", "--", "b.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("b.txt 제거 기대, got %q", got)
	}
}

func TestE2EFilterRepoCreatesBackup(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	gitDir := strings.TrimSpace(gitIn(t, dir, "rev-parse", "--absolute-git-dir"))
	backup := filepath.Join(gitDir, "filter-repo-backup")
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup 없음: %v", err)
	}
	// 백업에는 재작성 전 히스토리(삭제된 secret.txt 포함)가 그대로 남는다.
	if got := gitIn(t, backup, "ls-tree", "-r", "--name-only", "HEAD"); !strings.Contains(got, "secret.txt") {
		t.Errorf("backup HEAD에 secret.txt 유지 기대, got %q", got)
	}
	if got := gitIn(t, backup, "log", "--oneline"); len(strings.Split(strings.TrimSpace(got), "\n")) != 2 {
		t.Errorf("backup에 원본 커밋 2개 유지 기대, got:\n%s", got)
	}
}

func TestE2EFilterRepoBackupNotOverwritten(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	req := []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}
	if err := runFilterRepoIn(t, dir, req); err != nil {
		t.Fatal(err)
	}
	before := gitIn(t, dir, "rev-parse", "HEAD")
	err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "a.txt", "--invert-paths", "--force"})
	if err == nil || !strings.Contains(err.Error(), "backup already exists") {
		t.Fatalf("기존 backup 보호 기대, got %v", err)
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
		t.Errorf("거부 시 HEAD 변경 없음 기대: %s -> %s", before, got)
	}
}

func TestE2EFilterRepoBareRepository(t *testing.T) {
	src := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	bare := filepath.Join(t.TempDir(), "srv.git")
	gitIn(t, filepath.Dir(src), "clone", "--bare", "-q", src, bare)

	if err := runFilterRepoIn(t, bare, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, bare, "log", "--all", "--oneline", "--", "secret.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("bare 저장소 히스토리에서 secret.txt 제거 기대, got %q", got)
	}
	if got := gitIn(t, bare, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(got, "secret.txt") {
		t.Errorf("bare 저장소 HEAD에 secret.txt 제거 기대, got %q", got)
	}
}

func TestE2EFilterRepoDetachedHeadRefused(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	gitIn(t, dir, "checkout", "-q", "--detach")
	before := gitIn(t, dir, "rev-parse", "HEAD")

	err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"})
	if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
		t.Fatalf("detached HEAD 거부 기대, got %v", err)
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
		t.Errorf("거부 시 HEAD 변경 없음 기대: %s -> %s", before, got)
	}
	if got := gitIn(t, dir, "log", "--all", "--oneline", "--", "secret.txt"); strings.TrimSpace(got) == "" {
		t.Error("거부 시 히스토리 변경 없음 기대")
	}
}

func TestE2EFilterRepoRefusesStash(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "stash", "-q")
	before := gitIn(t, dir, "rev-parse", "HEAD")

	err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"})
	if err == nil || !strings.Contains(err.Error(), "stash is not empty") {
		t.Fatalf("stash 거부 기대, got %v", err)
	}
	if got := gitIn(t, dir, "stash", "list"); strings.TrimSpace(got) == "" {
		t.Error("stash 항목 유지 기대")
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
		t.Errorf("거부 시 HEAD 변경 없음 기대: %s -> %s", before, got)
	}
}

func TestE2EFilterRepoRefusesLinkedWorktrees(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n"})
	// 연결 worktree는 저장소 바깥에 둔다. 안에 두면 untracked 파일로 작업
	// 트리가 더러워져 worktree 검사보다 dirty 검사가 먼저 걸린다.
	linked := filepath.Join(t.TempDir(), "linked")
	gitIn(t, dir, "worktree", "add", linked, "-b", "wtb")
	before := gitIn(t, dir, "rev-parse", "HEAD")

	err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "a.txt", "--invert-paths", "--force"})
	if err == nil || !strings.Contains(err.Error(), "multiple worktrees") {
		t.Fatalf("연결 worktree 거부 기대, got %v", err)
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
		t.Errorf("거부 시 HEAD 변경 없음 기대: %s -> %s", before, got)
	}
}

func TestE2EFilterRepoRewritesBranchesAndTagsOnly(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	gitIn(t, dir, "tag", "-a", "v1", "-m", "tag one")
	remoteSha := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", remoteSha)

	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "log", "--branches", "--oneline", "--", "secret.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("branch 히스토리에서 secret.txt 제거 기대, got %q", got)
	}
	// 원격 추적 ref도 재작성한다. 남겨 두면 예전 커밋이 refs/remotes에 붙들려
	// gc 뒤에도 살아 있어 비밀 제거가 불완전해진다.
	if got := strings.TrimSpace(gitIn(t, dir, "rev-parse", "refs/remotes/origin/main")); got == remoteSha {
		t.Errorf("원격 추적 ref 재작성 기대: %s", got)
	}
}

func TestE2EFilterRepoPrunesObjectsBehindRemoteRefs(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	oldCommit := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	oldSecretBlob := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD:secret.txt"))
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", oldCommit)
	gitIn(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}
	// refs/remotes가 재작성 전 커밋을 붙들면 gc가 prune하지 못해 비밀이
	// `git show refs/remotes/origin/main:secret.txt`로 그대로 복구된다.
	for name, object := range map[string]string{"예전 커밋": oldCommit, "예전 블롭": oldSecretBlob} {
		cmd := exec.Command("git", "-C", dir, "cat-file", "-e", object)
		if err := cmd.Run(); err == nil {
			t.Errorf("%s 객체(%s)가 gc 뒤에도 남아 있다", name, object)
		}
	}
	if out, err := exec.Command("git", "-C", dir, "show", "refs/remotes/origin/main:secret.txt").CombinedOutput(); err == nil {
		t.Errorf("원격 추적 ref에서 비밀 제거 기대, got %q", out)
	}
	if got := gitIn(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD"); got != "refs/remotes/origin/main" {
		t.Errorf("origin/HEAD 심볼릭 유지 기대, got %q", got)
	}
}

// nopWriteCloser는 오류 없이 모든 쓰기를 받아들이는 io.WriteCloser다.
type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }

func (nopWriteCloser) Close() error { return nil }

// 재작성 후에는 예전 객체가 완전히 prune되어야 한다. logs/HEAD나 오래된
// index가 예전 커밋을 붙들고 있으면 비밀 제거가 실패한 것이다.
func TestE2EFilterRepoPrunesOldObjects(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	oldCommit := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	oldSecretBlob := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD:secret.txt"))

	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"}); err != nil {
		t.Fatal(err)
	}

	for name, object := range map[string]string{"예전 커밋": oldCommit, "예전 블롭": oldSecretBlob} {
		cmd := exec.Command("git", "-C", dir, "cat-file", "-e", object)
		if err := cmd.Run(); err == nil {
			t.Errorf("%s 객체(%s)가 gc 뒤에도 남아 있다", name, object)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".git", "logs", "HEAD")); err == nil {
		if strings.Contains(string(data), oldCommit) {
			t.Errorf("logs/HEAD에 예전 SHA가 남아 있다:\n%s", data)
		}
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD:a.txt"); got == "" {
		t.Error("새 HEAD의 a.txt를 읽지 못했다")
	}
}

func TestE2EFilterRepoRefusesNestedTags(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "config", "user.name", "T")
	gitIn(t, dir, "config", "user.email", "t@e.com")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "c1")
	gitIn(t, dir, "tag", "-a", "v1", "-m", "v1 message")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("world\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "c2")
	gitIn(t, dir, "tag", "-a", "v2", "-m", "v2 message", "v1")
	before := gitIn(t, dir, "rev-parse", "HEAD")

	for _, dryRun := range []bool{false, true} {
		mode := "--force"
		if dryRun {
			mode = "--dry-run"
		}
		err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--replace-text", "hello==>hallo", mode})
		if err == nil || !strings.Contains(err.Error(), "nested annotated tag v2") {
			t.Fatalf("중첩 태그 거부 기대 (dry-run=%v), got %v", dryRun, err)
		}
	}
	if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
		t.Errorf("거부 시 HEAD 변경 없음 기대: %s -> %s", before, got)
	}
	if got := gitIn(t, dir, "cat-file", "-t", "refs/tags/v2"); got != "tag" {
		t.Errorf("v2는 여전히 tag 객체여야 한다, got %s", got)
	}
	if got := gitIn(t, dir, "cat-file", "-t", "refs/tags/v1"); got != "tag" {
		t.Errorf("v1은 여전히 tag 객체여야 한다, got %s", got)
	}
}
