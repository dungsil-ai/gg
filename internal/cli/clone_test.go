package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func isolateCloneConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GG_HOME", t.TempDir())
	t.Setenv("GLAB_CONFIG_DIR", t.TempDir())
	for _, key := range []string{"GH_HOST", "GITLAB_HOST", "GITLAB_URI", "GL_HOST"} {
		t.Setenv(key, "")
	}
}

func TestPlanCloneShorthand(t *testing.T) {
	cases := []struct {
		name      string
		slug      string
		responses map[string]string
		glabYAML  string
		want      Invocation
		wantRepo  RepoURL
	}{
		{
			name: "github", slug: "o/r",
			responses: map[string]string{
				"BIN gh":                                 "",
				"gh auth status --json hosts":            `{"hosts":{"github.com":[{"active":true}]}}`,
				"gh api --hostname github.com repos/o/r": `{"id":1}`,
			},
			want:     Invocation{Bin: "gh", Args: []string{"repo", "clone", "https://github.com/o/r", "destination path"}},
			wantRepo: RepoURL{Host: "github.com", Owner: "o", Name: "r"},
		},
		{
			name: "gitlab after github miss", slug: "o/r.git",
			responses: map[string]string{
				"BIN gh": "", "BIN glab": "",
				"gh auth status --json hosts":                   `{"hosts":{"github.com":[]}}`,
				"glab api --hostname gitlab.com projects/o%2Fr": `{"id":1}`,
			},
			want:     Invocation{Bin: "glab", Args: []string{"repo", "clone", "https://gitlab.com/o/r", "destination path"}},
			wantRepo: RepoURL{Host: "gitlab.com", Owner: "o", Name: "r"},
		},
		{
			name: "github enterprise", slug: "o/r",
			responses: map[string]string{
				"BIN gh":                      "",
				"gh auth status --json hosts": `{"hosts":{"GIT.EXAMPLE.COM":[{"active":true}]}}`,
				"gh api --hostname git.example.com repos/o/r": `{"id":1}`,
			},
			want:     Invocation{Bin: "gh", Args: []string{"repo", "clone", "https://git.example.com/o/r", "destination path"}},
			wantRepo: RepoURL{Host: "git.example.com", Owner: "o", Name: "r"},
		},
		{
			name: "gitlab nested namespace and custom port", slug: "group/sub/repo",
			responses: map[string]string{
				"BIN glab": "",
				"glab api --hostname git.example.com:8443 projects/group%2Fsub%2Frepo": `{"id":1}`,
			},
			glabYAML: "hosts:\n  git.example.com:8443:\n    token: secret-not-to-print\n",
			want:     Invocation{Bin: "glab", Args: []string{"repo", "clone", "https://git.example.com:8443/group/sub/repo", "destination path"}},
			wantRepo: RepoURL{Host: "git.example.com", Owner: "group/sub", Name: "repo"},
		},
		{
			name: "tea login with port and subpath", slug: "o/r",
			responses: map[string]string{
				"BIN tea":                      "",
				"tea logins list --output csv": "\"Name\",\"URL\",\"User\"\n\"corp, work\",\"https://git.example.com:8443/gitea\",\"me\"\n",
				"tea api repos/o/r --login corp, work --repo o/r": `{"id":1}`,
			},
			want:     Invocation{Bin: "tea", Args: []string{"clone", "--login", "corp, work", "o/r", "destination path"}},
			wantRepo: RepoURL{Host: "git.example.com", Owner: "gitea/o", Name: "r"},
		},
		{
			name: "tea duplicate logins identify one repository", slug: "o/r",
			responses: map[string]string{
				"BIN tea":                                "",
				"tea logins list --output csv":           "Name,URL,User\nz,https://gitea.com,me\na,https://gitea.com,me\n",
				"tea api repos/o/r --login a --repo o/r": `{"id":1}`,
				"tea api repos/o/r --login z --repo o/r": `{"id":1}`,
			},
			want:     Invocation{Bin: "tea", Args: []string{"clone", "--login", "a", "o/r", "destination path"}},
			wantRepo: RepoURL{Host: "gitea.com", Owner: "o", Name: "r"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCloneConfig(t)
			if tc.glabYAML != "" {
				if err := os.WriteFile(filepath.Join(os.Getenv("GLAB_CONFIG_DIR"), "config.yml"), []byte(tc.glabYAML), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			fakeExec(t, tc.responses)
			lookup := runOut
			runOut = func(bin string, args ...string) (string, error) {
				if bin == "git" || args[0] == "clone" || args[0] == "repo" {
					t.Fatalf("해석 중 저장소 문맥 조회나 clone을 실행하면 안 된다: %s %v", bin, args)
				}
				return lookup(bin, args...)
			}
			for _, explain := range []bool{false, true} {
				got, err := resolvePlan(Request{Resource: "repo", Action: "clone", CloneURL: tc.slug, CloneDir: "destination path", Explain: explain})
				if err != nil || !reflect.DeepEqual(got.inv, tc.want) || got.repo != tc.wantRepo {
					t.Fatalf("resolvePlan(explain=%v) = %+v, %v; want %+v, %+v", explain, got, err, tc.want, tc.wantRepo)
				}
			}
		})
	}
}

func TestCloneShorthandAmbiguous(t *testing.T) {
	for _, sameProvider := range []bool{false, true} {
		t.Run(map[bool]string{false: "different providers", true: "same provider"}[sameProvider], func(t *testing.T) {
			isolateCloneConfig(t)
			responses := map[string]string{
				"BIN gh": "", "BIN glab": "",
				"gh auth status --json hosts":                   `{"hosts":{"github.com":[]}}`,
				"gh api --hostname github.com repos/o/r":        `{"id":1}`,
				"glab api --hostname gitlab.com projects/o%2Fr": `{"id":1}`,
			}
			secondURL := "https://gitlab.com/o/r"
			if sameProvider {
				delete(responses, "BIN glab")
				responses["gh auth status --json hosts"] = `{"hosts":{"github.com":[],"git.example.com":[]}}`
				responses["gh api --hostname git.example.com repos/o/r"] = `{"id":1}`
				secondURL = "https://git.example.com/o/r"
			}
			fakeExec(t, responses)
			_, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
			for _, want := range []string{"multiple repositories match", "https://github.com/o/r", secondURL, "use a full repository URL"} {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v; want %q", err, want)
				}
			}
		})
	}
}

func TestCloneShorthandNotFound(t *testing.T) {
	for _, installed := range []bool{false, true} {
		t.Run(map[bool]string{false: "no CLI", true: "lookup failure"}[installed], func(t *testing.T) {
			isolateCloneConfig(t)
			responses := map[string]string{}
			if installed {
				responses["BIN gh"] = ""
				responses["gh auth status --json hosts"] = `{"hosts":{"github.com":[]}}`
			}
			fakeExec(t, responses)
			_, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
			if err == nil || !strings.Contains(err.Error(), "cannot find an accessible repository") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCloneShorthandInvalid(t *testing.T) {
	isolateCloneConfig(t)
	fakeExec(t, nil)
	runOut = func(bin string, args ...string) (string, error) {
		t.Fatalf("잘못된 축약형으로 CLI를 실행하면 안 된다: %s %v", bin, args)
		return "", nil
	}
	for _, raw := range []string{"repo", "", "o/", "/o/r", "./o/r", "../repo", "o/../r", "o//r", "o/-r", "o/.git", "o/r?x=1", "o/r#ref", "o/r%2Fother", "o/r\\other", "o/r\n", "o/r s", "git@host/o/r"} {
		_, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: raw})
		if err == nil || !strings.Contains(err.Error(), "<namespace>/<name>") {
			t.Errorf("plan(%q) = %v", raw, err)
		}
	}
}

func TestCloneShorthandTeaHTTP(t *testing.T) {
	isolateCloneConfig(t)
	fakeExec(t, map[string]string{
		"BIN tea":                      "",
		"tea logins list --output csv": "Name,URL\nlocal,http://git.example.com:3000\n",
		"tea api repos/o/r --login local --repo o/r": `{"id":1}`,
	})
	lookup := runOut
	apiCalls := 0
	runOut = func(bin string, args ...string) (string, error) {
		if args[0] == "api" {
			apiCalls++
		}
		return lookup(bin, args...)
	}
	req := Request{Resource: "repo", Action: "clone", CloneURL: "o/r"}
	if _, err := plan(req); err == nil || !strings.Contains(err.Error(), "HTTP clone is blocked") || apiCalls != 0 {
		t.Fatalf("HTTP 조회도 차단되어야 한다: %v, calls=%d", err, apiCalls)
	}
	req.AllowInsecureHTTP = true
	got, err := plan(req)
	want := Invocation{Bin: "tea", Args: []string{"clone", "--login", "local", "o/r"}}
	if err != nil || !reflect.DeepEqual(got, want) || apiCalls != 1 {
		t.Fatalf("plan = %+v, %v, calls=%d", got, err, apiCalls)
	}
}

func TestCloneShorthandConfigAndEnvironment(t *testing.T) {
	for _, source := range []string{"gg config", "GH_HOST", "GITLAB_HOST", "glab config path"} {
		t.Run(source, func(t *testing.T) {
			isolateCloneConfig(t)
			responses := map[string]string{}
			wantBin := "gh"
			switch source {
			case "gg config":
				if err := SaveProvider("git.example.com", GH); err != nil {
					t.Fatal(err)
				}
			case "GH_HOST":
				t.Setenv("GH_HOST", "git.example.com")
			case "GITLAB_HOST":
				wantBin = "glab"
				t.Setenv("GITLAB_HOST", "https://git.example.com")
			case "glab config path":
				wantBin = "glab"
				path := filepath.Join(t.TempDir(), "config.yml")
				if err := os.WriteFile(path, []byte("hosts:\n  git.example.com:\n    token: secret\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				responses["glab config path"] = path
			}
			responses["BIN "+wantBin] = ""
			if wantBin == "gh" {
				responses["gh api --hostname git.example.com repos/o/r"] = `{"id":1}`
			} else {
				responses["glab api --hostname git.example.com projects/o%2Fr"] = `{"id":1}`
			}
			fakeExec(t, responses)
			got, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
			if err != nil || got.Bin != wantBin || got.Args[2] != "https://git.example.com/o/r" {
				t.Fatalf("plan = %+v, %v", got, err)
			}
		})
	}
}

func TestCloneShorthandMalformedDiscoveryDoesNotLeakSecrets(t *testing.T) {
	isolateCloneConfig(t)
	fakeExec(t, map[string]string{
		"BIN gh": "", "BIN glab": "", "BIN tea": "",
		"gh auth status --json hosts":  "secret-gh",
		"tea logins list --output csv": "Name,URL\n\"secret-tea",
	})
	if err := os.WriteFile(filepath.Join(os.Getenv("GLAB_CONFIG_DIR"), "config.yml"), []byte("hosts: [secret-glab"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
	if err == nil || strings.Contains(err.Error(), "secret-") {
		t.Fatalf("오류에 인증정보가 노출되면 안 된다: %v", err)
	}
	for _, want := range []string{"gh hosts", "glab hosts", "tea logins"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("진단 누락: %q in %v", want, err)
		}
	}
}

func TestCloneShorthandRejectsNonRepositoryResponse(t *testing.T) {
	for _, out := range []string{"", "<html>Sign in</html>", `{}`, `null`, `{"message":"Not Found"}`, `{"id":0}`, `{"id":"bad"}`} {
		t.Run(out, func(t *testing.T) {
			isolateCloneConfig(t)
			fakeExec(t, map[string]string{
				"BIN gh":                                 "",
				"gh auth status --json hosts":            `{"hosts":{"github.com":[]}}`,
				"gh api --hostname github.com repos/o/r": out,
			})
			if _, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"}); err == nil {
				t.Fatalf("저장소가 아닌 응답을 clone 대상으로 해석했다: %q", out)
			}
		})
	}
}

func TestCloneShorthandTriesOtherTeaAccounts(t *testing.T) {
	isolateCloneConfig(t)
	fakeExec(t, map[string]string{
		"BIN tea":                                "",
		"tea logins list --output csv":           "Name,URL\na,https://git.example.com\nz,https://git.example.com\n",
		"tea api repos/o/r --login z --repo o/r": `{"id":1}`,
	})
	got, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
	want := Invocation{Bin: "tea", Args: []string{"clone", "--login", "z", "o/r"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("접근 가능한 tea 계정을 유지해야 한다: %+v, %v", got, err)
	}
}

func TestCloneShorthandStaleProviderConfig(t *testing.T) {
	isolateCloneConfig(t)
	if err := SaveProvider("git.example.com", GLab); err != nil {
		t.Fatal(err)
	}
	fakeExec(t, map[string]string{
		"BIN gh": "", "BIN glab": "",
		"gh auth status --json hosts":                 `{"hosts":{"git.example.com":[]}}`,
		"gh api --hostname git.example.com repos/o/r": `{"id":1}`,
	})
	got, err := plan(Request{Resource: "repo", Action: "clone", CloneURL: "o/r"})
	if err != nil || got.Bin != "gh" {
		t.Fatalf("등록된 CLI의 실제 저장소 조회 결과를 사용해야 한다: %+v, %v", got, err)
	}
}
