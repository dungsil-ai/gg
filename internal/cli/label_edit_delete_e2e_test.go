package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ELabelEditDeleteArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "gh label edit color",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "edit", "bug", "--color", "00ff00"},
			want:     wantCall("gh", "label", "edit", "bug", "-R", "github.com/o/r", "--color", "00ff00"),
		},
		{
			name:     "gh label edit rename",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "edit", "bug", "--name", "defect"},
			want:     wantCall("gh", "label", "edit", "bug", "-R", "github.com/o/r", "--name", "defect"),
		},
		{
			name:     "gh label edit full",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "edit", "bug", "--name", "defect", "--color", "00ff00", "--description", "버그"},
			want:     wantCall("gh", "label", "edit", "bug", "-R", "github.com/o/r", "--name", "defect", "--color", "00ff00", "--description", "버그"),
		},
		{
			name:     "gh label delete",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "delete", "bug"},
			want:     wantCall("gh", "label", "delete", "bug", "-R", "github.com/o/r"),
		},
		{
			name:     "gh label delete with yes",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"label", "delete", "bug", "--yes"},
			want:     wantCall("gh", "label", "delete", "bug", "--yes", "-R", "github.com/o/r"),
		},
		{
			name:     "gh label edit repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "label", "edit", "bug", "--color", "00ff00"},
			want:     wantCall("gh", "label", "edit", "bug", "-R", "github.com/custom/repo", "--color", "00ff00"),
		},
		{
			name:     "gh label delete remote flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"label", "delete", "bug", "--remote", "upstream", "--yes"},
			want:     wantCall("gh", "label", "delete", "bug", "--yes", "-R", "github.com/o/upstream"),
		},
		{
			name:     "glab label delete",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"label", "delete", "bug"},
			want:     wantCall("glab", "label", "delete", "bug", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name:     "glab label delete with yes",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"label", "delete", "bug", "--yes"},
			want:     wantCall("glab", "label", "delete", "bug", "--repo", "https://gitlab.com/o/r"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeReadyBin(t, fakeDir, tc.fakeName, logFile, "", "", 0)
			workDir := t.TempDir()
			if tc.remote != "" {
				workDir = tempRepo(t, tc.remote)
			} else {
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

// numeric label id를 요구하는 glab·tea는 gg가 이름→id를 미리 조회한다.
// 조회 api 호출과 실제 명령 호출이 순서대로 기록되는지 본다.
func TestE2ELabelEditDeleteIDResolution(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		stdout   string // label 목록 api 응답
		login    bool   // tea logins csv 응답 여부
		args     []string
		calls    []string
	}{
		{
			name:     "glab edit resolves label id",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			stdout:   `[{"id":7,"name":"bug"}]`,
			args:     []string{"label", "edit", "bug", "--color", "00ff00"},
			calls: []string{
				wantCall("glab", "api", "projects/o%2Fr/labels"),
				wantCall("glab", "label", "edit", "--label-id", "7", "--color", "00ff00", "--repo", "https://gitlab.com/o/r"),
			},
		},
		{
			name:     "tea edit resolves label id",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			stdout:   `[{"id":9,"name":"bug"}]`,
			login:    true,
			args:     []string{"label", "edit", "bug", "--name", "defect"},
			calls: []string{
				wantCall("tea", "logins", "list", "--output", "csv"),
				wantCall("tea", "api", "repos/o/r/labels", "--login", "pub", "--repo", "o/r"),
				wantCall("tea", "labels", "update", "--id", "9", "--login", "pub", "--repo", "o/r", "--name", "defect"),
			},
		},
		{
			name:     "tea delete resolves label id",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			stdout:   `[{"id":9,"name":"bug"}]`,
			login:    true,
			args:     []string{"label", "delete", "bug", "--yes"},
			calls: []string{
				wantCall("tea", "logins", "list", "--output", "csv"),
				wantCall("tea", "api", "repos/o/r/labels", "--login", "pub", "--repo", "o/r"),
				wantCall("tea", "labels", "delete", "--id", "9", "--login", "pub", "--repo", "o/r"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeCLI(t, fakeDir, tc.fakeName, fakeCLIConfig{LogFile: logFile, TeaLogin: tc.login, Stdout: tc.stdout})
			repo := tempRepo(t, tc.remote)

			out, code := runGG(t, bin, fakeDir, repo, tc.args...)
			if code != 0 {
				t.Fatalf("gg %v: exit %d: %s", tc.args, code, out)
			}
			got := readLog(t, logFile)
			for _, want := range tc.calls {
				if !strings.Contains(got, want) {
					t.Errorf("호출 기록에 %q 없음:\n%s", want, got)
				}
			}
		})
	}
}

// 로그인이 없거나 목록에 없는 label은 오류로 끝난다.
func TestE2ELabelEditDeleteIDResolutionFailures(t *testing.T) {
	t.Run("glab edit without login", func(t *testing.T) {
		bin := buildGG(t)
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeCLI(t, fakeDir, "glab", fakeCLIConfig{LogFile: logFile, Stderr: "no auth", ExitCode: 1})
		repo := tempRepo(t, "https://gitlab.com/o/r.git")

		out, code := runGG(t, bin, fakeDir, repo, "label", "edit", "bug", "--color", "00ff00")
		if code == 0 {
			t.Fatalf("api 실패 시 종료 코드 0이어야 안 됨: %s", out)
		}
		if !strings.Contains(out, "cannot look up label") {
			t.Errorf("조회 실패 안내 기대, got %s", out)
		}
	})

	t.Run("tea edit label not found", func(t *testing.T) {
		bin := buildGG(t)
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeCLI(t, fakeDir, "tea", fakeCLIConfig{LogFile: logFile, TeaLogin: true, Stdout: `[{"id":9,"name":"bug"}]`})
		repo := tempRepo(t, "https://gitea.com/o/r.git")

		out, code := runGG(t, bin, fakeDir, repo, "label", "edit", "missing", "--color", "00ff00")
		if code != 1 {
			t.Fatalf("exit = %d, want 1: %s", code, out)
		}
		if !strings.Contains(out, `label "missing" not found`) {
			t.Errorf("label 부재 안내 기대, got %s", out)
		}
	})
}

func TestE2ELabelEditUsageErrors(t *testing.T) {
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
			name: "edit missing name",
			args: []string{"label", "edit"},
			want: "usage: gg label edit <name>",
		},
		{
			name: "edit without flags",
			args: []string{"label", "edit", "bug"},
			want: "label edit needs --name, --color, or --description",
		},
		{
			name: "edit with blank flags",
			args: []string{"label", "edit", "bug", "--name", "  ", "--color", " ", "--description", " "},
			want: "label edit needs --name, --color, or --description",
		},
		{
			name: "edit unknown flag",
			args: []string{"label", "edit", "bug", "--color", "00ff00", "--invalid"},
			want: "unknown flag",
		},
		{
			name: "delete missing name",
			args: []string{"label", "delete"},
			want: "usage: gg label delete <name>",
		},
		{
			name: "delete too many positional args",
			args: []string{"label", "delete", "bug", "extra"},
			want: "usage: gg label delete <name>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearFile(t, logFile)
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("args %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if stdout != "" {
				t.Errorf("args %v: stdout = %q, want empty", tc.args, stdout)
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

func TestE2ELabelEditDeleteExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "label", "edit", "bug", "--color", "00ff00"},
		{"label", "edit", "bug", "--color", "00ff00", "--explain"},
		{"label", "delete", "bug", "--explain"},
	} {
		clearFile(t, logFile)
		out, code := runGG(t, bin, fakeDir, repo, args...)
		if code != 0 {
			t.Fatalf("gg %v: exit %d: %s", args, code, out)
		}
		if !strings.Contains(out, "Provider: gh") || !strings.Contains(out, "CLI: gh") {
			t.Errorf("gg %v output unexpected:\n%s", args, out)
		}
		if got := readLog(t, logFile); got != "" {
			t.Errorf("gg %v child should not run, got: %q", args, got)
		}
	}
}

func TestE2ELabelEditDeleteHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "label", "edit", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg label edit --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg label edit <name> [flags]", "--name <text>", "--color <hex>", "--description <text>", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("label edit help missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, code = runGGStreams(t, bin, t.TempDir(), "label", "delete", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg label delete --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{"gg label delete <name> [flags]", "--yes", "--repo", "--remote", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("label delete help missing %q:\n%s", want, stdout)
		}
	}
}
