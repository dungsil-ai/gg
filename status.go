package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// 테스트 교체점.
var (
	osStdout  io.Writer = os.Stdout
	osStderr  io.Writer = os.Stderr
	osEnviron           = os.Environ
)

// exitCodeError는 자식 process의 실패 exit code를 그대로 보고하기 위한 것이다.
type exitCodeError struct {
	Code int
	Msg  string
}

func (e exitCodeError) Error() string { return e.Msg }

func childFailCode(err error) int {
	var ec exitCodeError
	if errors.As(err, &ec) {
		return ec.Code
	}
	return 1
}

// prStatus는 provider 상관없이 같은 필드와 값 범위를 쓴다.
type prStatus struct {
	Draft     string // yes | no | unknown
	Approval  string // approved | required | changes-requested | unknown
	CI        string // pass | fail | pending | none | unknown
	Conflict  string // yes | no | unknown
	Mergeable string // yes | no | unknown
}

func (s prStatus) text() string {
	return fmt.Sprintf("Draft: %s\nApproval: %s\nCI: %s\nConflict: %s\nMergeable: %s\n",
		s.Draft, s.Approval, s.CI, s.Conflict, s.Mergeable)
}

func ghStatusFields() string {
	return "isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup"
}

// runPRStatus는 상태 조회 성공 시 exit 0이다. 병합 불가, CI 실패, 승인 대기는
// 결과 값일 뿐이고, 조회 자체가 실패할 때만 0이 아닌 exit code를 낸다.
func runPRStatus(ep executionPlan) int {
	out, err := captureChild(ep.inv)
	if err != nil {
		fmt.Fprintln(osStderr, "gg:", err)
		return childFailCode(err)
	}
	s, perr := parsePRStatus(ep.provider, out)
	if perr != nil {
		fmt.Fprintln(osStderr, "gg:", perr)
		return 1
	}
	fmt.Fprint(osStdout, s.text())
	return 0
}

func captureChild(inv Invocation) (string, error) {
	path, err := lookPath(inv.Bin)
	if err != nil {
		return "", exitCodeError{Code: 127, Msg: fmt.Sprintf("%s is not installed or not on PATH", inv.Bin)}
	}
	cmd := exec.Command(path, inv.Args...)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = osStderr
	cmd.Env = append(osEnviron(), inv.Env...)
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code := childExitCode(ee)
			return "", exitCodeError{Code: code, Msg: fmt.Sprintf("%s failed with exit code %d", inv.Bin, code)}
		}
		return "", err
	}
	return stdout.String(), nil
}

func parsePRStatus(p Provider, out string) (prStatus, error) {
	switch p {
	case GH:
		return parseGHStatus([]byte(out))
	case GLab:
		return parseGLabStatus([]byte(out))
	}
	return prStatus{}, usageErr("pr status is not supported for " + string(p))
}

