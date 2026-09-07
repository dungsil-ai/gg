package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EAuthRelayArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name   string
		remote string
		args   []string
		want   string
	}{
		{
			name:   "auth token",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "token"},
			want:   "gh auth token",
		},
		{
			name:   "auth setup-git",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "setup-git"},
			want:   "gh auth setup-git",
		},
		{
			name:   "auth login with args",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "login", "--hostname", "git.example.com", "--with-token"},
			want:   "gh auth login --hostname git.example.com --with-token",
		},
		{
			name:   "auth logout with hostname",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "logout", "--hostname", "git.example.com"},
			want:   "gh auth logout --hostname git.example.com",
		},
		{
			name:   "auth refresh",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "refresh"},
			want:   "gh auth refresh",
		},
		{
			name:   "auth switch with user",
			remote: "https://github.com/o/r.git",
			args:   []string{"auth", "switch", "--user", "dungsil"},
			want:   "gh auth switch --user dungsil",
		},
		{
			name:   "auth token outside repo",
			remote: "",
			args:   []string{"auth", "token"},
			want:   "gh auth token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
			var workDir string
			if tc.remote == "" {
				workDir = t.TempDir()
			} else {
				workDir = tempRepo(t, tc.remote)
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

func TestE2EAuthRelayHelpPassesThrough(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	// stdout으로 help 텍스트를 내보내는 fake gh
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "gh auth login help", "", 0)
	workDir := t.TempDir()

	out, code := runGG(t, bin, fakeDir, workDir, "auth", "login", "--help")
	if code != 0 {
		t.Fatalf("gg auth login --help: exit %d: %s", code, out)
	}
	if !strings.Contains(out, "gh auth login help") {
		t.Errorf("--help가 gh로 전달되지 않음: %s", out)
	}
	if got := readLog(t, logFile); got != "gh auth login --help" {
		t.Errorf("argv = %q, want %q", got, "gh auth login --help")
	}
}

func TestE2EAuthRelayUsageErrors(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "relay rejects leading --repo",
			args: []string{"--repo", "https://github.com/o/r", "auth", "token"},
			want: "--repo is not supported for auth token",
		},
		{
			name: "relay rejects leading --remote",
			args: []string{"--remote", "origin", "auth", "token"},
			want: "--remote is not supported for auth token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("args %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("args %v: stderr = %q, want substring %q", tc.args, stderr, tc.want)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("fake provider should not be called, got: %q", got)
			}
		})
	}
}

func TestE2EAuthStatusStillWorks(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeReadyBin(t, fakeDir, "gh", logFile, "", "", 0)
	workDir := t.TempDir()

	out, code := runGG(t, bin, fakeDir, workDir, "auth", "status")
	if code != 0 {
		t.Fatalf("gg auth status: exit %d: %s", code, out)
	}
	if !strings.Contains(out, "HOST") || !strings.Contains(out, "PROVIDER") {
		t.Errorf("auth status 표 없음: %s", out)
	}
	// auth status는 내부적으로 로그인 조회를 위해 gh를 실행한다(릴레이와 무관).
	if got := readLog(t, logFile); got != "" && !strings.Contains(got, "gh auth status") {
		t.Errorf("auth status 외 실행이 있으면 안 된다, got %q", got)
	}
}
