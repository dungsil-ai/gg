package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2EAPIRelayArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "gh api relay",
			remote: "https://github.com/o/r.git",
			args:   []string{"api", "repos/o/r/issues/1"},
			want:   wantCall("gh", "api", "repos/o/r/issues/1"),
		},
		{
			name:   "gh api relay with flags",
			remote: "https://github.com/o/r.git",
			args:   []string{"api", "--method", "POST", "repos/o/r/issues", "-f", "title=t"},
			want:   wantCall("gh", "api", "--method", "POST", "repos/o/r/issues", "-f", "title=t"),
		},
		{
			name:   "glab api relay",
			remote: "https://gitlab.com/o/r.git",
			args:   []string{"api", "projects/o%2Fr/issues/1"},
			want:   wantCall("glab", "api", "projects/o%2Fr/issues/1"),
		},
		{
			name:   "tea api relay",
			remote: "https://gitea.com/o/r.git",
			args:   []string{"api", "repos/o/r/issues/1"},
			want:   wantTeaCall("api", "--login", "pub", "--repo", "o/r", "repos/o/r/issues/1"),
		},
		{
			name:   "gh api with context repo flag",
			remote: "",
			args:   []string{"--repo", "https://github.com/custom/repo", "api", "repos/custom/repo/issues/1"},
			want:   wantCall("gh", "api", "repos/custom/repo/issues/1"),
		},
		{
			name:   "gh api with context remote flag",
			remote: "",
			args:   []string{"--remote", "upstream", "api", "repos/o/upstream/issues/1"},
			want:   wantCall("gh", "api", "repos/o/upstream/issues/1"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			workDir := tempRepo(t, tc.remote)
			if tc.remote == "" {
				workDir = tempRepoWithUpstream(t)
			}

			out, code := runGG(t, bin, fakeDir, workDir, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestE2EAPIRelayNoRepoContext(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
	workDir := t.TempDir()

	out, code := runGG(t, bin, fakeDir, workDir, "api", "repos/o/r/issues/1")
	if code != 1 {
		t.Fatalf("exit = %d, want 1: %s", code, out)
	}
	if !strings.Contains(out, "not a git repository") && !strings.Contains(out, "cannot pick a remote") {
		t.Errorf("output에 문맥 오류 없음: %s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("gh should not run, got %q", got)
	}
}

func TestE2EAPIRelayPreservesArgumentBoundaries(t *testing.T) {
	bin := buildGG(t)
	for _, tc := range []struct {
		provider string
		remote   string
		want     string
	}{
		{"gh", "https://github.com/o/r.git", wantCall("gh", "api", "repos/o/r/issues", "-f", "title=hello world", "-f", "body=line one\nline two", "--header", "", "--header", "X-Quote: \"quoted\"")},
		{"glab", "https://gitlab.com/o/r.git", wantCall("glab", "api", "repos/o/r/issues", "-f", "title=hello world", "-f", "body=line one\nline two", "--header", "", "--header", "X-Quote: \"quoted\"")},
		{"tea", "https://gitea.com/o/r.git", wantTeaCall("api", "--login", "pub", "--repo", "o/r", "repos/o/r/issues", "-f", "title=hello world", "-f", "body=line one\nline two", "--header", "", "--header", "X-Quote: \"quoted\"")},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			if tc.provider == "tea" {
				writeFakeTeaWithLogin(t, fakeDir, logFile)
			} else {
				writeFakeBin(t, fakeDir, tc.provider, logFile)
			}
			repo := tempRepo(t, tc.remote)
			out, code := runGG(t, bin, fakeDir, repo, "api", "repos/o/r/issues", "-f", "title=hello world", "-f", "body=line one\nline two", "--header", "", "--header", "X-Quote: \"quoted\"")
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if got := readLog(t, logFile); got != tc.want {
				t.Errorf("provider argv = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestE2EAPIRelayHostEnvInjection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("환경변수 로그는 unix fake에서만 검증한다")
	}
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	// GITLAB_HOST env를 로그에 남기는 fake glab
	path := filepath.Join(fakeDir, "glab")
	body := "#!/bin/sh\necho \"GITLAB_HOST=$GITLAB_HOST\" >> \"" + logFile + "\"\necho \"glab $@\" >> \"" + logFile + "\"\nexit 0\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://git.example.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "api", "projects/o%2Fr/issues/1")
	if code != 0 {
		t.Fatalf("gg api: exit %d: %s", code, out)
	}
	got := readLog(t, logFile)
	if !strings.Contains(got, "GITLAB_HOST=git.example.com") {
		t.Errorf("GITLAB_HOST 주입 없음: %q", got)
	}
}
