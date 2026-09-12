package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseRequestFilterRepo(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "경로 삭제", args: []string{"repo", "filter-repo", "--path", "secret.txt", "--invert-paths", "--force"},
			want: Request{Resource: "repo", Action: "filter-repo", FilterPaths: []string{"secret.txt"}, FilterInvert: true, FilterForce: true}},
		{name: "경로 반복", args: []string{"repo", "filter-repo", "--path", "a", "--path", "b/c", "--force"},
			want: Request{Resource: "repo", Action: "filter-repo", FilterPaths: []string{"a", "b/c"}, FilterForce: true}},
		{name: "이름 변경", args: []string{"repo", "filter-repo", "--path-rename", "docs:manual", "--force"},
			want: Request{Resource: "repo", Action: "filter-repo", FilterRenames: []string{"docs:manual"}, FilterForce: true}},
		{name: "내용 치환과 미리보기", args: []string{"repo", "filter-repo", "--replace-text", "secret==>***", "--dry-run"},
			want: Request{Resource: "repo", Action: "filter-repo", FilterReplaces: []string{"secret==>***"}, FilterDryRun: true}},
		{name: "mailmap", args: []string{"repo", "filter-repo", "--mailmap", "map.txt", "--force"},
			want: Request{Resource: "repo", Action: "filter-repo", FilterMailmap: "map.txt", FilterForce: true}},
	}
	for _, c := range cases {
		got, err := ParseRequest(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestParseRequestFilterRepoErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "필터 없음", args: []string{"repo", "filter-repo", "--force"},
			want: "needs at least one of --path, --path-rename, --replace-text, or --mailmap"},
		{name: "필터 없이 단독", args: []string{"repo", "filter-repo"},
			want: "needs at least one of --path, --path-rename, --replace-text, or --mailmap"},
		{name: "invert 단독", args: []string{"repo", "filter-repo", "--invert-paths", "--path-rename", "a:b", "--force"},
			want: "--invert-paths needs --path"},
		{name: "rename 형식", args: []string{"repo", "filter-repo", "--path-rename", "abc"},
			want: "--path-rename must look like <old:new>"},
		{name: "값 없음", args: []string{"repo", "filter-repo", "--path"},
			want: "--path needs a value"},
		{name: "positional", args: []string{"repo", "filter-repo", "extra", "--path", "a"},
			want: "usage: gg repo filter-repo [flags]"},
		{name: "모르는 flag", args: []string{"repo", "filter-repo", "--path", "a", "--wat"},
			want: "unknown flag --wat"},
		{name: "앞의 repo 문맥", args: []string{"--repo", "https://github.com/o/r", "repo", "filter-repo", "--path", "a"},
			want: "--repo is not supported for repo filter-repo"},
		{name: "뒤의 repo 문맥", args: []string{"repo", "filter-repo", "--path", "a", "--repo", "https://github.com/o/r"},
			want: "--repo is not supported for repo filter-repo"},
		{name: "remote 미지원", args: []string{"repo", "filter-repo", "--path", "a", "--remote", "origin"},
			want: "--remote is not supported for repo filter-repo"},
		{name: "explain 미지원", args: []string{"repo", "filter-repo", "--path", "a", "--explain"},
			want: "--explain is not supported for repo filter-repo"},
	}
	for _, c := range cases {
		_, err := ParseRequest(c.args)
		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("%s: UsageError 기대, got %v", c.name, err)
			continue
		}
		if !strings.Contains(usage.Msg, c.want) {
			t.Errorf("%s 오류 = %q, want %q 포함", c.name, usage.Msg, c.want)
		}
	}
}

