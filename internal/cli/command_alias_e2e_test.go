package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// runCommandAlias는 저장소 밖에서 fake CLI만 있는 PATH로 gg를 실행한다.
// git도 같은 로그에 기록하므로 의도하지 않은 저장소 조회를 검출한다.
func runCommandAlias(t *testing.T, bin string, args ...string) (stdout, stderr string, code int, calls string) {
	t.Helper()
	fakeDir := t.TempDir()
	logFile := filepath.Join(fakeDir, "calls.log")
	for _, name := range []string{"gh", "glab", "tea", "git"} {
		config := fakeCLIConfig{LogFile: logFile, TeaLogin: name == "tea"}
		if name == "git" {
			config.ExitCode = 99
		}
		writeFakeCLI(t, fakeDir, name, config)
	}
	cmd := ggCommandWithHomeAndPath(bin, t.TempDir(), t.TempDir(), fakeDir, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	return out.String(), errOut.String(),
		processExitCode(t, err, "stdout: "+out.String()+"\nstderr: "+errOut.String()), readLog(t, logFile)
}

func TestE2ECommandAliasesRouteByRepository(t *testing.T) {
	bin := buildGG(t)
	const title = `hello "world" & | ; %PATH% $HOME`
	const body = "line one\nline two\t끝\\"
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "gh run rerun on gitlab",
			args: []string{"--repo", "https://gitlab.com/grp/sub/r", "run", "rerun", "42"},
			want: wantCall("glab", "ci", "retry", "42", "--repo", "https://gitlab.com/grp/sub/r"),
		},
		{
			name: "glab ci get on github",
			args: []string{"ci", "get", "42", "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "run", "view", "42", "-R", "github.com/o/r"),
		},
		{
			name: "glab ci trace on github",
			args: []string{"ci", "trace", "42", "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "run", "watch", "42", "-R", "github.com/o/r"),
		},
		{
			name: "glab issue note on github preserves body",
			args: []string{"issue", "note", "18", "--body", body, "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "issue", "comment", "18", "--body", body, "-R", "github.com/o/r"),
		},
		{
			name: "glab issue update on github preserves title and body",
			args: []string{"issue", "update", "18", "--title", title, "--body", body, "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "issue", "edit", "18", "-R", "github.com/o/r", "--title", title, "--body", body),
		},
		{
			name: "glab mr note on github preserves body",
			args: []string{"mr", "note", "18", "--body", body, "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "pr", "comment", "18", "--body", body, "-R", "github.com/o/r"),
		},
		{
			name: "glab mr update on github preserves title and body",
			args: []string{"mr", "update", "18", "--title", title, "--body", body, "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "pr", "edit", "18", "-R", "github.com/o/r", "--title", title, "--body", body),
		},
		{
			name: "glab mr update on gitea",
			args: []string{"mr", "update", "18", "--title", title, "--body", body, "--repo", "https://gitea.com/o/r"},
			want: wantTeaCall("pulls", "edit", "18", "--login", "pub", "--repo", "o/r", "--title", title, "--description", body),
		},
		{
			name: "pipeline cancel pipeline on github consumes both action words",
			args: []string{"pipeline", "cancel", "pipeline", "42", "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "run", "cancel", "42", "-R", "github.com/o/r"),
		},
		{
			name: "ci cancel pipeline on gitlab does not duplicate pipeline",
			args: []string{"ci", "cancel", "pipeline", "42", "--repo", "https://gitlab.com/o/r"},
			want: wantCall("glab", "ci", "cancel", "pipeline", "42", "--repo", "https://gitlab.com/o/r"),
		},
		{
			name: "actions list on gitlab translates flags",
			args: []string{"actions", "list", "--branch", "feature/alias", "--limit", "5", "--repo", "https://gitlab.com/o/r"},
			want: wantCall("glab", "ci", "list", "--repo", "https://gitlab.com/o/r", "--ref", "feature/alias", "--per-page", "5"),
		},
		{
			name: "pipe get on github",
			args: []string{"pipe", "get", "42", "--repo", "https://github.com/o/r"},
			want: wantCall("gh", "run", "view", "42", "-R", "github.com/o/r"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code, calls := runCommandAlias(t, bin, tc.args...)
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("gg %q: exit %d, stdout %q, stderr %q", tc.args, code, stdout, stderr)
			}
			if calls != tc.want {
				t.Errorf("gg %q argv = %s, want %s", tc.args, calls, tc.want)
			}
		})
	}
}

func TestE2ECommandAliasesTeaPluralResourcesAcrossProviders(t *testing.T) {
	bin := buildGG(t)
	const label = `needs "review" & triage`
	cases := []struct {
		args []string
		gh   string
		glab string
	}{
		{
			args: []string{"repos", "view"},
			gh:   wantCall("gh", "repo", "view", "https://github.com/o/r"),
			glab: wantCall("glab", "repo", "view", "https://gitlab.com/o/r"),
		},
		{
			args: []string{"issues", "view", "18"},
			gh:   wantCall("gh", "issue", "view", "18", "-R", "github.com/o/r"),
			glab: wantCall("glab", "issue", "view", "18", "--repo", "https://gitlab.com/o/r"),
		},
		{
			args: []string{"labels", "create", "--name", label, "--color", "00ff00"},
			gh:   wantCall("gh", "label", "create", label, "-R", "github.com/o/r", "--color", "00ff00"),
			glab: wantCall("glab", "label", "create", "--repo", "https://gitlab.com/o/r", "--name", label, "--color", "00ff00"),
		},
		{
			args: []string{"releases", "view", "v1.2.3"},
			gh:   wantCall("gh", "release", "view", "v1.2.3", "-R", "github.com/o/r"),
			glab: wantCall("glab", "release", "view", "v1.2.3", "--repo", "https://gitlab.com/o/r"),
		},
		{
			args: []string{"pulls", "view", "18"},
			gh:   wantCall("gh", "pr", "view", "18", "-R", "github.com/o/r"),
			glab: wantCall("glab", "mr", "view", "18", "--repo", "https://gitlab.com/o/r"),
		},
	}
	for _, tc := range cases {
		for _, target := range []struct {
			host string
			want string
		}{
			{host: "github.com", want: tc.gh},
			{host: "gitlab.com", want: tc.glab},
		} {
			t.Run(tc.args[0]+"/"+target.host, func(t *testing.T) {
				args := append([]string{"--repo", "https://" + target.host + "/o/r"}, tc.args...)
				stdout, stderr, code, calls := runCommandAlias(t, bin, args...)
				if code != 0 || stdout != "" || stderr != "" {
					t.Errorf("gg %q: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
				}
				if calls != target.want {
					t.Errorf("gg %q argv = %s, want %s", args, calls, target.want)
				}
			})
		}
	}
}

func TestE2ECommandAliasesHelpMatchesCanonical(t *testing.T) {
	bin := buildGG(t)
	cases := []struct {
		alias     string
		canonical string
	}{
		{"repos", "repo"},
		{"issues", "issue"},
		{"labels", "label"},
		{"releases", "release"},
		{"pulls", "pr"},
		{"mr", "pr"},
		{"run", "ci"},
		{"pipeline", "ci"},
		{"issue note", "issue comment"},
		{"issue update", "issue edit"},
		{"mr note", "pr comment"},
		{"mr update", "pr edit"},
		{"ci get", "ci view"},
		{"ci trace", "ci watch"},
		{"run rerun", "ci retry"},
		{"pipeline cancel pipeline", "ci cancel"},
	}
	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			canonicalArgs := append(strings.Fields(tc.canonical), "--help")
			want, stderr, code, calls := runCommandAlias(t, bin, canonicalArgs...)
			if code != 0 || stderr != "" || !strings.Contains(want, "Usage:\n  gg "+tc.canonical+" ") {
				t.Fatalf("canonical help: exit %d, stdout %q, stderr %q", code, want, stderr)
			}
			if calls != "" {
				t.Errorf("canonical help should not run external CLIs, got %s", calls)
			}
			forms := [][]string{append(strings.Fields(tc.alias), "--help")}
			if !strings.Contains(tc.alias, " ") {
				forms = append(forms, []string{"help", tc.alias})
			}
			for _, args := range forms {
				stdout, stderr, code, calls := runCommandAlias(t, bin, args...)
				if code != 0 || stderr != "" || stdout != want {
					t.Errorf("gg %q: exit %d, stderr %q, help %q; want exit 0, empty stderr, help %q", args, code, stderr, stdout, want)
				}
				if calls != "" {
					t.Errorf("gg %q should not run external CLIs, got %s", args, calls)
				}
			}
		})
	}
}

func TestE2ECommandAliasesKeepOptionAndPositionalContracts(t *testing.T) {
	bin := buildGG(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"run", "rerun"}, "usage: gg ci retry <id>"},
		{[]string{"ci", "trace", "42", "43"}, "usage: gg ci watch [<id>]"},
		{[]string{"pipeline", "cancel", "pipeline"}, "usage: gg ci cancel <id>"},
		{[]string{"issue", "note", "18", "--message", "body"}, "unknown flag --message"},
		{[]string{"mr", "update", "18", "--description", "body"}, "unknown flag --description"},
		{[]string{"ci", "get", "--pipeline-id", "42"}, "unknown flag --pipeline-id"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			args := append([]string{"--repo", "https://github.com/o/r"}, tc.args...)
			stdout, stderr, code, calls := runCommandAlias(t, bin, args...)
			if code != 2 || stdout != "" || !strings.Contains(stderr, tc.want) {
				t.Errorf("gg %q: exit %d, stdout %q, stderr %q; want exit 2 with %q", args, code, stdout, stderr, tc.want)
			}
			if calls != "" {
				t.Errorf("invalid command should not run external CLIs, got %s", calls)
			}
		})
	}
}

func TestE2ECommandAliasesTeaCIUnsupportedWithoutExternalCalls(t *testing.T) {
	bin := buildGG(t)
	for _, command := range [][]string{
		{"actions", "list"},
		{"run", "rerun", "42"},
		{"pipe", "get", "42"},
		{"pipeline", "trace", "42"},
		{"ci", "cancel", "pipeline", "42"},
	} {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			args := append([]string{"--repo", "https://gitea.com/o/r"}, command...)
			stdout, stderr, code, calls := runCommandAlias(t, bin, args...)
			if code != 2 || stdout != "" || !strings.Contains(stderr, "ci is not supported for tea") {
				t.Errorf("gg %q: exit %d, stdout %q, stderr %q; want exit 2 with tea CI unsupported error", args, code, stdout, stderr)
			}
			if calls != "" {
				t.Errorf("unsupported CI must not invoke any CLI, including tea login lookup, got %s", calls)
			}
		})
	}
}
