package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EReleaseInvocations는 gg release의 실제 gh/glab argv를 본다.
func TestE2EReleaseInvocations(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "gh", logFile)
	writeFakeBin(t, fakeDir, "glab", logFile)
	ghRepo := tempRepo(t, "https://github.com/o/r.git")
	glabRepo := tempRepo(t, "https://gitlab.com/o/r.git")

	cases := []struct {
		args []string
		dir  string
		want string
	}{
		{[]string{"release", "list"}, ghRepo, wantCall("gh", "release", "list", "-R", "github.com/o/r")},
		{[]string{"release", "list", "--limit", "5"}, ghRepo, wantCall("gh", "release", "list", "-R", "github.com/o/r", "--limit", "5")},
		{[]string{"release", "view", "v1.0.0"}, ghRepo, wantCall("gh", "release", "view", "v1.0.0", "-R", "github.com/o/r")},
		{[]string{"release", "view"}, ghRepo, wantCall("gh", "release", "view", "-R", "github.com/o/r")},
		{[]string{"release", "create", "v1.0.0", "--title", "t", "--notes", "n", "--draft"}, ghRepo,
			wantCall("gh", "release", "create", "v1.0.0", "--title", "t", "--notes", "n", "--draft", "-R", "github.com/o/r")},
		{[]string{"release", "delete", "v1.0.0", "--yes"}, ghRepo, wantCall("gh", "release", "delete", "v1.0.0", "-R", "github.com/o/r", "--yes")},
		{[]string{"release", "edit", "v1.0.0", "--draft=false", "--title", "t"}, ghRepo,
			wantCall("gh", "release", "edit", "v1.0.0", "-R", "github.com/o/r", "--title", "t", "--draft=false")},
		{[]string{"release", "upload", "v1.0.0", "a.zip"}, ghRepo, wantCall("gh", "release", "upload", "v1.0.0", "a.zip", "-R", "github.com/o/r")},
		{[]string{"release", "download"}, ghRepo, wantCall("gh", "release", "download", "--pattern", "*", "-R", "github.com/o/r")},
		{[]string{"release", "list"}, glabRepo, wantCall("glab", "release", "list", "--repo", "https://gitlab.com/o/r")},
		{[]string{"release", "create", "v1.0.0", "a.zip", "--ref", "main"}, glabRepo,
			wantCall("glab", "release", "create", "v1.0.0", "a.zip", "--ref", "main", "--repo", "https://gitlab.com/o/r")},
		{[]string{"release", "download", "--pattern", "*.zip", "--dir", "dist"}, glabRepo,
			wantCall("glab", "release", "download", "--asset-name", "*.zip", "--dir", "dist", "--repo", "https://gitlab.com/o/r")},
	}
	for _, tc := range cases {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, tc.dir, tc.args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
		}
		if got := readLog(t, logFile); got != tc.want {
			t.Errorf("gg %v argv = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestE2EReleaseGiteaArgv는 gitea remote에서 중계되는 release action과
// 여전히 미지원인 action을 본다.
func TestE2EReleaseGiteaArgv(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeTeaWithLogin(t, fakeDir, logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	relays := []struct {
		args []string
		want string
	}{
		{[]string{"release", "list"}, wantTeaCall("releases", "list", "--login", "pub", "--repo", "o/r")},
		{[]string{"release", "list", "--limit", "5"}, wantTeaCall("releases", "list", "--login", "pub", "--repo", "o/r", "--limit", "5")},
		{[]string{"release", "create", "v1.0.0", "--title", "t", "--notes", "n", "--ref", "main"}, wantTeaCall("releases", "create", "--login", "pub", "--repo", "o/r", "--tag", "v1.0.0", "--title", "t", "--note", "n", "--target", "main")},
		{[]string{"release", "create", "v1.0.0", "a.zip"}, wantTeaCall("releases", "create", "--login", "pub", "--repo", "o/r", "--tag", "v1.0.0", "--asset", "a.zip")},
		{[]string{"release", "delete", "v1.0.0", "--yes", "--cleanup-tag"}, wantTeaCall("releases", "delete", "v1.0.0", "--login", "pub", "--repo", "o/r", "--confirm", "--delete-tag")},
		{[]string{"release", "edit", "v1.0.0", "--title", "t2"}, wantTeaCall("releases", "edit", "v1.0.0", "--login", "pub", "--repo", "o/r", "--title", "t2")},
		{[]string{"release", "edit", "v1.0.0", "--draft=true"}, wantTeaCall("releases", "edit", "v1.0.0", "--login", "pub", "--repo", "o/r", "--draft=true")},
		{[]string{"release", "edit", "v1.0.0", "--draft=false", "--prerelease=true"}, wantTeaCall("releases", "edit", "v1.0.0", "--login", "pub", "--repo", "o/r", "--draft=false", "--prerelease=true")},
	}
	for _, tc := range relays {
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
	}

	// view·download·upload·delete-asset은 여전히 미지원이다.
	for _, args := range [][]string{
		{"release", "view", "v1.0.0"},
		{"release", "download"},
		{"release", "upload", "v1.0.0", "a.zip"},
		{"release", "delete-asset", "v1.0.0", "a.zip"},
	} {
		if err := os.WriteFile(logFile, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "does not support") {
			t.Errorf("gg %v output에 미지원 오류 없음: %s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v tea should not run, got %q", args, got)
		}
	}

	// edit의 draft·prerelease는 true|false 값이 필요하다.
	for _, args := range [][]string{
		{"release", "edit", "v1.0.0", "--draft"},
		{"release", "edit", "v1.0.0", "--draft=maybe"},
		{"release", "edit", "v1.0.0", "--prerelease=yes"},
	} {
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 2 {
			t.Fatalf("gg %v: exit = %d, want 2: %s", args, code, out)
		}
		if !strings.Contains(out, "--draft") && !strings.Contains(out, "--prerelease") {
			t.Errorf("gg %v output에 draft·prerelease 안내 없음: %s", args, out)
		}
	}
}