func TestNormalizeFilterRepoPath(t *testing.T) {
	for _, p := range []string{"a.txt", "docs", "docs/", "./docs", "a/b/c"} {
		if _, err := normalizeFilterRepoPath(p); err != nil {
			t.Errorf("normalize(%q): %v", p, err)
		}
	}
	if got, _ := normalizeFilterRepoPath("docs/"); got != "docs" {
		t.Errorf("trailing slash = %q, want docs", got)
	}
	for _, p := range []string{"", "  ", ".", "..", "../a", "a/../../b", "a//b"} {
		if _, err := normalizeFilterRepoPath(p); err == nil {
			t.Errorf("normalize(%q): 오류 기대", p)
		}
	}
}

func TestKeepAndRenameFilterRepoPath(t *testing.T) {
	keepOnly := &resolvedFilters{paths: []string{"docs", "keep.txt"}}
	for _, f := range []string{"docs", "docs/a.txt", "keep.txt"} {
		if !keepFilterRepoPath(keepOnly, f) {
			t.Errorf("keep-only %q: 유지 기대", f)
		}
	}
	for _, f := range []string{"other.txt", "docs2", "docs2/a"} {
		if keepFilterRepoPath(keepOnly, f) {
			t.Errorf("keep-only %q: 제거 기대", f)
		}
	}
	remove := &resolvedFilters{paths: []string{"secret.txt", "keys"}, invert: true}
	if keepFilterRepoPath(remove, "secret.txt") || keepFilterRepoPath(remove, "keys/a") {
		t.Error("invert: secret/keys 제거 기대")
	}
	if !keepFilterRepoPath(remove, "a.txt") {
		t.Error("invert: a.txt 유지 기대")
	}
	ren := &resolvedFilters{renames: []pathRename{{old: "docs", new: "manual"}, {old: "a.txt", new: "b.txt"}}}
	if got, ok := renameFilterRepoPath(ren, "docs/note.txt"); !ok || got != "manual/note.txt" {
		t.Errorf("rename dir = %q, %v", got, ok)
	}
	if got, ok := renameFilterRepoPath(ren, "a.txt"); !ok || got != "b.txt" {
		t.Errorf("rename file = %q, %v", got, ok)
	}
	if _, ok := renameFilterRepoPath(ren, "other.txt"); ok {
		t.Error("rename other.txt: 변경 없음 기대")
	}
}

func TestCompileReplaceRule(t *testing.T) {
	r, err := compileReplaceRule(`password=\S+==>password=***`)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.re.ReplaceAllString("a password=hunter2 b", r.replacement); got != "a password=*** b" {
		t.Errorf("replace = %q", got)
	}
	for _, bad := range []string{"nodelim", "==>x", ""} {
		if _, err := compileReplaceRule(bad); err == nil {
			t.Errorf("compile(%q): 오류 기대", bad)
		}
	}
	if _, err := compileReplaceRule("([==>x"); err == nil {
		t.Error("bad regex: 오류 기대")
	}
}

func TestE2EFilterRepoReplaceTextFromFile(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{
		"a.txt":    "secret password=hunter2\npreserve me\n",
		"note.txt": "public secret password=abc123\n",
	})
	f := filepath.Join(t.TempDir(), "replacements.txt")
	content := "# comment\n\nsecret==>***\npassword=\\S+==>password=?\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, Request{
		Resource: "repo", Action: "filter-repo",
		FilterReplaces: []string{f}, FilterForce: true,
	}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		object string
		want   string
	}{
		{"HEAD^:a.txt", "*** password=?\npreserve me"},
		{"HEAD^:note.txt", "public *** password=?"},
		{"HEAD:a.txt", "second"},
		{"HEAD:note.txt", "public *** password=?"},
	} {
		if got := gitIn(t, dir, "show", c.object); got != c.want {
			t.Errorf("%s = %q, want %q", c.object, got, c.want)
		}
	}
	for name, want := range map[string]string{
		"a.txt":    "second\n",
		"note.txt": "public *** password=?\n",
	} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("working tree %s = %q, want %q", name, got, want)
		}
	}
}

func TestParseMailmapFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "mailmap")
	content := "New Name <new@example.com> <old@example.com>\n" +
		"Other <o@n.com> Old <o@o.com>\n" +
		"<keep@example.com> <keep@example.com>\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	mm, err := parseMailmapFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if nn, ee, changed := applyMailmap(mm, "Anything", "old@example.com"); !changed || nn != "New Name" || ee != "new@example.com" {
		t.Errorf("email map = %q <%q> changed=%v", nn, ee, changed)
	}
	if nn, ee, changed := applyMailmap(mm, "Old", "o@o.com"); !changed || nn != "Other" || ee != "o@n.com" {
		t.Errorf("name+email map = %q <%q> changed=%v", nn, ee, changed)
	}
	if _, _, changed := applyMailmap(mm, "Nobody", "nobody@example.com"); changed {
		t.Error("unknown: 변경 없음 기대")
	}
	if _, err := parseMailmapFile(filepath.Join(dir, "missing")); err == nil {
		t.Error("missing file: 오류 기대")
	}
}

// TestFilterFastExportStream은 git 없이 합성 스트림으로 필터를 본다.
func TestFilterFastExportStream(t *testing.T) {
	in := "blob\n" +
		"mark :1\n" +
		"data 16\n" +
		"hi hunter2 keep\n" +
		"commit refs/heads/main\n" +
		"mark :2\n" +
		"author Old <old@example.com> 1700000000 +0900\n" +
		"committer Old <old@example.com> 1700000000 +0900\n" +
		"data 4\n" +
		"msg\n" +
		"deleteall\n" +
		"M 100644 :1 docs/note.txt\n" +
		"M 100644 :1 secret.txt\n"
	f := &resolvedFilters{
		paths:   []string{"secret.txt"},
		invert:  true,
		renames: []pathRename{{old: "docs", new: "manual"}},
		mail: &mailmap{
			byEmail:     map[string]mailmapEntry{"old@example.com": {name: "New", email: "new@example.com"}},
			byNameEmail: map[string]mailmapEntry{},
		},
	}
	rule, err := compileReplaceRule(`hunter2==>***`)
	if err != nil {
		t.Fatal(err)
	}
	f.replaces = []replaceRule{rule}
	var out bytes.Buffer
	stats := &filterStats{}
	if err := filterFastExportStream(strings.NewReader(in), &out, f, stats, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"data 12\nhi *** keep\n",
		"author New <new@example.com>",
		"committer New <new@example.com>",
		"M 100644 :1 manual/note.txt\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("출력에 %q 없음:\n%s", want, got)
		}
	}
	if strings.Contains(got, "secret.txt") || strings.Contains(got, "hunter2") || strings.Contains(got, "old@example.com") {
		t.Errorf("필터 잔재 있음:\n%s", got)
	}
	if stats.blobsChanged != 1 || stats.identitiesDone != 2 || stats.pathsDropped != 1 || stats.pathsRenamed != 1 {
		t.Errorf("stats = %+v", stats)
	}
}

// filterRepoTestRepo는 실제 git 저장소를 만들고 두 커밋을 쌓는다.
func filterRepoTestRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "config", "user.name", "Old Name")
	gitIn(t, dir, "config", "user.email", "old@example.com")
	// 테스트 환경의 전역 커밋 서명(gpg/ssh)이 끼지 않도록 끈다.
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "first")
	second := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(second, []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "second")
	return dir
}

func runFilterRepoIn(t *testing.T, dir string, req Request) error {
	t.Helper()
	_, err := runFilterRepoInOutput(t, dir, req)
	return err
}

func runFilterRepoInOutput(t *testing.T, dir string, req Request) (string, error) {
	t.Helper()
	origStdout, origStderr := osStdout, osStderr
	defer func() { osStdout, osStderr = origStdout, origStderr }()
	var stdout, stderr bytes.Buffer
	osStdout, osStderr = &stdout, &stderr
	t.Chdir(dir)
	err := runFilterRepo(req)
	return stdout.String(), err
}

