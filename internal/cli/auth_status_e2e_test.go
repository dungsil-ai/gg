package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// TestE2EAuthStatusShowsProviderTable은 기본 domain과 Provider 설정 host의
// 로그인 상태가 host 정렬 표로 나오는지 본다(ADR 0006).
func TestE2EAuthStatusShowsProviderTable(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeStatusBin(t, fakeDir, "gh", logFile, `{"hosts":{"github.com":[{"login":"dungsil","active":true}]}}`)
	writeFakeStatusBin(t, fakeDir, "glab", logFile, "yes")
	writeFakeStatusBin(t, fakeDir, "tea", logFile, `[{"name":"my-login","url":"https://git.example.com"}]`)
	ggHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(ggHome, "config.json"), []byte(`{"hosts":{"git.example.com":"tea"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runGGWithHome(t, bin, fakeDir, t.TempDir(), ggHome, "auth", "status")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := "HOST             PROVIDER  LOGIN\n" +
		"git.example.com  tea       my-login\n" +
		"gitea.com        tea       no\n" +
		"github.com       gh        dungsil\n" +
		"gitlab.com       glab      yes\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	// host당 provider CLI 호출이 정확히 한 번이다. Windows cmd echo가
	// redirection 앞 공백을 남기므로 행별로 정돈해 비교한다.
	got := normalizeLines(readLog(t, logFile))
	wantLog := "tea logins list --output json\n" +
		"tea logins list --output json\n" +
		"gh auth status --hostname github.com --json hosts\n" +
		"glab auth status --hostname gitlab.com"
	if got != wantLog {
		t.Fatalf("provider argv log = %q, want %q", got, wantLog)
	}
}

// normalizeLines는 로그 비교를 위해 각 행을 trim하고 \n으로 맞춘다.
func normalizeLines(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

// TestE2EAuthStatusRowFailuresAreValues는 행별 조회 실패와 미로그인이
// 전체 실패가 아니라 LOGIN 값(no, no cli)으로 나오는지 본다.
func TestE2EAuthStatusRowFailuresAreValues(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeStatusBin(t, fakeDir, "gh", logFile, `{"hosts":{"github.com":[{"login":"dungsil","active":true}]}}`)
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(fakeDir, "glab.cmd")
		body = "@echo off\r\nexit /b 1\r\n"
	} else {
		path = filepath.Join(fakeDir, "glab")
		body = "#!/bin/sh\nexit 1\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runGG(t, bin, fakeDir, t.TempDir(), "auth", "status")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := []string{"HOST", "PROVIDER", "LOGIN",
		"gitea.com", "tea", "no", "cli",
		"github.com", "gh", "dungsil",
		"gitlab.com", "glab", "no",
	}
	if got := strings.Fields(out); !slices.Equal(got, want) {
		t.Fatalf("stdout fields = %q, want %q", got, want)
	}
}

// TestE2EAuthStatusWithoutProviderCLIsExitZero는 provider CLI가 하나도
// 없어도 표 조회가 성공(exit 0)하는 계약을 본다.
func TestE2EAuthStatusWithoutProviderCLIsExitZero(t *testing.T) {
	bin := buildGG(t)
	out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), t.TempDir(), "auth", "status")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := []string{"HOST", "PROVIDER", "LOGIN",
		"gitea.com", "tea", "no", "cli",
		"github.com", "gh", "no", "cli",
		"gitlab.com", "glab", "no", "cli",
	}
	if got := strings.Fields(out); !slices.Equal(got, want) {
		t.Fatalf("stdout fields = %q, want %q", got, want)
	}
}

// TestE2EAuthStatusBrokenConfigFails는 조회 자체 실패(손상된 config.json)가
// exit code 1이 되는 계약을 본다.
func TestE2EAuthStatusBrokenConfigFails(t *testing.T) {
	bin := buildGG(t)
	ggHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(ggHome, "config.json"), []byte(`{"hosts":{"git.example.com":"gh"`), 0o600); err != nil {
		t.Fatal(err)
	}
	out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), ggHome, "auth", "status")
	if code != 1 {
		t.Fatalf("exit = %d, want 1: %s", code, out)
	}
	if !strings.Contains(out, "broken config") {
		t.Errorf("output에 broken config 오류 없음: %s", out)
	}
}

func TestE2EAuthStatusRejectsRepositoryContextFlags(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeStatusBin(t, fakeDir, "gh", logFile, `{"hosts":{}}`)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--repo", "https://github.com/o/r", "auth", "status"}, "--repo is not supported for auth status"},
		{[]string{"auth", "status", "--repo", "https://github.com/o/r"}, "--repo is not supported for auth status"},
		{[]string{"--remote", "origin", "auth", "status"}, "--remote is not supported for auth status"},
		{[]string{"auth", "status", "--remote", "origin"}, "--remote is not supported for auth status"},
		{[]string{"--explain", "auth", "status"}, "--explain is not supported for auth status"},
		{[]string{"auth", "status", "--explain"}, "--explain is not supported for auth status"},
	}
	for _, tc := range cases {
		out, code := runGG(t, bin, fakeDir, t.TempDir(), tc.args...)
		if code != 2 || !strings.Contains(out, tc.want) {
			t.Errorf("gg %v = exit %d, output %s; want exit 2 with %q", tc.args, code, out, tc.want)
		}
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("provider CLI should not run, got %q", got)
	}
}

func TestE2EAuthStatusUsageErrors(t *testing.T) {
	bin := buildGG(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"auth"}, "auth needs an action: status"},
		{[]string{"auth", "status", "extra"}, "usage: gg auth status"},
		{[]string{"auth", "unsupported"}, "auth does not support unsupported"},
	}
	for _, tc := range cases {
		out, code := runGGWithoutProviderCLIs(t, bin, t.TempDir(), t.TempDir(), tc.args...)
		if code != 2 || !strings.Contains(out, tc.want) {
			t.Errorf("gg %v = exit %d, output %s; want exit 2 with %q", tc.args, code, out, tc.want)
		}
	}
}

func TestE2EAuthHelpOmitsRepositoryContextFlags(t *testing.T) {
	bin := buildGG(t)
	assertGGHelp(t, bin, []string{"auth", "--help"}, []string{
		"Show provider CLI login status.", "Usage:", "gg auth <command>",
		"status", "Show login status for each host", "Flags:", "--help",
	})
	assertGGHelp(t, bin, []string{"auth", "status", "--help"}, []string{
		"Show login status for each host.", "Usage:", "gg auth status", "Flags:", "--help",
	})
	assertGGHelpOmits(t, bin, []string{"auth", "--help"}, "--repo")
	assertGGHelpOmits(t, bin, []string{"auth", "status", "--help"}, "--repo")
}
