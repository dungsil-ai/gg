package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestAuthStatusRowsCoverDefaultsSettingsAndLoginValues(t *testing.T) {
	fakeExec(t, map[string]string{
		"BIN gh":   "gh",
		"BIN glab": "glab",
		"BIN tea":  "tea",
		"gh auth status --hostname github.com --json hosts": `{"hosts":{"github.com":[{"login":"dungsil","active":true}]}}`,
		"glab auth status --hostname gitlab.com":            "yes",
		"tea logins list --output csv": `"Name","URL","SSHHost","User","Default"
"my-login","https://git.example.com","","my-login","false"`,
	})
	cfg := Config{Hosts: map[string]string{"git.example.com": "tea"}}

	rows := authStatusRows(&cfg)
	want := []authStatusRow{
		{Host: "git.example.com", Provider: Tea, Login: "my-login"},
		{Host: "gitea.com", Provider: Tea, Login: "no"},
		{Host: "github.com", Provider: GH, Login: "dungsil"},
		{Host: "gitlab.com", Provider: GLab, Login: "yes"},
	}
	if !slices.Equal(rows, want) {
		t.Errorf("authStatusRows = %+v, want %+v", rows, want)
	}
}

func TestAuthStatusRowsMissingCLIsSkipLookup(t *testing.T) {
	origRunOut, origLookPath := runOut, lookPath
	t.Cleanup(func() { runOut, lookPath = origRunOut, origLookPath })
	runOut = func(name string, args ...string) (string, error) {
		t.Errorf("runOut(%s %v) should not run when the CLI is missing", name, args)
		return "", nil
	}
	lookPath = func(string) (string, error) { return "", errors.New("not installed") }

	rows := authStatusRows(&Config{})
	want := []authStatusRow{
		{Host: "gitea.com", Provider: Tea, Login: "no cli"},
		{Host: "github.com", Provider: GH, Login: "no cli"},
		{Host: "gitlab.com", Provider: GLab, Login: "no cli"},
	}
	if !slices.Equal(rows, want) {
		t.Errorf("authStatusRows = %+v, want %+v", rows, want)
	}
}

func TestAuthLoginValueReportsNoWhenLookupSaysNotLoggedIn(t *testing.T) {
	fakeExec(t, map[string]string{
		"BIN gh":   "gh",
		"BIN glab": "glab",
		"BIN tea":  "tea",
		"gh auth status --hostname github.com --json hosts":       `{"hosts":{"other.com":[{"login":"someone"}]}}`,
		"gh auth status --hostname empty.github.com --json hosts": `{"hosts":{"empty.github.com":[]}}`,
		"tea logins list --output csv":                            `"Name","URL","SSHHost","User","Default"`,
	})

	cases := []struct {
		name string
		p    Provider
		host string
		want string
	}{
		{"gh host missing", GH, "github.com", "no"},
		{"gh logged in without a name", GH, "empty.github.com", "yes"},
		{"glab failure", GLab, "gitlab.com", "no"},
		{"tea no matching login", Tea, "gitea.com", "no"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := authLoginValue(tc.p, tc.host); got != tc.want {
				t.Errorf("authLoginValue(%s, %s) = %q, want %q", tc.p, tc.host, got, tc.want)
			}
		})
	}
}

func TestGHLoginNamePrefersActiveAccountAndMatchesHostCase(t *testing.T) {
	fakeExec(t, map[string]string{
		"BIN gh": "gh",
		"gh auth status --hostname github.com --json hosts":        `{"hosts":{"github.com":[{"login":"first"},{"login":"active-one","active":true}]}}`,
		"gh auth status --hostname upper.example.com --json hosts": `{"hosts":{"UPPER.example.com":[{"login":"case-user","active":true}]}}`,
	})
	if name, ok := ghLoginName("github.com"); !ok || name != "active-one" {
		t.Errorf("ghLoginName(github.com) = %q, %v; want active-one, true", name, ok)
	}
	if name, ok := ghLoginName("upper.example.com"); !ok || name != "case-user" {
		t.Errorf("ghLoginName(upper.example.com) = %q, %v; want case-user, true", name, ok)
	}
	if _, ok := ghLoginName("missing.example.com"); ok {
		t.Error("ghLoginName for an unknown host should report not logged in")
	}
	if _, ok := ghLoginName("unlisted.example.com"); ok {
		t.Error("ghLoginName for a command failure should report not logged in")
	}
}

