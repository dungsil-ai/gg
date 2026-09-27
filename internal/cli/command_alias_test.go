package cli

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestParseProviderResourceAliases(t *testing.T) {
	for _, tc := range []struct {
		canonical string
		aliases   []string
	}{
		{"repo", []string{"repos"}},
		{"issue", []string{"issues"}},
		{"label", []string{"labels"}},
		{"pr", []string{"mr", "pulls"}},
		{"release", []string{"releases"}},
		{"ci", []string{"actions", "run", "pipe", "pipeline"}},
	} {
		for _, alias := range tc.aliases {
			t.Run(alias, func(t *testing.T) {
				args := []string{"--repo", "https://github.com/o/r", "--explain", alias, "list", "--limit", "7"}
				got, err := ParseRequest(args)
				want := Request{Resource: tc.canonical, Action: "list", RepoFlag: "https://github.com/o/r", Explain: true, Limit: "7"}
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("ParseRequest(%v) = %+v, %v; want %+v", args, got, err, want)
				}
				for _, tail := range [][]string{nil, {"unknown"}, {"list", "--unknown"}} {
					_, aliasErr := ParseRequest(append([]string{alias}, tail...))
					_, canonicalErr := ParseRequest(append([]string{tc.canonical}, tail...))
					if aliasErr == nil || !reflect.DeepEqual(aliasErr, canonicalErr) {
						t.Errorf("%s %v: error %v, want %v", alias, tail, aliasErr, canonicalErr)
					}
				}
			})
		}
	}
}

func TestParseProviderActionAliases(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want Request
	}{
		{[]string{"issue", "note", "42", "--body", "hello\nworld"}, Request{Resource: "issue", Action: "comment", Number: "42", Body: "hello\nworld"}},
		{[]string{"issues", "update", "42", "--title", "note"}, Request{Resource: "issue", Action: "edit", Number: "42", Title: "note"}},
		{[]string{"mr", "note", "42", "--body", "rerun", "--remote", "upstream"}, Request{Resource: "pr", Action: "comment", Number: "42", Body: "rerun", RemoteFlag: "upstream"}},
		{[]string{"pulls", "update", "42", "--body", "cancel pipeline"}, Request{Resource: "pr", Action: "edit", Number: "42", Body: "cancel pipeline"}},
		{[]string{"ci", "get"}, Request{Resource: "ci", Action: "view"}},
		{[]string{"run", "get", "123", "--explain"}, Request{Resource: "ci", Action: "view", Number: "123", Explain: true}},
		{[]string{"pipeline", "trace", "123"}, Request{Resource: "ci", Action: "watch", Number: "123"}},
		{[]string{"pipe", "trace"}, Request{Resource: "ci", Action: "watch"}},
		{[]string{"run", "rerun", "123"}, Request{Resource: "ci", Action: "retry", Number: "123"}},
		{[]string{"actions", "rerun", "123", "--help"}, Request{Resource: "ci", Action: "retry", Number: "123", Help: true}},
		{[]string{"ci", "cancel", "pipeline", "123"}, Request{Resource: "ci", Action: "cancel", Number: "123"}},
		{[]string{"run", "cancel", "pipeline", "123", "--remote", "upstream"}, Request{Resource: "ci", Action: "cancel", Number: "123", RemoteFlag: "upstream"}},
		{[]string{"pulls", "comment", "list", "42"}, Request{Resource: "pr", Action: "comment list", Number: "42"}},
		{[]string{"issues", "comment", "delete", "42", "77"}, Request{Resource: "issue", Action: "comment delete", Number: "42", CommentID: "77"}},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			got, err := ParseRequest(tc.args)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseRequest(%v) = %+v, %v; want %+v", tc.args, got, err, tc.want)
			}
		})
	}
}

func TestProviderActionAliasValidation(t *testing.T) {
	for _, tc := range []struct {
		alias     []string
		canonical []string
	}{
		{[]string{"issues", "note"}, []string{"issue", "comment"}},
		{[]string{"mr", "note", "42"}, []string{"pr", "comment", "42"}},
		{[]string{"issues", "update", "42"}, []string{"issue", "edit", "42"}},
		{[]string{"mr", "update", "42", "--draft"}, []string{"pr", "edit", "42", "--draft"}},
		{[]string{"run", "rerun"}, []string{"ci", "retry"}},
		{[]string{"ci", "get", "1", "2"}, []string{"ci", "view", "1", "2"}},
		{[]string{"ci", "trace", "--unknown"}, []string{"ci", "watch", "--unknown"}},
		{[]string{"ci", "cancel", "pipeline"}, []string{"ci", "cancel"}},
		{[]string{"ci", "cancel", "pipeline", "1", "2"}, []string{"ci", "cancel", "1", "2"}},
		{[]string{"--repo", "https://github.com/o/r", "run", "rerun", "1", "--remote", "upstream"}, []string{"--repo", "https://github.com/o/r", "ci", "retry", "1", "--remote", "upstream"}},
	} {
		t.Run(strings.Join(tc.alias, " "), func(t *testing.T) {
			_, got := ParseRequest(tc.alias)
			_, want := ParseRequest(tc.canonical)
			if got == nil || !reflect.DeepEqual(got, want) {
				t.Errorf("ParseRequest(%v) error = %v; want %v", tc.alias, got, want)
			}
		})
	}
	for _, args := range [][]string{
		{"repo", "rerun", "42"}, {"issue", "get", "42"},
		{"pr", "trace", "42"}, {"ci", "note", "42"},
		{"run", "cancel", "job", "42"},
	} {
		if _, err := ParseRequest(args); err == nil {
			t.Errorf("ParseRequest(%v) accepted an unsupported action", args)
		}
	}
}

