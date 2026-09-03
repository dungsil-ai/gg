package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeStatusBin은 argv를 LOG에 기록하고 고정 JSON을 stdout으로 내는 fake provider CLI다.
func writeFakeStatusBin(t *testing.T, dir, name, logFile, jsonOut string) {
	t.Helper()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name+".cmd")
		body = "@echo off\r\necho " + name + " %* >> \"" + logFile + "\"\r\necho " + jsonOut + "\r\nexit /b 0\r\n"
	} else {
		path = filepath.Join(dir, name)
		body = "#!/bin/sh\necho \"" + name + " $@\" >> \"" + logFile + "\"\nprintf '%s\\n' '" + jsonOut + "'\nexit 0\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestE2EPRStatusGitHubShowsCommonFieldsInOrder(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeStatusBin(t, fakeDir, "gh", logFile,
		`{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[{"__typename":"StatusContext","state":"SUCCESS"}]}`)
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "42")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := "Draft: no\nApproval: approved\nCI: pass\nConflict: no\nMergeable: yes\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if got := readLog(t, logFile); got != "gh pr view 42 -R github.com/o/r --json isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup" {
		t.Errorf("gh argv = %q", got)
	}
}

func TestE2EPRStatusGitHubBadValuesStillExitZero(t *testing.T) {
	bin := buildGG(t)
	scenarios := []struct {
		name string
		json string
		want string
	}{
		{"conflict", `{"isDraft":false,"reviewDecision":"CHANGES_REQUESTED","mergeable":"CONFLICTING","mergeStateStatus":"DIRTY","statusCheckRollup":[{"__typename":"CheckRun","status":"COMPLETED","conclusion":"FAILURE"}]}`,
			"Draft: no\nApproval: changes-requested\nCI: fail\nConflict: yes\nMergeable: no\n"},
		{"computing", `{"isDraft":false,"reviewDecision":"","mergeable":"UNKNOWN","mergeStateStatus":"UNKNOWN","statusCheckRollup":[{"__typename":"StatusContext","state":"PENDING"}]}`,
			"Draft: no\nApproval: required\nCI: pending\nConflict: unknown\nMergeable: unknown\n"},
		{"no ci", `{"isDraft":true,"reviewDecision":"REVIEW_REQUIRED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[]}`,
			"Draft: yes\nApproval: required\nCI: none\nConflict: no\nMergeable: yes\n"},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			writeFakeStatusBin(t, fakeDir, "gh", filepath.Join(t.TempDir(), "calls.log"), tc.json)
			repo := tempRepo(t, "https://github.com/o/r.git")
			out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "42")
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if out != tc.want {
				t.Fatalf("stdout = %q, want %q", out, tc.want)
			}
		})
	}
}

func TestE2EPRStatusGitLabShowsCommonFields(t *testing.T) {
	bin := buildGG(t)
	scenarios := []struct {
		name string
		json string
		want string
	}{
		{"ready", `{"draft":false,"approved_by":[{"user":{"username":"a"}}],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable","head_pipeline":{"status":"success"}}`,
			"Draft: no\nApproval: approved\nCI: pass\nConflict: no\nMergeable: yes\n"},
		{"pending conflict", `{"draft":false,"approved_by":[],"has_conflicts":true,"merge_status":"cannot_be_merged","detailed_merge_status":"conflicts","head_pipeline":{"status":"running"}}`,
			"Draft: no\nApproval: required\nCI: pending\nConflict: yes\nMergeable: no\n"},
		{"unknown draft", `{"approved_by":[],"has_conflicts":false,"merge_status":"can_be_merged","detailed_merge_status":"mergeable"}`,
			"Draft: unknown\nApproval: required\nCI: none\nConflict: no\nMergeable: yes\n"},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			logFile := filepath.Join(t.TempDir(), "calls.log")
			writeFakeStatusBin(t, fakeDir, "glab", logFile, tc.json)
			repo := tempRepo(t, "https://gitlab.com/o/r.git")
			out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "7")
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if out != tc.want {
				t.Fatalf("stdout = %q, want %q", out, tc.want)
			}
			if got := readLog(t, logFile); got != "glab mr view 7 --output json --repo https://gitlab.com/o/r" {
				t.Errorf("glab argv = %q", got)
			}
		})
	}
}

