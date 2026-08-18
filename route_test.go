package main

import "testing"

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		in    string
		want  RepoURL
		isErr bool
	}{
		{in: "https://github.com/cli/cli.git", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "https://GitHub.com/cli/cli", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "http://git.example.com/o/r", want: RepoURL{Host: "git.example.com", Owner: "o", Name: "r"}},
		{in: "https://gitlab.com/grp/sub/proj.git", want: RepoURL{Host: "gitlab.com", Owner: "grp/sub", Name: "proj"}},
		{in: "ssh://git@git.example.com:2222/o/r.git", want: RepoURL{Host: "git.example.com", Owner: "o", Name: "r"}},
		{in: "git@github.com:cli/cli.git", want: RepoURL{Host: "github.com", Owner: "cli", Name: "cli"}},
		{in: "git@gitea.com:gitea/tea", want: RepoURL{Host: "gitea.com", Owner: "gitea", Name: "tea"}},
		{in: "", isErr: true},
		{in: "ftp://x.com/a/b", isErr: true},
		{in: "https://github.com/onlyowner", isErr: true},
		{in: "https:///a/b", isErr: true},
		{in: "plain-text", isErr: true},
	}
	for _, c := range cases {
		got, err := ParseRepoURL(c.in)
		if c.isErr {
			if err == nil {
				t.Errorf("ParseRepoURL(%q): error 기대, got %+v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRepoURL(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseRepoURL(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestRepoURLHelpers(t *testing.T) {
	r := RepoURL{Host: "git.example.com", Owner: "grp/sub", Name: "proj"}
	if r.Slug() != "grp/sub/proj" {
		t.Errorf("Slug() = %q", r.Slug())
	}
	if r.HTTPS() != "https://git.example.com/grp/sub/proj" {
		t.Errorf("HTTPS() = %q", r.HTTPS())
	}
}
