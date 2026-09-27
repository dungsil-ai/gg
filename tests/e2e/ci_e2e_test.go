package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2ECIInvocations는 gg ci의 실제 gh/glab argv를 본다.
func TestE2ECIInvocations(t *testing.T) {
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
		{[]string{"ci", "list"}, ghRepo, wantCall("gh", "run", "list", "-R", "github.com/o/r")},
		{[]string{"ci", "list", "--limit", "5"}, ghRepo, wantCall("gh", "run", "list", "-R", "github.com/o/r", "--limit", "5")},
		{[]string{"ci", "list", "--branch", "main"}, ghRepo, wantCall("gh", "run", "list", "-R", "github.com/o/r", "--branch", "main")},
		{[]string{"ci", "view", "123"}, ghRepo, wantCall("gh", "run", "view", "123", "-R", "github.com/o/r")},
		{[]string{"ci", "watch", "123"}, ghRepo, wantCall("gh", "run", "watch", "123", "-R", "github.com/o/r")},
		{[]string{"ci", "retry", "123"}, ghRepo, wantCall("gh", "run", "rerun", "123", "-R", "github.com/o/r")},
		{[]string{"ci", "cancel", "123"}, ghRepo, wantCall("gh", "run", "cancel", "123", "-R", "github.com/o/r")},
		{[]string{"ci", "list"}, glabRepo, wantCall("glab", "ci", "list", "--repo", "https://gitlab.com/o/r")},
		{[]string{"ci", "list", "--branch", "main", "--limit", "5"}, glabRepo, wantCall("glab", "ci", "list", "--repo", "https://gitlab.com/o/r", "--ref", "main", "--per-page", "5")},
		{[]string{"ci", "view", "123"}, glabRepo, wantCall("glab", "ci", "get", "--pipeline-id", "123", "--repo", "https://gitlab.com/o/r")},
		{[]string{"ci", "watch", "123"}, glabRepo, wantCall("glab", "ci", "trace", "123", "--repo", "https://gitlab.com/o/r")},
		{[]string{"ci", "retry", "123"}, glabRepo, wantCall("glab", "ci", "retry", "123", "--repo", "https://gitlab.com/o/r")},
		{[]string{"ci", "cancel", "123"}, glabRepo, wantCall("glab", "ci", "cancel", "pipeline", "123", "--repo", "https://gitlab.com/o/r")},
		// alias: actions는 ci와 같은 invocation을 낸다
		{[]string{"actions", "list", "--limit", "3"}, ghRepo, wantCall("gh", "run", "list", "-R", "github.com/o/r", "--limit", "3")},
		{[]string{"actions", "cancel", "123"}, glabRepo, wantCall("glab", "ci", "cancel", "pipeline", "123", "--repo", "https://gitlab.com/o/r")},
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

// TestE2ECIAliasHelpMatchesCanonical은 gg actions --help가 gg ci --help와
// 같은 출력을 내는지 본다.
func TestE2ECIAliasHelpMatchesCanonical(t *testing.T) {
	bin := buildGG(t)
	aliasOut, aliasErr, aliasCode := runGGStreams(t, bin, t.TempDir(), "actions", "--help")
	canonOut, canonErr, canonCode := runGGStreams(t, bin, t.TempDir(), "ci", "--help")
	if aliasCode != 0 || canonCode != 0 {
		t.Fatalf("help exit = %d(alias), %d(canon)", aliasCode, canonCode)
	}
	if aliasErr != "" || canonErr != "" {
		t.Fatalf("help stderr = %q(alias), %q(canon)", aliasErr, canonErr)
	}
	if aliasOut != canonOut {
		t.Errorf("gg actions --help가 gg ci --help와 다름:\n%q\n%q", aliasOut, canonOut)
	}
	if !strings.Contains(aliasOut, "watch") || !strings.Contains(aliasOut, "cancel") {
		t.Errorf("gg ci --help에 action 목록 없음:\n%s", aliasOut)
	}
}

// TestE2ECIGiteaUnsupported는 gitea remote에서 gg ci가 tea를 실행하지 않고
// usage error로 끝나는지 본다.
func TestE2ECIGiteaUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "ci", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2: %s", code, out)
	}
	if !strings.Contains(out, "ci is not supported for tea") {
		t.Errorf("output: %s", out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea should not run, got %q", got)
	}
}
