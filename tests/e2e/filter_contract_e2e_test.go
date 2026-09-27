package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2EFilterRepoLiteralReplacementsAndUnusualPaths(t *testing.T) {
	names := []string{"plain.txt", "with space.txt", "café.txt", "한글.txt"}
	if runtime.GOOS != "windows" {
		names = append(names, "tab\there.txt", "q\"uote.txt", "back\\slash.txt", "ctrl\x01.txt")
	}
	files := map[string]string{"docs2/keep.txt": "unaffected\n"}
	for _, name := range names {
		files["docs/"+name] = "PRICE=10 name=abc\n"
	}
	dir := filterRepoTestRepo(t, files)
	args := []string{"repo", "filter-repo", "--path", "./docs/", "--path-rename", "docs:manual", "--replace-text", `PRICE=[0-9]+==>US$99`, "--replace-text", `name=\w+==>name=${x}`, "--force"}
	if err := runFilterRepoIn(t, dir, args); err != nil {
		t.Fatal(err)
	}
	for _, revision := range []string{"HEAD", "HEAD^"} {
		for _, name := range names {
			if got := gitIn(t, dir, "show", revision+":manual/"+name); got != "US$99 name=${x}" {
				t.Fatalf("%s/%q: %q", revision, name, got)
			}
		}
		if got := gitIn(t, dir, "ls-tree", "-r", "--name-only", revision); strings.Contains(got, "docs2/") {
			t.Fatalf("path prefix matched docs2: %s", got)
		}
	}
	if got := gitIn(t, dir, "status", "--porcelain"); got != "" {
		t.Fatalf("dirty rewritten tree: %s", got)
	}
}

func TestE2EFilterRepoMailmapFormats(t *testing.T) {
	for _, tc := range []struct{ name, mapping, want string }{
		{"name and email", "New Name <new@example.com> <old@example.com>\n", "New Name <new@example.com>"},
		{"email only", "<new@example.com> <old@example.com>\n", "Old Name <new@example.com>"},
		{"name only", "New Name <old@example.com>\n", "New Name <old@example.com>"},
		{"specific identity wins", "General <general@example.com> <old@example.com>\nSpecific <specific@example.com> Old Name <old@example.com>\n", "Specific <specific@example.com>"},
		{"BOM and comments", "\xef\xbb\xbf# comment\r\n\r\nNew Name <new@example.com> <old@example.com>\r\n", "New Name <new@example.com>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filterRepoTestRepo(t, map[string]string{"a.txt": "first\n"})
			mapping := filepath.Join(t.TempDir(), "mailmap")
			if err := os.WriteFile(mapping, []byte(tc.mapping), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--mailmap", mapping, "--force"}); err != nil {
				t.Fatal(err)
			}
			for _, format := range []string{"--format=%an <%ae>", "--format=%cn <%ce>"} {
				if got := gitIn(t, dir, "log", format); got != tc.want+"\n"+tc.want {
					t.Fatalf("identities: %q; want %q twice", got, tc.want)
				}
			}
		})
	}
}

func TestE2EFilterRepoPreEpochIdentity(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "first\n"})
	tree := gitIn(t, dir, "rev-parse", "HEAD^{tree}")
	commit := "tree " + tree + "\nauthor Old Name <old@example.com> -86400 +0000\ncommitter Old Name <old@example.com> -86400 +0000\n\npre-epoch\n"
	cmd := exec.Command("git", "hash-object", "--literally", "-t", "commit", "-w", "--stdin")
	cmd.Dir, cmd.Stdin = dir, strings.NewReader(commit)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commit fixture: %v: %s", err, out)
	}
	gitIn(t, dir, "update-ref", "HEAD", strings.TrimSpace(string(out)))
	mapping := filepath.Join(t.TempDir(), "mailmap")
	if err := os.WriteFile(mapping, []byte("New Name <new@example.com> <old@example.com>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, []string{"repo", "filter-repo", "--mailmap", mapping, "--force"}); err != nil {
		t.Fatal(err)
	}
	got := gitIn(t, dir, "cat-file", "commit", "HEAD")
	for _, role := range []string{"author", "committer"} {
		if !strings.Contains(got, role+" New Name <new@example.com> -86400 +0000") {
			t.Fatalf("identity/timestamp lost: %s", got)
		}
	}
}

