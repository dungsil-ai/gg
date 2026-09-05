package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ELabelArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		fakeName string
		remote   string
		args     []string
		want     string
	}{
		{
			name:     "gh label list without limit",
			fakeName: "gh",
			remote:   "https://github.com/o/r.git",
			args:     []string{"label", "list"},
			want:     "gh label list -R github.com/o/r",
		},
		{
			name:     "gh label list with limit",
			fakeName: "gh",
			remote:   "https://github.com/o/r.git",
			args:     []string{"label", "list", "--limit", "5"},
			want:     "gh label list -R github.com/o/r --limit 5",
		},
		{
			name:     "gh label create full",
			fakeName: "gh",
			remote:   "https://github.com/o/r.git",
			args:     []string{"label", "create", "--name", "bug", "--color", "ff0000", "--description", "버그"},
			want:     "gh label create bug -R github.com/o/r --color ff0000 --description 버그",
		},
		{
			name:     "glab label list without limit",
			fakeName: "glab",
			remote:   "https://gitlab.com/o/r.git",
			args:     []string{"label", "list"},
			want:     "glab label list --repo https://gitlab.com/o/r",
		},
		{
			name:     "glab label list with limit",
			fakeName: "glab",
			remote:   "https://gitlab.com/o/r.git",
			args:     []string{"label", "list", "--limit", "3"},
			want:     "glab label list --repo https://gitlab.com/o/r --per-page 3",
		},
		{
			name:     "glab label create minimal",
			fakeName: "glab",
			remote:   "https://gitlab.com/o/r.git",
			args:     []string{"label", "create", "--name", "bug"},
			want:     "glab label create --repo https://gitlab.com/o/r --name bug",
		},
		{
			name:     "glab label create full",
			fakeName: "glab",
			remote:   "https://gitlab.com/o/r.git",
			args:     []string{"label", "create", "--name", "bug", "--color", "#FF0000", "--description", "broken"},
			want:     "glab label create --repo https://gitlab.com/o/r --name bug --color #FF0000 --description broken",
		},
		{
			name:     "glab label create repo flag",
			fakeName: "glab",
			remote:   "https://gitlab.com/o/r.git",
			args:     []string{"--repo", "https://gitlab.com/custom/repo", "label", "create", "--name", "bug"},
			want:     "glab label create --repo https://gitlab.com/custom/repo --name bug",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeBin(t, fakeDir, tc.fakeName, logFile)
			repo := tempRepo(t, tc.remote)

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

func TestE2ELabelCreateUsageErrors(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "name 없음",
			args: []string{"label", "create"},
			want: "usage: gg label create --name <text>",
		},
		{
			name: "공백 name",
			args: []string{"label", "create", "--name", "  "},
			want: "usage: gg label create --name <text>",
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

func TestE2ELabelExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "label", "list"},
		{"label", "list", "--explain"},
		{"label", "create", "--name", "bug", "--explain"},
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