func TestE2EFilterRepoRemovesPath(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "secret.txt": "topsecret\n"})
	if err := runFilterRepoIn(t, dir, Request{
		Resource: "repo", Action: "filter-repo",
		FilterPaths: []string{"secret.txt"}, FilterInvert: true, FilterForce: true,
	}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "log", "--oneline", "--all"); len(strings.Split(strings.TrimSpace(got), "\n")) != 2 {
		t.Errorf("커밋 2개 유지 기대, got:\n%s", got)
	}
	if got := gitIn(t, dir, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(got, "secret.txt") {
		t.Errorf("secret.txt 제거 기대, got %q", got)
	}
	if got := gitIn(t, dir, "log", "--all", "--oneline", "--", "secret.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("히스토리에서 secret.txt 제거 기대, got %q", got)
	}
	if out := gitIn(t, dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("작업 트리 clean 기대, got %q", out)
	}
}

func TestE2EFilterRepoKeepRenameReplaceMailmap(t *testing.T) {
	dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n", "docs/note.txt": "pw=hunter2\n"})
	mailmap := filepath.Join(t.TempDir(), "mailmap")
	if err := os.WriteFile(mailmap, []byte("New Name <new@example.com> <old@example.com>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFilterRepoIn(t, dir, Request{
		Resource: "repo", Action: "filter-repo",
		FilterPaths: []string{"docs"}, FilterRenames: []string{"docs:manual"},
		FilterReplaces: []string{`hunter2==>***`}, FilterMailmap: mailmap, FilterForce: true,
	}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dir, "ls-tree", "-r", "--name-only", "HEAD"); got != "manual/note.txt" {
		t.Errorf("HEAD files = %q, want manual/note.txt", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "manual", "note.txt"))
	if err != nil || !strings.Contains(string(data), "***") || strings.Contains(string(data), "hunter2") {
		t.Errorf("replace 기대, got %q, err %v", data, err)
	}
	if got := gitIn(t, dir, "log", "--format=%an <%ae>", "--all"); strings.Contains(got, "old@example.com") {
		t.Errorf("mailmap 기대, got %q", got)
	}
}

func TestE2EFilterRepoDryRunChangesNothing(t *testing.T) {
	for _, c := range []struct {
		name        string
		path        string
		rename      string
		replacement string
		want        string
	}{
		{
			name: "변경 미리보기", path: "secret.txt", rename: "docs:manual", replacement: "hunter2==>***",
			want: "dry-run: 2 commits, 4 blobs (1 would change), 4 identities (0 would change), 2 paths dropped, 2 renamed\n",
		},
		{
			name: "일치하는 대상 없음", path: "missing.txt", rename: "missing:other", replacement: "absent==>***",
			want: "dry-run: 2 commits, 4 blobs (0 would change), 4 identities (0 would change), 0 paths dropped, 0 renamed\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := filterRepoTestRepo(t, map[string]string{
				"a.txt": "keep\n", "secret.txt": "topsecret\n", "docs/note.txt": "password=hunter2\n",
			})
			gitIn(t, dir, "branch", "saved", "HEAD^")
			gitIn(t, dir, "tag", "before-filter", "HEAD^")
			beforeHead := gitIn(t, dir, "rev-parse", "HEAD")
			beforeRefs := gitIn(t, dir, "show-ref")
			beforeTree := gitIn(t, dir, "ls-tree", "-r", "HEAD")
			beforeFiles := map[string][]byte{}
			for _, name := range []string{"a.txt", "secret.txt", "docs/note.txt"} {
				data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				beforeFiles[name] = data
			}

			out, err := runFilterRepoInOutput(t, dir, Request{
				Resource: "repo", Action: "filter-repo",
				FilterPaths: []string{c.path}, FilterInvert: true, FilterDryRun: true,
				FilterRenames: []string{c.rename}, FilterReplaces: []string{c.replacement},
			})
			if err != nil {
				t.Fatal(err)
			}
			if out != c.want {
				t.Errorf("dry-run preview = %q, want %q", out, c.want)
			}
			if got := gitIn(t, dir, "rev-parse", "HEAD"); got != beforeHead {
				t.Error("dry-run: HEAD 변경 없음 기대")
			}
			if got := gitIn(t, dir, "show-ref"); got != beforeRefs {
				t.Errorf("dry-run: refs 변경 없음 기대, got %q, want %q", got, beforeRefs)
			}
			if got := gitIn(t, dir, "ls-tree", "-r", "HEAD"); got != beforeTree {
				t.Errorf("dry-run: tree 변경 없음 기대, got %q, want %q", got, beforeTree)
			}
			for name, want := range beforeFiles {
				got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("dry-run: %s 변경 없음 기대, got %q, want %q", name, got, want)
				}
			}
			if got := gitIn(t, dir, "status", "--porcelain"); got != "" {
				t.Errorf("dry-run: 작업 트리 clean 기대, got %q", got)
			}
		})
	}
}