func TestE2EFilterRepoInvalidFiltersLeaveHistoryUntouched(t *testing.T) {
	bin := buildGG(t)
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "original\n"})
	before := gitIn(t, dir, "show-ref")
	cases := [][]string{
		{}, {"--force"}, {"--invert-paths", "--path-rename", "a:b"},
		{"--path-rename", "abc"}, {"--path"}, {"--path", "a", "--wat"},
		{"--path", "a", "--repo", "https://github.com/o/r"},
		{"--path", "a", "--remote", "origin"}, {"--path", "a", "--explain"},
		{"--replace-text", "nodelim", "--force"}, {"--replace-text", "==>x", "--force"},
		{"--replace-text", "([==>x", "--force"}, {"--mailmap", filepath.Join(t.TempDir(), "missing"), "--force"},
	}
	for _, path := range []string{"", "  ", ".", "..", "../a", "a/../../b", "a//b"} {
		cases = append(cases, []string{"--path", path, "--force"})
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, code := runGGStreams(t, bin, dir, append([]string{"repo", "filter-repo"}, args...)...)
			if code != 2 || stdout != "" || !strings.Contains(stderr, "gg:") {
				t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
			}
			if got := gitIn(t, dir, "show-ref"); got != before {
				t.Fatalf("refs changed: %s", got)
			}
			if _, err := os.Stat(filepath.Join(dir, ".git", "filter-repo-backup")); !os.IsNotExist(err) {
				t.Fatalf("invalid filter created backup: %v", err)
			}
		})
	}
}

func TestE2EFilterRepoPipelineFailuresPreserveHistory(t *testing.T) {
	bin := buildGG(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, stream, want     string
		exportExit, importExit int
	}{
		{"truncated blob", "blob\nmark :1\ndata 999\nshort", "EOF", 0, 0},
		{"huge truncated blob", "blob\nmark :1\ndata 3000000000\nshort", "EOF", 0, 0},
		{"invalid blob header", "blob\nmark :1\ninvalid\n", "bad fast-export blob", 0, 0},
		{"export failure", "", "git fast-export failed", 17, 0},
		{"import failure", "", "git fast-import failed", 0, 19},
	} {
		for _, mode := range []string{"--force", "--dry-run"} {
			if mode == "--dry-run" && tc.importExit != 0 {
				continue
			}
			t.Run(tc.name+mode, func(t *testing.T) {
				dir := filterRepoTestRepo(t, map[string]string{"a.txt": "original\n"})
				before := gitIn(t, dir, "show-ref")
				fakeDir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
				rules := []fakeCLIRule{{Args: []string{"fast-export"}, Prefix: true, Stdout: tc.stream, ExitCode: tc.exportExit}}
				if tc.importExit != 0 {
					rules = append(rules, fakeCLIRule{Args: []string{"fast-import"}, Prefix: true, ExitCode: tc.importExit})
				}
				writeFakeCLI(t, fakeDir, "git", fakeCLIConfig{LogFile: log, Forward: realGit, Rules: rules})
				out, code := runGG(t, bin, fakeDir, dir, "repo", "filter-repo", "--replace-text", "original==>changed", mode)
				if code != 1 || !strings.Contains(out, tc.want) {
					t.Fatalf("exit %d: %s; want %s", code, out, tc.want)
				}
				if got := gitIn(t, dir, "show-ref"); got != before {
					t.Fatalf("partial rewrite changed refs: %s", got)
				}
				if calls := readLog(t, log); strings.Contains(calls, `"gc"`) || strings.Contains(calls, `"reset"`) {
					t.Fatalf("failed pipeline continued cleanup: %s", calls)
				}
				if mode == "--force" {
					if got := gitIn(t, filepath.Join(dir, ".git", "filter-repo-backup"), "show", "HEAD:a.txt"); got != "second" {
						t.Fatalf("backup lost original data: %q", got)
					}
				}
			})
		}
	}
}