func TestProviderAliasesPreserveGitArguments(t *testing.T) {
	gitArgs := []string{"note", "rerun", "--repo", "issues", "--help"}
	for _, prefix := range [][]string{
		{"pull"}, {"repo", "pull"}, {"repos", "pull"},
		{"update-ref"}, {"repo", "update-ref"}, {"repos", "update-ref"},
	} {
		got, err := ParseRequest(slices.Concat(prefix, gitArgs))
		if err != nil || got.Resource != "repo" || got.Action != prefix[len(prefix)-1] || !slices.Equal(got.GitArgs, gitArgs) {
			t.Errorf("%v: request = %+v, error = %v", prefix, got, err)
		}
	}
}

func TestProviderAliasHelpMatchesCanonical(t *testing.T) {
	for _, tc := range []struct {
		alias, canonical string
	}{
		{"repos", "repo"}, {"issues", "issue"}, {"labels", "label"},
		{"pulls", "pr"}, {"releases", "release"}, {"run", "ci"},
		{"pipe", "ci"}, {"pipeline", "ci"},
		{"issues update", "issue edit"}, {"issue note", "issue comment"},
		{"mr update", "pr edit"}, {"pulls note", "pr comment"},
		{"run rerun", "ci retry"}, {"pipeline get", "ci view"},
		{"pipe trace", "ci watch"}, {"actions cancel pipeline", "ci cancel"},
		{"pulls comment list", "pr comment list"},
	} {
		t.Run(tc.alias, func(t *testing.T) {
			want, wantOK := nestedHelp(append(strings.Fields(tc.canonical), "--help"))
			for _, prefix := range [][]string{nil, {"--repo", "https://github.com/o/r", "--explain"}} {
				args := slices.Concat(prefix, strings.Fields(tc.alias), []string{"--help"})
				got, ok := nestedHelp(args)
				if !ok || !wantOK || got != want {
					t.Errorf("nestedHelp(%v) = %q, %v; want %q, %v", args, got, ok, want, wantOK)
				}
			}
		})
	}
	for _, args := range [][]string{{"repos", "diff", "--help"}, {"--explain", "repos", "pull", "--help"}} {
		if _, ok := nestedHelp(args); ok {
			t.Errorf("nestedHelp(%v) intercepted Git arguments", args)
		}
	}
	for _, want := range []string{"alias: repos", "alias: issues", "alias: labels", "alias: mr, pulls", "alias: releases", "alias: actions, run, pipe, pipeline"} {
		if !strings.Contains(topLevelHelp(), want) {
			t.Errorf("top-level help is missing %q", want)
		}
	}
	for _, tc := range []struct{ resource, text string }{
		{"issue", "alias: note"}, {"issue", "alias: update"},
		{"pr", "alias: note"}, {"pr", "alias: update"},
		{"ci", "alias: get"}, {"ci", "alias: trace"},
		{"ci", "alias: rerun"}, {"ci", "alias: cancel pipeline"},
	} {
		if !strings.Contains(renderResourceHelp(commandDefs[tc.resource]), tc.text) {
			t.Errorf("%s help is missing %q", tc.resource, tc.text)
		}
	}
}

func TestProviderAliasDefinitionsHaveNoCollisions(t *testing.T) {
	resources := map[string]string{}
	for name := range commandDefs {
		resources[name] = name
	}
	for _, ad := range commandDefs["repo"].actions {
		if ad.passthrough {
			resources[ad.name] = "repo " + ad.name
		}
	}
	for name, rd := range commandDefs {
		for _, alias := range rd.aliases {
			if previous, exists := resources[alias]; exists {
				t.Errorf("%s alias %q conflicts with %s", name, alias, previous)
			}
			resources[alias] = name
		}
		actions := map[string]string{}
		for _, ad := range rd.actions {
			actions[ad.name] = ad.name
		}
		for _, ad := range rd.actions {
			for _, alias := range ad.aliases {
				if previous, exists := actions[alias]; exists {
					t.Errorf("%s %s alias %q conflicts with %s", name, ad.name, alias, previous)
				}
				actions[alias] = ad.name
			}
		}
	}
}
