package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseRequestRelease(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "release list", args: []string{"release", "list"},
			want: Request{Resource: "release", Action: "list"}},
		{name: "release list limit", args: []string{"release", "list", "--limit", "5"},
			want: Request{Resource: "release", Action: "list", Limit: "5"}},
		{name: "release view 생략하면 latest", args: []string{"release", "view"},
			want: Request{Resource: "release", Action: "view"}},
		{name: "release view tag", args: []string{"release", "view", "v1.0.0"},
			want: Request{Resource: "release", Action: "view", Tag: "v1.0.0"}},
		{name: "release create flag", args: []string{"release", "create", "v1.0.0", "--title", "t", "--notes", "n", "--draft"},
			want: Request{Resource: "release", Action: "create", Tag: "v1.0.0", Title: "t", Notes: "n", Draft: true}},
		{name: "release create asset와 prerelease", args: []string{"release", "create", "v1.0.0", "a.zip", "b.zip", "--prerelease", "--ref", "main"},
			want: Request{Resource: "release", Action: "create", Tag: "v1.0.0", Files: []string{"a.zip", "b.zip"}, Prerelease: true, Ref: "main"}},
		{name: "release edit", args: []string{"release", "edit", "v1.0.0", "--notes", "n"},
			want: Request{Resource: "release", Action: "edit", Tag: "v1.0.0", Notes: "n"}},
		{name: "release delete", args: []string{"release", "delete", "v1.0.0", "--yes", "--cleanup-tag"},
			want: Request{Resource: "release", Action: "delete", Tag: "v1.0.0", Yes: true, CleanupTag: true}},
		{name: "release download 생략하면 latest", args: []string{"release", "download", "--pattern", "*.zip", "--dir", "dist"},
			want: Request{Resource: "release", Action: "download", Pattern: "*.zip", Dir: "dist"}},
		{name: "release download tag", args: []string{"release", "download", "v1.0.0"},
			want: Request{Resource: "release", Action: "download", Tag: "v1.0.0"}},
		{name: "release upload", args: []string{"release", "upload", "v1.0.0", "a.zip"},
			want: Request{Resource: "release", Action: "upload", Tag: "v1.0.0", Files: []string{"a.zip"}}},
		{name: "release delete-asset", args: []string{"release", "delete-asset", "v1.0.0", "a.zip", "--yes"},
			want: Request{Resource: "release", Action: "delete-asset", Tag: "v1.0.0", Asset: "a.zip", Yes: true}},
		{name: "release repo 문맥", args: []string{"release", "list", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "release", Action: "list", RepoFlag: "https://github.com/o/r"}},
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

	bad := []struct {
		args []string
		want string
	}{
		{args: []string{"release"}, want: "release needs an action: list, view, create, edit, delete, download, upload, delete-asset"},
		{args: []string{"release", "publish"}, want: "release does not support publish"},
		{args: []string{"release", "view", "a", "b"}, want: "usage: gg release view [<tag>]"},
		{args: []string{"release", "create"}, want: "usage: gg release create <tag> [asset...]"},
		{args: []string{"release", "edit"}, want: "usage: gg release edit <tag>"},
		{args: []string{"release", "edit", "a", "b"}, want: "usage: gg release edit <tag>"},
		{args: []string{"release", "delete"}, want: "usage: gg release delete <tag>"},
		{args: []string{"release", "download", "a", "b"}, want: "usage: gg release download [<tag>]"},
		{args: []string{"release", "upload", "v1.0.0"}, want: "usage: gg release upload <tag> <asset>..."},
		{args: []string{"release", "delete-asset", "v1.0.0"}, want: "usage: gg release delete-asset <tag> <asset>"},
		{args: []string{"release", "delete-asset", "v1.0.0", "a", "b"}, want: "usage: gg release delete-asset <tag> <asset>"},
		{args: []string{"release", "list", "--draft"}, want: "unknown flag --draft"},
		{args: []string{"release", "create", "v1.0.0", "--notes"}, want: "--notes needs a value"},
	}
	for _, c := range bad {
		_, err := ParseRequest(c.args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", c.args, err)
			continue
		}
		if ue.Msg != c.want {
			t.Errorf("ParseRequest(%v) = %q, want %q", c.args, ue.Msg, c.want)
		}
	}
}

