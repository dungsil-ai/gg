package cli

import (
	"errors"
	"slices"
	"testing"
)

func TestAuthStatusRowsCoverDefaultsSettingsAndLoginValues(t *testing.T) {
	fakeExec(t, map[string]string{
		"BIN gh":   "gh",
		"BIN glab": "glab",
		"BIN tea":  "tea",
		"gh auth status --hostname github.com --json hosts": `{"hosts":{"github.com":[{"login":"dungsil","active":true}]}}`,
		"glab auth status --hostname gitlab.com":            "yes",
		"tea logins list --output json":                     `[{"name":"my-login","url":"https://git.example.com"}]`,
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
		"tea logins list --output json":                           `[]`,
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
