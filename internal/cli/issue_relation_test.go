package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestIssueRelationParse는 관계 등록 action이 positional과 flag를 Request로 옮기는지 본다.
func TestIssueRelationParse(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{
			name: "sub-issue",
			args: []string{"issue", "sub-issue", "42", "--parent", "7"},
			want: Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7"},
		},
		{
			name: "blocked-by",
			args: []string{"issue", "blocked-by", "42", "--blocker", "7"},
			want: Request{Resource: "issue", Action: "blocked-by", Number: "42", Blocker: "7"},
		},
		{
			name: "type",
			args: []string{"issue", "type", "42", "--name", "Bug"},
			want: Request{Resource: "issue", Action: "type", Number: "42", IssueType: "Bug"},
		},
		{
			name: "context flag before command",
			args: []string{"--repo", "https://github.com/o/r", "issue", "sub-issue", "42", "--parent", "7"},
			want: Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7", RepoFlag: "https://github.com/o/r"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := ParseRequest(tc.args)
			if err != nil {
				t.Fatalf("ParseRequest(%v): %v", tc.args, err)
			}
			if req.Resource != tc.want.Resource || req.Action != tc.want.Action ||
				req.Number != tc.want.Number || req.Parent != tc.want.Parent ||
				req.Blocker != tc.want.Blocker || req.IssueType != tc.want.IssueType ||
				req.RepoFlag != tc.want.RepoFlag {
				t.Errorf("ParseRequest(%v) = %+v, want %+v", tc.args, req, tc.want)
			}
		})
	}
}

// TestIssueRelationParseErrors는 관계 등록 action의 사용법 오류를 본다.
func TestIssueRelationParseErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "sub-issue missing parent",
			args: []string{"issue", "sub-issue", "42"},
			want: "usage: gg issue sub-issue <number> --parent <parent>",
		},
		{
			name: "sub-issue missing number",
			args: []string{"issue", "sub-issue"},
			want: "usage: gg issue sub-issue <number> --parent <parent>",
		},
		{
			name: "sub-issue non-number parent",
			args: []string{"issue", "sub-issue", "42", "--parent", "https://github.com/o/r/issues/7"},
			want: "usage: gg issue sub-issue <number> --parent <parent>",
		},
		{
			name: "sub-issue self parent",
			args: []string{"issue", "sub-issue", "42", "--parent", "42"},
			want: "--parent must be a different issue than 42",
		},
		{
			name: "blocked-by missing blocker",
			args: []string{"issue", "blocked-by", "42"},
			want: "usage: gg issue blocked-by <number> --blocker <blocker>",
		},
		{
			name: "blocked-by self blocker",
			args: []string{"issue", "blocked-by", "42", "--blocker", "42"},
			want: "--blocker must be a different issue than 42",
		},
		{
			name: "type missing name",
			args: []string{"issue", "type", "42"},
			want: "usage: gg issue type <number> --name <name>",
		},
		{
			name: "type blank name",
			args: []string{"issue", "type", "42", "--name", " "},
			want: "usage: gg issue type <number> --name <name>",
		},
		{
			name: "type non-number",
			args: []string{"issue", "type", "latest", "--name", "Bug"},
			want: "usage: gg issue type <number> --name <name>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRequest(tc.args)
			var ue UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("ParseRequest(%v) = %v, want UsageError %q", tc.args, err, tc.want)
			}
			if ue.Msg != tc.want {
				t.Errorf("ParseRequest(%v) = %q, want %q", tc.args, ue.Msg, tc.want)
			}
		})
	}
}

// TestIssueRelationTranslateGH는 gh 번역이 numeric database id를 body로 옮기는지 본다.
func TestIssueRelationTranslateGH(t *testing.T) {
	r := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	cases := []struct {
		name string
		req  Request
		want []string
	}{
		{
			name: "sub-issue",
			req:  Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7", RelatedID: "5277108047"},
			want: []string{"api", "--method", "POST", "repos/o/r/issues/7/sub_issues", "--hostname", "github.com", "-F", "sub_issue_id=5277108047"},
		},
		{
			name: "blocked-by",
			req:  Request{Resource: "issue", Action: "blocked-by", Number: "42", Blocker: "7", RelatedID: "100"},
			want: []string{"api", "--method", "POST", "repos/o/r/issues/42/dependencies/blocked_by", "--hostname", "github.com", "-F", "issue_id=100"},
		},
		{
			name: "type keeps the name as-is",
			req:  Request{Resource: "issue", Action: "type", Number: "42", IssueType: "유형:조사"},
			want: []string{"api", "--method", "PATCH", "repos/o/r/issues/42", "--hostname", "github.com", "-F", "type=유형:조사"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv, err := Translate(tc.req, r, GH, "")
			if err != nil {
				t.Fatalf("Translate(%+v): %v", tc.req, err)
			}
			if inv.Bin != "gh" || !slices.Equal(inv.Args, tc.want) {
				t.Errorf("Translate(%+v) = %s %v, want gh %v", tc.req, inv.Bin, inv.Args, tc.want)
			}
		})
	}
}

// TestIssueRelationUnsupportedProviders는 관계 등록이 glab/tea에서 미지원으로
// 확정되는지 본다.
func TestIssueRelationUnsupportedProviders(t *testing.T) {
	r := RepoURL{Host: "gitlab.com", Owner: "o", Name: "r"}
	for _, p := range []Provider{GLab, Tea} {
		for _, action := range []string{"sub-issue", "blocked-by", "type"} {
			_, err := Translate(Request{Resource: "issue", Action: action, Number: "42"}, r, p, "login")
			var ue UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("%s %s: %v, want UsageError", p, action, err)
			}
			want := "issue " + action + " is not supported for " + string(p)
			if ue.Msg != want {
				t.Errorf("%s %s = %q, want %q", p, action, ue.Msg, want)
			}
		}
	}
}

