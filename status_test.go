package main

import (
	"strings"
	"testing"
)

func TestParseRequestPRStatus(t *testing.T) {
	req, err := ParseRequest([]string{"pr", "status", "42"})
	if err != nil {
		t.Fatalf("pr status 42: %v", err)
	}
	if req.Resource != "pr" || req.Action != "status" || req.Number != "42" {
		t.Fatalf("req = %+v", req)
	}
	req, err = ParseRequest([]string{"pr", "status", "42", "--remote", "upstream"})
	if err != nil || req.RemoteFlag != "upstream" {
		t.Fatalf("--remote: req %+v, err %v", req, err)
	}
	for _, args := range [][]string{
		{"pr", "status"},
		{"pr", "status", "1", "2"},
	} {
		if _, err := ParseRequest(args); err == nil {
			t.Errorf("ParseRequest(%v) should fail", args)
		}
	}
}

func TestTranslatePRStatus(t *testing.T) {
	r := RepoURL{Host: "github.com", Owner: "o", Name: "r"}
	inv, err := Translate(Request{Resource: "pr", Action: "status", Number: "7"}, r, GH, "")
	if err != nil {
		t.Fatalf("gh: %v", err)
	}
	want := "pr view 7 -R github.com/o/r --json isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup"
	if got := strings.Join(inv.Args, " "); got != want {
		t.Errorf("gh args = %q, want %q", got, want)
	}

	r = RepoURL{Host: "gitlab.com", Owner: "o", Name: "r"}
	inv, err = Translate(Request{Resource: "pr", Action: "status", Number: "7"}, r, GLab, "")
	if err != nil {
		t.Fatalf("glab: %v", err)
	}
	want = "mr view 7 --output json --repo https://gitlab.com/o/r"
	if got := strings.Join(inv.Args, " "); got != want {
		t.Errorf("glab args = %q, want %q", got, want)
	}

	if _, err := Translate(Request{Resource: "pr", Action: "status", Number: "7"}, RepoURL{Host: "gitea.com", Owner: "o", Name: "r"}, Tea, "pub"); err == nil {
		t.Error("tea pr status should fail")
	} else if !strings.Contains(err.Error(), "not supported for tea") {
		t.Errorf("tea error = %v", err)
	}
}

func TestParseGHStatus(t *testing.T) {
	cases := []struct {
		name string
		json string
		want prStatus
	}{
		{"ready", `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[{"__typename":"StatusContext","state":"SUCCESS"}]}`,
			prStatus{Draft: "no", Approval: "approved", CI: "pass", Conflict: "no", Mergeable: "yes"}},
		{"changes requested", `{"isDraft":false,"reviewDecision":"CHANGES_REQUESTED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[{"__typename":"CheckRun","status":"COMPLETED","conclusion":"SUCCESS"}]}`,
			prStatus{Draft: "no", Approval: "changes-requested", CI: "pass", Conflict: "no", Mergeable: "yes"}},
		{"review required", `{"isDraft":true,"reviewDecision":"REVIEW_REQUIRED","mergeable":"MERGEABLE","mergeStateStatus":"BLOCKED","statusCheckRollup":[{"__typename":"StatusContext","state":"SUCCESS"}]}`,
			prStatus{Draft: "yes", Approval: "required", CI: "pass", Conflict: "no", Mergeable: "yes"}},
		{"ci pending", `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[{"__typename":"CheckRun","status":"IN_PROGRESS","conclusion":""}]}`,
			prStatus{Draft: "no", Approval: "approved", CI: "pending", Conflict: "no", Mergeable: "yes"}},
		{"ci fail", `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"UNSTABLE","statusCheckRollup":[{"__typename":"StatusContext","state":"SUCCESS"},{"__typename":"CheckRun","status":"COMPLETED","conclusion":"FAILURE"}]}`,
			prStatus{Draft: "no", Approval: "approved", CI: "fail", Conflict: "no", Mergeable: "yes"}},
		{"ci none", `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[]}`,
			prStatus{Draft: "no", Approval: "approved", CI: "none", Conflict: "no", Mergeable: "yes"}},
		{"conflict", `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"CONFLICTING","mergeStateStatus":"DIRTY","statusCheckRollup":[{"__typename":"StatusContext","state":"SUCCESS"}]}`,
			prStatus{Draft: "no", Approval: "approved", CI: "pass", Conflict: "yes", Mergeable: "no"}},
		{"computing", `{"isDraft":false,"reviewDecision":"","mergeable":"UNKNOWN","mergeStateStatus":"UNKNOWN","statusCheckRollup":[{"__typename":"StatusContext","state":"PENDING"}]}`,
			prStatus{Draft: "no", Approval: "required", CI: "pending", Conflict: "unknown", Mergeable: "unknown"}},
		{"empty review decision", `{"isDraft":false,"mergeable":"MERGEABLE","mergeStateStatus":"CLEAN"}`,
			prStatus{Draft: "no", Approval: "required", CI: "none", Conflict: "no", Mergeable: "yes"}},
	}
	for _, tc := range cases {
		s, err := parseGHStatus([]byte(tc.json))
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if s != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, s, tc.want)
		}
	}
	if _, err := parseGHStatus([]byte("not json")); err == nil {
		t.Error("bad JSON should fail")
	}
}

func TestParseGLabStatus(t *testing.T) {
	cases := []struct {
		name string
		json string
		want prStatus
	}{
		{"ready", `{"draft":false,"approved_by":[{"user":{"username":"a"}}],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable","head_pipeline":{"status":"success"}}`,
			prStatus{Draft: "no", Approval: "approved", CI: "pass", Conflict: "no", Mergeable: "yes"}},
		{"approval required", `{"draft":false,"approved_by":[],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable","head_pipeline":{"status":"success"}}`,
			prStatus{Draft: "no", Approval: "required", CI: "pass", Conflict: "no", Mergeable: "yes"}},
		{"pipeline pending", `{"draft":true,"approved_by":[],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable","head_pipeline":{"status":"running"}}`,
			prStatus{Draft: "yes", Approval: "required", CI: "pending", Conflict: "no", Mergeable: "yes"}},
		{"no pipeline", `{"draft":false,"approved_by":[],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable"}`,
			prStatus{Draft: "no", Approval: "required", CI: "none", Conflict: "no", Mergeable: "yes"}},
		{"conflict", `{"draft":false,"approved_by":[],"has_conflicts":true,"merge_status":"cannot_be_merged","detailed_merge_status":"conflicts","head_pipeline":{"status":"failed"}}`,
			prStatus{Draft: "no", Approval: "required", CI: "fail", Conflict: "yes", Mergeable: "no"}},
		{"unknown merge status", `{"draft":false,"approved_by":[],"has_conflicts":false,"merge_status":"unknown","detailed_merge_status":"checking"}`,
			prStatus{Draft: "no", Approval: "required", CI: "none", Conflict: "no", Mergeable: "unknown"}},
		{"missing fields", `{}`,
			prStatus{Draft: "unknown", Approval: "unknown", CI: "none", Conflict: "unknown", Mergeable: "unknown"}},
		{"fallback merge status", `{"draft":false,"approved_by":[],"has_conflicts":false,"merge_status":"can_be_merged"}`,
			prStatus{Draft: "no", Approval: "required", CI: "none", Conflict: "no", Mergeable: "yes"}},
	}
	for _, tc := range cases {
		s, err := parseGLabStatus([]byte(tc.json))
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if s != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, s, tc.want)
		}
	}
	if _, err := parseGLabStatus([]byte("nope")); err == nil {
		t.Error("bad JSON should fail")
	}
}