func TestTranslateRelease(t *testing.T) {
	gh := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}

	cases := []struct {
		name string
		req  Request
		repo RepoURL
		p    Provider
		want Invocation
	}{
		{name: "gh release list",
			req:  Request{Resource: "release", Action: "list", Limit: "5"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "list", "-R", "github.com/o/r", "--limit", "5"}}},
		{name: "gh release view tag",
			req:  Request{Resource: "release", Action: "view", Tag: "v1.0.0"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "view", "v1.0.0", "-R", "github.com/o/r"}}},
		{name: "gh release view latest는 tag 없음",
			req:  Request{Resource: "release", Action: "view"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "view", "-R", "github.com/o/r"}}},
		{name: "gh release create 전체 flag",
			req:  Request{Resource: "release", Action: "create", Tag: "v1.0.0", Files: []string{"a.zip"}, Title: "t", Notes: "n", Ref: "main", Draft: true, Prerelease: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "create", "v1.0.0", "a.zip", "--title", "t", "--notes", "n", "--target", "main", "--draft", "--prerelease", "-R", "github.com/o/r"}}},
		{name: "gh release create 최소 인자",
			req:  Request{Resource: "release", Action: "create", Tag: "v1.0.0"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "create", "v1.0.0", "-R", "github.com/o/r"}}},
		{name: "gh release edit",
			req:  Request{Resource: "release", Action: "edit", Tag: "v1.0.0", Title: "t"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "edit", "v1.0.0", "-R", "github.com/o/r", "--title", "t"}}},
		{name: "gh release delete",
			req:  Request{Resource: "release", Action: "delete", Tag: "v1.0.0", Yes: true, CleanupTag: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "delete", "v1.0.0", "-R", "github.com/o/r", "--yes", "--cleanup-tag"}}},
		{name: "gh release download tag",
			req:  Request{Resource: "release", Action: "download", Tag: "v1.0.0"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "download", "v1.0.0", "-R", "github.com/o/r"}}},
		{name: "gh release download latest에 pattern과 dir",
			req:  Request{Resource: "release", Action: "download", Pattern: "*.zip", Dir: "dist"},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "download", "--pattern", "*.zip", "--dir", "dist", "-R", "github.com/o/r"}}},
		{name: "gh release upload",
			req:  Request{Resource: "release", Action: "upload", Tag: "v1.0.0", Files: []string{"a.zip", "b.zip"}},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "upload", "v1.0.0", "a.zip", "b.zip", "-R", "github.com/o/r"}}},
		{name: "gh release delete-asset",
			req:  Request{Resource: "release", Action: "delete-asset", Tag: "v1.0.0", Asset: "a.zip", Yes: true},
			repo: gh, p: GH,
			want: Invocation{Bin: "gh", Args: []string{"release", "delete-asset", "v1.0.0", "a.zip", "--yes", "-R", "github.com/o/r"}}},
		{name: "glab release list는 per-page",
			req:  Request{Resource: "release", Action: "list", Limit: "5"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "list", "--repo", "https://git.example.com/grp/sub/p", "--per-page", "5"}}},
		{name: "glab release view",
			req:  Request{Resource: "release", Action: "view", Tag: "v1.0.0"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "view", "v1.0.0", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab release create는 --name과 --ref",
			req:  Request{Resource: "release", Action: "create", Tag: "v1.0.0", Files: []string{"a.zip"}, Title: "t", Notes: "n", Ref: "main"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "create", "v1.0.0", "a.zip", "--name", "t", "--notes", "n", "--ref", "main", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab release delete는 --with-tag",
			req:  Request{Resource: "release", Action: "delete", Tag: "v1.0.0", Yes: true, CleanupTag: true},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "delete", "v1.0.0", "--repo", "https://git.example.com/grp/sub/p", "--yes", "--with-tag"}}},
		{name: "glab release download pattern은 --asset-name",
			req:  Request{Resource: "release", Action: "download", Pattern: "*.zip", Dir: "dist"},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "download", "--asset-name", "*.zip", "--dir", "dist", "--repo", "https://git.example.com/grp/sub/p"}}},
		{name: "glab release upload",
			req:  Request{Resource: "release", Action: "upload", Tag: "v1.0.0", Files: []string{"a.zip"}},
			repo: gl, p: GLab,
			want: Invocation{Bin: "glab", Args: []string{"release", "upload", "v1.0.0", "a.zip", "--repo", "https://git.example.com/grp/sub/p"}}},
	}
	for _, c := range cases {
		got, err := Translate(c.req, c.repo, c.p, "")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestTranslateReleaseUnsupported(t *testing.T) {
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}
	// glab release에는 edit와 delete-asset 하위 명령이 없다.
	for _, action := range []string{"edit", "delete-asset"} {
		_, err := Translate(Request{Resource: "release", Action: action, Tag: "v1.0.0"}, gl, GLab, "")
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Fatalf("Translate(release %s, glab): UsageError 기대, got %v", action, err)
		}
		if ue.Msg != "release does not support "+action {
			t.Errorf("glab 오류 = %q", ue.Msg)
		}
	}
	// tea release는 list/create/edit/delete만 중계한다. view/download/upload/
	// delete-asset은 builder가 없어 dispatch의 미지원 오류로 걸러진다.
	te := RepoURL{Host: "gitea.com", Owner: "o", Name: "r"}
	got, err := Translate(Request{Resource: "release", Action: "list", Limit: "5"}, te, Tea, "corp")
	if err != nil {
		t.Fatalf("Translate(release list, tea): %v", err)
	}
	wantInv := Invocation{Bin: "tea", Args: []string{"releases", "list", "--login", "corp", "--repo", "o/r", "--limit", "5"}}
	if !reflect.DeepEqual(got, wantInv) {
		t.Errorf("tea release list\n got %+v\nwant %+v", got, wantInv)
	}

	for _, action := range []string{"view", "download", "upload", "delete-asset"} {
		_, err := Translate(Request{Resource: "release", Action: action, Tag: "v1.0.0"}, te, Tea, "corp")
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Fatalf("Translate(release %s, tea): UsageError 기대, got %v", action, err)
		}
		if ue.Msg != "release does not support "+action {
			t.Errorf("tea 오류 = %q", ue.Msg)
		}
	}
}

