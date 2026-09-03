package main

import (
	"strings"
	"testing"
)

// allFlagNames은 gg가 아는 모든 flag 이름을 모은다.
func allFlagNames(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	var names []string
	for _, name := range commandOrder {
		for i := range commandDefs[name].actions {
			for _, f := range commandDefs[name].actions[i].flags {
				if !seen[f.name] {
					seen[f.name] = true
					names = append(names, f.name)
				}
			}
		}
	}
	for _, f := range []flagDef{repoContextFlag, remoteContextFlag, explainFlag, helpFlag} {
		if !seen[f.name] {
			seen[f.name] = true
			names = append(names, f.name)
		}
	}
	return names
}

// TestActionHelpShowsOnlyDefinedFlags는 action help가 정의에 없는 flag을
// 광고하지 않는지, 정의된 flag은 모두 표시하는지 본다.
func TestActionHelpShowsOnlyDefinedFlags(t *testing.T) {
	for _, name := range commandOrder {
		rd := commandDefs[name]
		for i := range rd.actions {
			ad := &rd.actions[i]
			shown := map[string]bool{}
			for _, f := range actionFlags(ad) {
				shown[f.name] = true
			}
			help := renderActionHelp(rd, ad)
			for _, flagName := range allFlagNames(t) {
				if shown[flagName] && !strings.Contains(help, flagName) {
					t.Errorf("gg %s %s --help에 정의된 %q 없음:\n%s", name, ad.name, flagName, help)
				}
				if !shown[flagName] && strings.Contains(help, flagName) {
					t.Errorf("gg %s %s --help가 지원하지 않는 %q를 광고함:\n%s", name, ad.name, flagName, help)
				}
			}
		}
	}
}

// TestResourceHelpSharedFlags는 resource help가 모든 action이 함께
// 지원하는 flag만 표시하는지 본다.
func TestResourceHelpSharedFlags(t *testing.T) {
	for _, name := range commandOrder {
		rd := commandDefs[name]
		shown := map[string]bool{}
		for _, f := range resourceFlags(rd) {
			shown[f.name] = true
		}
		help := renderResourceHelp(rd)
		for _, flagName := range allFlagNames(t) {
			if shown[flagName] && !strings.Contains(help, flagName) {
				t.Errorf("gg %s --help에 정의된 %q 없음:\n%s", name, flagName, help)
			}
			if !shown[flagName] && strings.Contains(help, flagName) {
				t.Errorf("gg %s --help가 일부 action만 지원하는 %q를 광고함:\n%s", name, flagName, help)
			}
		}
		for i := range rd.actions {
			ad := &rd.actions[i]
			for _, want := range []string{ad.name, ad.summary} {
				if !strings.Contains(help, want) {
					t.Errorf("gg %s --help에 %q 없음:\n%s", name, want, help)
				}
			}
		}
	}
}

func TestNestedHelpPaths(t *testing.T) {
	// resource 단계
	for _, name := range commandOrder {
		help, ok := nestedHelp([]string{name, "--help"})
		if !ok || !strings.Contains(help, "Commands:") {
			t.Errorf("nestedHelp(%s --help) = %q, %v", name, help, ok)
		}
	}
	// action 단계: git passthrough은 --help까지 git에 전달한다.
	for _, name := range commandOrder {
		rd := commandDefs[name]
		for i := range rd.actions {
			ad := &rd.actions[i]
			path := []string{name, ad.name, "--help"}
			help, ok := nestedHelp(path)
			if name == "repo" && isGitPassthroughAction(ad.name) {
				if ok {
					t.Errorf("nestedHelp(%v)가 git passthrough help를 가로챔: %q", path, help)
				}
				continue
			}
			if !ok || !strings.Contains(help, ad.usage) {
				t.Errorf("nestedHelp(%v) = %q, %v", path, help, ok)
			}
		}
	}
	// repo 생략 형태는 repo 접두 형태와 같은 help를 내야 한다
	for alias := range helpAliases {
		omitted, ok := nestedHelp([]string{alias, "--help"})
		prefixed, ok2 := nestedHelp([]string{"repo", alias, "--help"})
		if !ok || !ok2 || omitted == "" || omitted != prefixed {
			t.Errorf("alias %s help가 비었거나 다름", alias)
		}
	}
	// help 경로가 아닌 입력
	for _, path := range [][]string{
		{"commit", "--help"}, // commit 생략형은 --help를 git에 전달한다
		{"unknown", "--help"},
		{"issue", "delete", "--help"},
		{"issue", "list", "extra", "--help"},
		{"repo", "list", "extra", "--help"},
		{"--help"},
		{"--repo", "x", "--help"},
	} {
		if help, ok := nestedHelp(path); ok {
			t.Errorf("nestedHelp(%v)가 help를 반환함: %q", path, help)
		}
	}
}

func TestTopLevelHelpContent(t *testing.T) {
	help := topLevelHelp()
	wants := []string{
		"gg config --help", "gg issue --help", "gg issue list --help", "gg pr create --help", "gg pr ready --help", "gg pr merge --help",
		"--repo <URL>", "--remote <name>", "--explain", "-h, --help", "--version",
	}
	for _, name := range commandOrder {
		wants = append(wants, commandDefs[name].summary)
	}
	for _, name := range topLevelAliases {
		wants = append(wants, commandDefs["repo"].action(name).summary)
	}
	for _, want := range wants {
		if !strings.Contains(help, want) {
			t.Errorf("topLevelHelp에 %q 없음:\n%s", want, help)
		}
	}
	if strings.Contains(help, "gg <command> --help") {
		t.Errorf("topLevelHelp가 일반형 help를 광고함:\n%s", help)
	}
}