func TestE2EPRStatusRepositoryContextFlags(t *testing.T) {
	bin := buildGG(t)
	jsonOut := `{"isDraft":false,"reviewDecision":"APPROVED","mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","statusCheckRollup":[]}`

	t.Run("repo flag", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeStatusBin(t, fakeDir, "gh", logFile, jsonOut)
		out, code := runGG(t, bin, fakeDir, t.TempDir(),
			"--repo", "https://github.com/custom/repo", "pr", "status", "5")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); !strings.Contains(got, "-R github.com/custom/repo") {
			t.Errorf("gh argv = %q", got)
		}
	})

	t.Run("remote flag", func(t *testing.T) {
		fakeDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		writeFakeStatusBin(t, fakeDir, "gh", logFile, jsonOut)
		repo := tempRepoWithUpstream(t)
		out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "5", "--remote", "upstream")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if got := readLog(t, logFile); !strings.Contains(got, "-R github.com/o/upstream") {
			t.Errorf("gh argv = %q", got)
		}
	})
}

func TestE2EPRStatusBadJSONFails(t *testing.T) {
	for _, tc := range []struct {
		name, bin, fake, json string
	}{
		{"github", "gh", "gh", "not json at all"},
		{"gitlab", "glab", "glab", "this is not json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := buildGG(t)
			fakeDir := t.TempDir()
			writeFakeStatusBin(t, fakeDir, tc.fake, filepath.Join(t.TempDir(), "calls.log"), tc.json)
			var repo string
			if tc.name == "github" {
				repo = tempRepo(t, "https://github.com/o/r.git")
			} else {
				repo = tempRepo(t, "https://gitlab.com/o/r.git")
			}
			out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "42")
			if code == 0 {
				t.Fatalf("bad JSON should fail: %s", out)
			}
			if !strings.Contains(out, "cannot parse") {
				t.Errorf("output = %s", out)
			}
		})
	}
}

func TestE2EPRStatusProviderFailureFails(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	var path, body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(fakeDir, "gh.cmd")
		body = "@echo off\r\necho gh: not found\r\nexit /b 1\r\n"
	} else {
		path = filepath.Join(fakeDir, "gh")
		body = "#!/bin/sh\necho 'gh: not found'\nexit 1\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := tempRepo(t, "https://github.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "42")
	if code == 0 {
		t.Fatalf("provider failure should be non-zero: %s", out)
	}
	if !strings.Contains(out, "gh failed with exit code 1") {
		t.Errorf("output = %s", out)
	}
}

func TestE2EPRStatusMissingNumberIsUsageError(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	writeFakeStatusBin(t, fakeDir, "gh", filepath.Join(t.TempDir(), "calls.log"), `{}`)
	out, code := runGG(t, bin, fakeDir, t.TempDir(), "pr", "status")
	if code != 2 || !strings.Contains(out, "usage: gg pr status <number>") {
		t.Fatalf("exit %d, output %s", code, out)
	}
}

func TestE2EPRStatusTeaUnsupported(t *testing.T) {
	bin := buildGG(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "calls.log")
	writeFakeBin(t, fakeDir, "tea", logFile)
	repo := tempRepo(t, "https://gitea.com/o/r.git")

	out, code := runGG(t, bin, fakeDir, repo, "pr", "status", "42")
	if code != 2 || !strings.Contains(out, "pr status is not supported for tea") {
		t.Fatalf("exit %d, output %s", code, out)
	}
	if got := readLog(t, logFile); got != "" {
		t.Errorf("tea should not run, got %q", got)
	}
}