func TestTranslateReleaseGlabCreateDraftUnsupported(t *testing.T) {
	gl := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "p"}
	for _, req := range []Request{
		{Resource: "release", Action: "create", Tag: "v1.0.0", Draft: true},
		{Resource: "release", Action: "create", Tag: "v1.0.0", Prerelease: true},
	} {
		_, err := Translate(req, gl, GLab, "")
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Fatalf("Translate(%+v, glab): UsageError 기대, got %v", req, err)
		}
		if ue.Msg != "release create --draft/--prerelease is not supported for glab" {
			t.Errorf("glab 오류 = %q", ue.Msg)
		}
	}
}

func TestPlanReleaseTeaListRelayedAndViewSkipsLogin(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	fakeExec(t, map[string]string{
		"BIN tea":                       "tea",
		"tea logins list --output json": `[{"name":"pub","url":"https://gitea.com"}]`,
	})

	inv, err := plan(Request{Resource: "release", Action: "list", RepoFlag: "https://gitea.com/o/r"})
	if err != nil {
		t.Fatalf("plan(release list, tea): %v", err)
	}
	if got, want := inv.Bin+" "+strings.Join(inv.Args, " "), "tea releases list --login pub --repo o/r"; got != want {
		t.Errorf("release list tea argv = %q, want %q", got, want)
	}

	// view는 tea 미지원이라 login을 묻지 않고 usage error로 끝난다.
	_, err = plan(Request{Resource: "release", Action: "view", Tag: "v1.0.0", RepoFlag: "https://gitea.com/o/r"})
	var ue UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("plan(release view, tea): UsageError 기대, got %v", err)
	}
	if ue.Msg != "release does not support view" {
		t.Errorf("tea 오류 = %q", ue.Msg)
	}
}

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
		{[]string{"release", "upload", "v1.0.0", "a.zip"}, ghRepo, wantCall("gh", "release", "upload", "v1.0.0", "a.zip", "-R", "github.com/o/r")},
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
		{[]string{"release", "edit", "v1.0.0", "--draft"}, wantTeaCall("releases", "edit", "v1.0.0", "--login", "pub", "--repo", "o/r", "--draft=true")},
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
}
