package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPRReviewArgv(t *testing.T) {
	bin := buildGG(t)

	cases := []struct {
		name     string
		remote   string
		fakeName string
		teaLogin bool
		args     []string
		want     string
	}{
		{
			name:     "github approve",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "review", "42", "--approve"},
			want:     "gh pr review 42 -R github.com/o/r --approve",
		},
		{
			name:     "github request changes",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "review", "42", "--request-changes", "--body", "고쳐주세요"},
			want:     "gh pr review 42 -R github.com/o/r --request-changes --body 고쳐주세요",
		},
		{
			name:     "github comment",
			remote:   "https://github.com/o/r.git",
			fakeName: "gh",
			args:     []string{"pr", "review", "42", "--comment", "--body", "검토중"},
			want:     "gh pr review 42 -R github.com/o/r --comment --body 검토중",
		},
		{
			name:     "github approve repo flag",
			remote:   "",
			fakeName: "gh",
			args:     []string{"--repo", "https://github.com/custom/repo", "pr", "review", "42", "--approve"},
			want:     "gh pr review 42 -R github.com/custom/repo --approve",
		},
		{
			name:     "glab approve",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "review", "42", "--approve"},
			want:     "glab mr approve 42 --repo https://gitlab.com/o/r",
		},
		{
			name:     "tea approve",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			teaLogin: true,
			args:     []string{"pr", "review", "42", "--approve"},
			want:     "tea pulls approve 42 --login pub --repo o/r",
		},
		{
			name:     "tea request changes",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			teaLogin: true,
			args:     []string{"pr", "review", "42", "--request-changes", "--body", "고쳐주세요"},
			want:     "tea pulls reject 42 고쳐주세요 --login pub --repo o/r",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			if tc.teaLogin {
				writeFakeTeaWithLogin(t, fakeDir, logFile)
			} else {
				writeFakeReadyBin(t, fakeDir, tc.fakeName, logFile, "", "", 0)
			}
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

func TestE2EPRReviewUnsupported(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		fakeName string
		args     []string
		want     string
	}{
		{
			name:     "glab request changes",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "review", "42", "--request-changes", "--body", "고쳐주세요"},
			want:     "pr review --request-changes/--comment is not supported for glab",
		},
		{
			name:     "glab comment",
			remote:   "https://gitlab.com/o/r.git",
			fakeName: "glab",
			args:     []string{"pr", "review", "42", "--comment", "--body", "검토중"},
			want:     "pr review --request-changes/--comment is not supported for glab",
		},
		{
			name:     "tea comment",
			remote:   "https://gitea.com/o/r.git",
			fakeName: "tea",
			args:     []string{"pr", "review", "42", "--comment", "--body", "검토중"},
			want:     "pr review --comment is not supported for tea",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeTeaWithLogin(t, fakeDir, logFile)
			writeFakeReadyBin(t, fakeDir, "glab", logFile, "", "", 0)
			repo := tempRepo(t, tc.remote)

			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, repo, tc.args...)
			if code != 2 {
				t.Errorf("gg %v: exit code = %d, want 2 (stdout: %s, stderr: %s)", tc.args, code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("gg %v: stderr = %q, want substring %q", tc.args, stderr, tc.want)
			}
			if got := readLog(t, logFile); got != "" {
				t.Errorf("fake provider should not be called, got: %q", got)
			}
		})
	}
}

func TestE2EPRReviewUsageErrors(t *testing.T) {
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
			name: "review without kind",
			args: []string{"pr", "review", "42"},
			want: "usage: gg pr review <number> (--approve | --request-changes | --comment)",
		},
		{
			name: "review with two kinds",
			args: []string{"pr", "review", "42", "--approve", "--comment", "--body", "x"},
			want: "usage: gg pr review <number> (--approve | --request-changes | --comment)",
		},
		{
			name: "request changes without body",
			args: []string{"pr", "review", "42", "--request-changes"},
			want: "pr review --request-changes needs --body <text>",
		},
		{
			name: "request changes with blank body",
			args: []string{"pr", "review", "42", "--request-changes", "--body", "  "},
			want: "pr review --request-changes needs --body <text>",
		},
		{
			name: "review missing number",
			args: []string{"pr", "review"},
			want: "usage: gg pr review <number>",
		},
		{
			name: "review unknown flag",
			args: []string{"pr", "review", "42", "--approve", "--invalid"},
			want: "unknown flag",
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

func TestE2EPRReviewExplain(t *testing.T) {
	bin, fakeDir, logFile := setupFakeGH(t)
	repo := tempRepo(t, "https://github.com/o/r.git")

	for _, args := range [][]string{
		{"--explain", "pr", "review", "42", "--approve"},
		{"pr", "review", "42", "--approve", "--explain"},
		{"pr", "review", "42", "--request-changes", "--body", "b", "--explain"},
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

func TestE2EPRReviewHelp(t *testing.T) {
	bin := buildGG(t)

	stdout, stderr, code := runGGStreams(t, bin, t.TempDir(), "pr", "review", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("gg pr review --help = stderr %q, exit %d", stderr, code)
	}
	for _, want := range []string{
		"gg pr review <number> (--approve | --request-changes | --comment) [flags]",
		"--approve", "--request-changes", "--comment", "--body <text>", "--repo", "--remote", "--explain",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("pr review help missing %q:\n%s", want, stdout)
		}
	}
}