// TestResolvePlanIssueRelationIDs는 resolvePlan이 sub-issue와 blocked-by에 한해
// 번호→database id 조회를 실행하고, explain에서는 건너뛰는지 본다.
func TestResolvePlanIssueRelationIDs(t *testing.T) {
	t.Setenv("GG_HOME", t.TempDir())
	orig := runOut
	t.Cleanup(func() { runOut = orig })
	var ghArgs [][]string
	runOut = func(name string, args ...string) (string, error) {
		if name == "gh" {
			ghArgs = append(ghArgs, append([]string{name}, args...))
		}
		return "5277108047", nil
	}

	t.Run("sub-issue resolves the child id", func(t *testing.T) {
		ghArgs = nil
		ep, err := resolvePlan(Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7", RepoFlag: "https://github.com/o/r"})
		if err != nil {
			t.Fatalf("resolvePlan: %v", err)
		}
		wantLookup := []string{"gh", "api", "--hostname", "github.com", "repos/o/r/issues/42", "--jq", ".id"}
		if len(ghArgs) != 1 || !slices.Equal(ghArgs[0], wantLookup) {
			t.Errorf("gh calls = %v, want [%v]", ghArgs, wantLookup)
		}
		if !slices.Contains(ep.inv.Args, "-F") || !slices.Contains(ep.inv.Args, "sub_issue_id=5277108047") {
			t.Errorf("args = %v, want sub_issue_id=5277108047", ep.inv.Args)
		}
	})

	t.Run("blocked-by resolves the blocker id", func(t *testing.T) {
		ghArgs = nil
		ep, err := resolvePlan(Request{Resource: "issue", Action: "blocked-by", Number: "42", Blocker: "7", RepoFlag: "https://github.com/o/r"})
		if err != nil {
			t.Fatalf("resolvePlan: %v", err)
		}
		wantLookup := []string{"gh", "api", "--hostname", "github.com", "repos/o/r/issues/7", "--jq", ".id"}
		if len(ghArgs) != 1 || !slices.Equal(ghArgs[0], wantLookup) {
			t.Errorf("gh calls = %v, want [%v]", ghArgs, wantLookup)
		}
		if !slices.Contains(ep.inv.Args, "issue_id=5277108047") {
			t.Errorf("args = %v, want issue_id=5277108047", ep.inv.Args)
		}
	})

	t.Run("type does not resolve", func(t *testing.T) {
		ghArgs = nil
		ep, err := resolvePlan(Request{Resource: "issue", Action: "type", Number: "42", IssueType: "Bug", RepoFlag: "https://github.com/o/r"})
		if err != nil {
			t.Fatalf("resolvePlan: %v", err)
		}
		if len(ghArgs) != 0 {
			t.Errorf("gh calls = %v, want none", ghArgs)
		}
		if !slices.Contains(ep.inv.Args, "type=Bug") {
			t.Errorf("args = %v, want type=Bug", ep.inv.Args)
		}
	})

	t.Run("explain skips the lookup", func(t *testing.T) {
		ghArgs = nil
		ep, err := resolvePlan(Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7", RepoFlag: "https://github.com/o/r", Explain: true})
		if err != nil {
			t.Fatalf("resolvePlan: %v", err)
		}
		if len(ghArgs) != 0 {
			t.Errorf("gh calls = %v, want none", ghArgs)
		}
		if ep.provider != GH || ep.inv.Bin != "gh" {
			t.Errorf("plan = %s/%s, want gh", ep.provider, ep.inv.Bin)
		}
	})

	t.Run("lookup failure is wrapped", func(t *testing.T) {
		runOut = func(name string, args ...string) (string, error) {
			return "", errors.New("HTTP 404: Not Found")
		}
		_, err := resolvePlan(Request{Resource: "issue", Action: "sub-issue", Number: "42", Parent: "7", RepoFlag: "https://github.com/o/r"})
		if err == nil || !strings.Contains(err.Error(), "cannot resolve issue 42 id") {
			t.Errorf("resolvePlan error = %v, want cannot resolve issue 42 id", err)
		}
	})
}

// TestGHIssueDatabaseID는 번호→database id 조회 argv와 오류 문구를 본다.
func TestGHIssueDatabaseID(t *testing.T) {
	orig := runOut
	t.Cleanup(func() { runOut = orig })
	r := RepoURL{Host: "github.com", Owner: "o", Name: "r"}

	var gotArgs []string
	runOut = func(name string, args ...string) (string, error) {
		gotArgs = append([]string{name}, args...)
		return "5277108047", nil
	}
	id, err := ghIssueDatabaseID(r, "42")
	if err != nil || id != "5277108047" {
		t.Fatalf("ghIssueDatabaseID(42) = %q, %v; want 5277108047", id, err)
	}
	want := []string{"gh", "api", "--hostname", "github.com", "repos/o/r/issues/42", "--jq", ".id"}
	if !slices.Equal(gotArgs, want) {
		t.Errorf("argv = %v, want %v", gotArgs, want)
	}

	runOut = func(name string, args ...string) (string, error) { return "", errors.New("boom") }
	if _, err := ghIssueDatabaseID(r, "999"); err == nil || !strings.Contains(err.Error(), "cannot resolve issue 999 id") {
		t.Errorf("ghIssueDatabaseID(999) error = %v, want cannot resolve issue 999 id", err)
	}

	runOut = func(name string, args ...string) (string, error) { return "", nil }
	if _, err := ghIssueDatabaseID(r, "1"); err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Errorf("ghIssueDatabaseID(1) error = %v, want empty response", err)
	}
}
