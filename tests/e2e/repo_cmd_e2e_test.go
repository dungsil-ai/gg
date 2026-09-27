package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ERepoLifecycleInvocations(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"fork", []string{"repo", "fork"}, wantCall("gh", "repo", "fork", "https://github.com/o/r")},
		{"delete", []string{"repo", "delete", "--yes"}, wantCall("gh", "repo", "delete", "https://github.com/o/r", "--yes")},
		{"edit 설명", []string{"repo", "edit", "--description", "hello"}, wantCall("gh", "repo", "edit", "https://github.com/o/r", "--description", "hello")},
		{"edit 가시성", []string{"repo", "edit", "--private"}, wantCall("gh", "repo", "edit", "https://github.com/o/r", "--visibility", "private", "--accept-visibility-change-consequences")},
		{"rename", []string{"repo", "rename", "newname", "--yes"}, wantCall("gh", "repo", "rename", "newname", "-R", "github.com/o/r", "--yes")},
		{"sync", []string{"repo", "sync", "--source", "o/up", "--force"}, wantCall("gh", "repo", "sync", "https://github.com/o/r", "--source", "o/up", "--force")},
		{"sync source 없으면 로컬 동기화", []string{"repo", "sync", "--branch", "main", "--force"}, wantCall("gh", "repo", "sync", "--branch", "main", "--force")},
		{"sync 무 flag 로컬 동기화", []string{"repo", "sync"}, wantCall("gh", "repo", "sync")},
		{"set-default", []string{"repo", "set-default"}, wantCall("gh", "repo", "set-default", "https://github.com/o/r")},
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

	t.Run("set-default unset과 view는 문맥 조회 없이 gh로 바로 간다", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			want string
		}{
			{[]string{"repo", "set-default", "--unset"}, wantCall("gh", "repo", "set-default", "--unset")},
			{[]string{"repo", "set-default", "--view"}, wantCall("gh", "repo", "set-default", "--view")},
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
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"fork", []string{"repo", "fork"}, wantTeaCall("repos", "fork", "--login", "pub", "--repo", "o/r")},
		{"delete", []string{"repo", "delete", "--yes"}, wantTeaCall("repos", "delete", "--login", "pub", "--owner", "o", "--name", "r", "--force")},
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

// tea에 대응 하위 명령이 없는 repo action은 tea login이 없어도 login 요구 대신
// 미지원 오류로 거부된다 — login을 추가해 다시 시도해도 같은 미지원 오류가
// 나므로 헛수고를 막는 것이 contributors·transfer·mirror와 같은 원칙이다.
func TestE2ERepoLifecycleUnsupportedForTea(t *testing.T) {
	bin := buildGG(t)
	// tea 바이너리를 PATH에 두지 않아 login 조회가 실패하는 상태에서도
	// 미지원이 먼저 확정되는지 본다.
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	writeFakeBin(t, fakeDir, "glab", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

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
		if strings.Contains(out, "no tea login") {
			t.Errorf("gg %v: login 요구가 먼저 나오면 안 된다: %s", tc.args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child command should not run, got %q", tc.args, got)
		}
	}
}

// glab repo delete는 positional slug로 host를 정하지 않으므로 GITLAB_HOST로
// 대상 인스턴스를 고정해야 한다. 자가 호스팅에서 gitlab.com으로 삭제 요청이
// 향하는 것을 막는다.
func TestE2ERepoDeleteGitLabHostEnv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeCLI(t, fakeDir, "glab", fakeCLIConfig{LogFile: logFile, EnvKeys: []string{"GITLAB_HOST"}})
	repo := tempRepo(t, "https://git.example.com/grp/p.git")

	out, code := runGG(t, bin, fakeDir, repo, "repo", "delete", "--yes")
	if code != 0 {
		t.Fatalf("gg repo delete: exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "GITLAB_HOST=git.example.com") {
		t.Errorf("GITLAB_HOST 주입 없음: %q", got)
	}
	if !strings.Contains(got, wantCall("glab", "repo", "delete", "grp/p", "--yes")) {
		t.Errorf("glab argv 예상과 다름: %q", got)
	}
}
