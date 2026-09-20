package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// releasePrepareOrigin은 로컬 bare origin과 그 clone(커밋 1건 push 완료)을
// 만든다. 네트워크 없이 push·ls-remote가 동작하는 실제 git 원격이다.
func releasePrepareOrigin(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	gitIn(t, base, "init", "-q", "--bare", "-b", "main", "origin.git")
	work := filepath.Join(base, "work")
	gitIn(t, base, "clone", "-q", origin, "work")
	gitIn(t, work, "config", "user.name", "T")
	gitIn(t, work, "config", "user.email", "t@e.com")
	gitIn(t, work, "config", "commit.gpgsign", "false")
	gitIn(t, work, "config", "core.autocrlf", "false")
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("v\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, work, "add", ".")
	gitIn(t, work, "commit", "-qm", "c1")
	gitIn(t, work, "push", "-q", "origin", "HEAD:refs/heads/main")
	// 빈 저장소를 clone하면 origin/HEAD가 만들어지지 않으므로 명시적으로 채운다.
	gitIn(t, work, "remote", "set-head", "origin", "main")
	return work
}

func TestE2EReleasePrepareHappyPath(t *testing.T) {
	bin := buildGG(t)
	work := releasePrepareOrigin(t)

	out, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
	if code != 0 {
		t.Fatalf("exit %d: stdout=%s stderr=%s", code, out, stderr)
	}
	for _, want := range []string{
		"ok: tag format v0.1.0",
		"ok: working tree is clean",
		"ok: current branch main (default)",
		"ok: tag v0.1.0 does not exist locally",
		"ok: HEAD is the remote tip of main",
		"ok: tag v0.1.0 does not exist on origin",
		"pushed HEAD:refs/heads/main and refs/tags/v0.1.0 to origin",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout에 %q 없음:\n%s", want, out)
		}
	}
	if got := gitIn(t, work, "log", "-1", "--format=%s"); got != "release: v0.1.0" {
		t.Errorf("릴리즈 커밋 제목 = %q", got)
	}
	if got := gitIn(t, work, "cat-file", "-t", "refs/tags/v0.1.0"); got != "tag" {
		t.Errorf("annotated tag 기대, got %s", got)
	}
	if got := gitIn(t, work, "ls-remote", "--tags", "origin", "refs/tags/v0.1.0"); !strings.Contains(got, "refs/tags/v0.1.0") {
		t.Errorf("원격 tag 부재, got %q", got)
	}
}

func TestE2EReleasePrepareCheckFailures(t *testing.T) {
	bin := buildGG(t)

	t.Run("tag 형식", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "1.2.3")
		if code != 1 || !strings.Contains(stderr, `invalid release tag "1.2.3"`) {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("지저분한 working tree", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		if err := os.WriteFile(filepath.Join(work, "dirty.txt"), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, "working tree is dirty") {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("default branch가 아님", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		gitIn(t, work, "checkout", "-q", "-b", "feature")
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, `is not the default branch "main"`) {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("로컬 tag 이미 존재", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		gitIn(t, work, "tag", "v0.1.0")
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, "already exists locally") {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("HEAD가 원격 tip이 아님", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		if err := os.WriteFile(filepath.Join(work, "ahead.txt"), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitIn(t, work, "add", ".")
		gitIn(t, work, "commit", "-qm", "ahead")
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, "is not the remote tip of main") {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("원격 tag 이미 존재", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		// 원격에만 tag가 있고 로컬에는 없는 상태를 만든다.
		gitIn(t, work, "tag", "v0.1.0")
		gitIn(t, work, "push", "-q", "origin", "refs/tags/v0.1.0")
		gitIn(t, work, "tag", "-d", "v0.1.0")
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, "already exists on origin") {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})

	t.Run("origin/HEAD 부재", func(t *testing.T) {
		work := releasePrepareOrigin(t)
		gitIn(t, work, "remote", "set-head", "origin", "-d")
		_, stderr, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0")
		if code != 1 || !strings.Contains(stderr, "git remote set-head origin --auto") {
			t.Fatalf("exit %d: %s", code, stderr)
		}
	})
}

func TestE2EReleasePrepareExplainChangesNothing(t *testing.T) {
	bin := buildGG(t)
	work := releasePrepareOrigin(t)
	before := gitIn(t, work, "rev-parse", "HEAD")

	out, _, code := runGGStreams(t, bin, work, "release", "prepare", "v0.1.0", "--explain")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{"release prepare v0.1.0", "git commit --no-gpg-sign --allow-empty", "git tag -a", "git push --atomic origin"} {
		if !strings.Contains(out, want) {
			t.Errorf("explain에 %q 없음:\n%s", want, out)
		}
	}
	if got := gitIn(t, work, "rev-parse", "HEAD"); got != before {
		t.Errorf("explain은 아무 것도 만들지 않는다: %s -> %s", before, got)
	}
	if got := gitIn(t, work, "tag"); got != "" {
		t.Errorf("explain은 tag를 만들지 않는다, got %q", got)
	}
}

func TestE2EReleasePrepareRejectsContextFlags(t *testing.T) {
	bin := buildGG(t)
	work := releasePrepareOrigin(t)

	for _, args := range [][]string{
		{"--repo", "https://github.com/o/r", "release", "prepare", "v0.1.0"},
		{"release", "prepare", "v0.1.0", "--repo", "https://github.com/o/r"},
		{"release", "prepare", "v0.1.0", "--remote", "upstream"},
	} {
		_, stderr, code := runGGStreams(t, bin, work, args...)
		if code != 2 {
			t.Errorf("gg %v: exit %d, want 2: %s", args, code, stderr)
		}
		if !strings.Contains(stderr, "is not supported for release prepare") {
			t.Errorf("gg %v: stderr = %q", args, stderr)
		}
	}
}