func TestParseRequestRejectsAuthContextFlags(t *testing.T) {
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
		if _, err := ParseRequest(tc.args); err == nil {
			t.Errorf("ParseRequest(%v) should fail", tc.args)
		} else if err.Error() != tc.want {
			t.Errorf("ParseRequest(%v) = %q, want %q", tc.args, err, tc.want)
		}
	}
}

func TestSplitAuthProvider(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantRest []string
		wantProv Provider
	}{
		{name: "없으면 gh", args: []string{"--hostname", "x"}, wantRest: []string{"--hostname", "x"}, wantProv: GH},
		{name: "값 형태", args: []string{"--provider", "glab", "--hostname", "x"}, wantRest: []string{"--hostname", "x"}, wantProv: GLab},
		{name: "등호 형태", args: []string{"--hostname", "x", "--provider=glab"}, wantRest: []string{"--hostname", "x"}, wantProv: GLab},
		{name: "명시 gh", args: []string{"--provider", "gh"}, wantRest: []string{}, wantProv: GH},
	}
	for _, tc := range cases {
		rest, p, err := splitAuthProvider(tc.args)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !slices.Equal(rest, tc.wantRest) || p != tc.wantProv {
			t.Errorf("%s: rest = %v, provider = %s, want %v / %s", tc.name, rest, p, tc.wantRest, tc.wantProv)
		}
	}

	if _, _, err := splitAuthProvider([]string{"--provider"}); err == nil {
		t.Error("값 없는 --provider는 실패해야 한다")
	}
	if _, _, err := splitAuthProvider([]string{"--provider=tea"}); err == nil {
		t.Error("--provider tea는 실패해야 한다")
	}
	if _, _, err := splitAuthProvider([]string{"--provider", "gitea"}); err == nil {
		t.Error("알 수 없는 provider는 실패해야 한다")
	}
}

func TestAuthRelayInvocationGlab(t *testing.T) {
	cases := []struct {
		name    string
		action  string
		args    []string
		wantBin string
		want    []string
		wantErr string
	}{
		{
			name: "login", action: "login", args: []string{"--provider", "glab", "--hostname", "git.example.com"},
			wantBin: "glab", want: []string{"auth", "login", "--hostname", "git.example.com"},
		},
		{
			name: "token 등호 형태", action: "token", args: []string{"--provider=glab"},
			wantBin: "glab", want: []string{"auth", "token"},
		},
		{
			name: "gh 기본값", action: "refresh", args: []string{"--hostname", "x"},
			wantBin: "gh", want: []string{"auth", "refresh", "--hostname", "x"},
		},
		{
			name: "glab에 없는 하위 명령", action: "refresh", args: []string{"--provider", "glab"},
			wantErr: "glab auth refresh is not supported",
		},
		{
			name: "glab에 없는 setup-git", action: "setup-git", args: []string{"--provider", "glab"},
			wantErr: "glab auth setup-git is not supported",
		},
		{
			name: "tea 거부", action: "login", args: []string{"--provider", "tea"},
			wantErr: "--provider tea is not supported",
		},
	}
	for _, tc := range cases {
		req := Request{Resource: "auth", Action: tc.action, GitArgs: tc.args}
		inv, err := authRelayInvocation(req)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%s: err = %v, want %q 포함", tc.name, err, tc.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if inv.Bin != tc.wantBin || strings.Join(inv.Args, " ") != strings.Join(tc.want, " ") {
			t.Errorf("%s: inv = %s %v, want %s %v", tc.name, inv.Bin, inv.Args, tc.wantBin, tc.want)
		}
	}
}