func TestE2EFilterRepoSafety(t *testing.T) {
	t.Run("force 없이 거부", func(t *testing.T) {
		dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n"})
		before := gitIn(t, dir, "rev-parse", "HEAD")
		err := runFilterRepoIn(t, dir, Request{
			Resource: "repo", Action: "filter-repo", FilterPaths: []string{"a.txt"}, FilterInvert: true,
		})
		if err == nil || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("--force 안내 기대, got %v", err)
		}
		if got := gitIn(t, dir, "rev-parse", "HEAD"); got != before {
			t.Error("거부 시 HEAD 변경 없음 기대")
		}
	})
	t.Run("dirty 작업 트리 거부", func(t *testing.T) {
		dir := filterRepoTestRepo(t, map[string]string{"a.txt": "keep\n"})
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dirty\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := runFilterRepoIn(t, dir, Request{
			Resource: "repo", Action: "filter-repo",
			FilterPaths: []string{"a.txt"}, FilterInvert: true, FilterForce: true,
		})
		if err == nil || !strings.Contains(err.Error(), "dirty") {
			t.Fatalf("dirty 안내 기대, got %v", err)
		}
	})
	t.Run("저장소 밖 거부", func(t *testing.T) {
		err := runFilterRepoIn(t, t.TempDir(), Request{
			Resource: "repo", Action: "filter-repo",
			FilterPaths: []string{"a.txt"}, FilterInvert: true, FilterForce: true,
		})
		if err == nil || !strings.Contains(err.Error(), "not a git repository") {
			t.Fatalf("저장소 안내 기대, got %v", err)
		}
	})
}

func TestE2EFilterRepoPreservesMergesAndTags(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "config", "user.name", "T")
	gitIn(t, dir, "config", "user.email", "t@e.com")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "base")
	gitIn(t, dir, "checkout", "-qb", "side")
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "side")
	gitIn(t, dir, "checkout", "-q", "-")
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "main")
	gitIn(t, dir, "merge", "-q", "--no-ff", "side", "-m", "merge side")
	gitIn(t, dir, "tag", "-a", "v1", "-m", "release one")

	if err := runFilterRepoIn(t, dir, Request{
		Resource: "repo", Action: "filter-repo",
		FilterPaths: []string{"b.txt"}, FilterInvert: true, FilterForce: true,
	}); err != nil {
		t.Fatal(err)
	}
	// 머지 토폴로지: 머지 커밋의 부모가 2개여야 한다.
	parents := gitIn(t, dir, "log", "-1", "--format=%P", "HEAD")
	if len(strings.Fields(parents)) != 2 {
		t.Errorf("merge parents = %q, want 2", parents)
	}
	if tags := gitIn(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "v1" {
		t.Errorf("tags = %q, want v1", tags)
	}
	if got := gitIn(t, dir, "log", "--all", "--oneline", "--", "b.txt"); strings.TrimSpace(got) != "" {
		t.Errorf("b.txt 제거 기대, got %q", got)
	}
}