func parseGHStatus(data []byte) (prStatus, error) {
	var v struct {
		IsDraft           bool      `json:"isDraft"`
		ReviewDecision    string    `json:"reviewDecision"`
		Mergeable         string    `json:"mergeable"`
		MergeStateStatus  string    `json:"mergeStateStatus"`
		StatusCheckRollup []ghCheck `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return prStatus{}, fmt.Errorf("cannot parse gh pr view output: %v", err)
	}
	s := prStatus{
		Draft:     yesNo(v.IsDraft),
		Approval:  ghApproval(v.ReviewDecision),
		CI:        ghCIRollup(v.StatusCheckRollup),
		Conflict:  ghConflict(v.Mergeable, v.MergeStateStatus),
		Mergeable: ghMergeable(v.Mergeable, v.MergeStateStatus),
	}
	return s, nil
}

type ghCheck struct {
	TypeName   string `json:"__typename"`
	State      string `json:"state"`
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
}

func ghApproval(decision string) string {
	switch decision {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes-requested"
	case "REVIEW_REQUIRED", "":
		return "required"
	}
	return "unknown"
}

func ghCheckState(c ghCheck) string {
	if c.TypeName == "CheckRun" {
		switch c.Conclusion {
		case "SUCCESS":
			return "pass"
		case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE", "CANCELLED", "ACTION_REQUIRED":
			return "fail"
		case "NEUTRAL", "SKIPPED":
			return "neutral"
		case "":
			return "pending" // status QUEUED/IN_PROGRESS 등 계산 중
		}
		return "unknown"
	}
	switch c.State {
	case "SUCCESS", "EXPECTED":
		return "pass"
	case "FAILURE", "ERROR":
		return "fail"
	case "PENDING":
		return "pending"
	}
	return "unknown"
}

func ghCIRollup(checks []ghCheck) string {
	if len(checks) == 0 {
		return "none"
	}
	overall := ""
	for _, c := range checks {
		st := ghCheckState(c)
		switch st {
		case "fail":
			return "fail"
		case "pending":
			return "pending"
		case "unknown":
			overall = "unknown"
		case "pass", "neutral":
			if overall == "" {
				overall = "pass"
			}
		}
	}
	if overall == "" {
		return "unknown"
	}
	return overall
}

func ghConflict(mergeable, state string) string {
	switch mergeable {
	case "CONFLICTING":
		return "yes"
	case "MERGEABLE":
		return "no"
	}
	if state == "DIRTY" {
		return "yes"
	}
	return "unknown"
}

func ghMergeable(mergeable, state string) string {
	switch mergeable {
	case "MERGEABLE":
		return "yes"
	case "CONFLICTING":
		return "no"
	}
	switch state {
	case "CLEAN":
		return "yes"
	case "DIRTY":
		return "no"
	}
	return "unknown"
}

func parseGLabStatus(data []byte) (prStatus, error) {
	var v struct {
		Draft               *bool              `json:"draft"`
		ApprovedBy          *[]json.RawMessage `json:"approved_by"`
		HasConflicts        *bool              `json:"has_conflicts"`
		MergeStatus         string             `json:"merge_status"`
		DetailedMergeStatus string             `json:"detailed_merge_status"`
		HeadPipeline        *struct {
			Status string `json:"status"`
		} `json:"head_pipeline"`
		Pipeline *struct {
			Status string `json:"status"`
		} `json:"pipeline"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return prStatus{}, fmt.Errorf("cannot parse glab mr view output: %v", err)
	}
	draft := "unknown"
	if v.Draft != nil {
		draft = yesNo(*v.Draft)
	}
	approval := "unknown"
	if v.ApprovedBy != nil {
		if len(*v.ApprovedBy) > 0 {
			approval = "approved"
		} else {
			approval = "required"
		}
	}
	ci := "none"
	pipe := v.HeadPipeline
	if pipe == nil {
		pipe = v.Pipeline
	}
	if pipe != nil {
		ci = glabCIStatus(pipe.Status)
	}
	conflict := "unknown"
	if v.HasConflicts != nil {
		conflict = yesNo(*v.HasConflicts)
	}
	return prStatus{
		Draft:     draft,
		Approval:  approval,
		CI:        ci,
		Conflict:  conflict,
		Mergeable: glabMergeable(v.DetailedMergeStatus, v.MergeStatus),
	}, nil
}

func glabCIStatus(status string) string {
	switch status {
	case "success":
		return "pass"
	case "failed", "canceled":
		return "fail"
	case "created", "waiting_for_resource", "preparing", "pending", "running", "scheduled":
		return "pending"
	}
	return "unknown"
}

func glabMergeable(detailed, mergeStatus string) string {
	switch detailed {
	case "mergeable", "can_be_merged":
		return "yes"
	case "conflicts":
		return "no"
	}
	switch mergeStatus {
	case "can_be_merged":
		return "yes"
	case "cannot_be_merged":
		return "no"
	}
	return "unknown"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
