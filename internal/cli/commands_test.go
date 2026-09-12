package cli

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// 기대값은 registry나 help 생성 함수에서 가져오지 않는다. 명령 이름과
// 사용법, flag 값의 공개 계약만 고정하고 설명 문구와 행 순서는 고정하지 않는다.
type helpContract struct {
	args  string
	flags string
}

const forgeHelpFlags = "--repo <URL> --remote <name> --explain --help"

var actionHelpContracts = map[string]helpContract{
	"repo list":         {"[flags]", "--limit <N> " + forgeHelpFlags},
	"repo view":         {"[flags]", forgeHelpFlags},
	"repo create":       {"[flags]", "--repo <URL> --description <text> --public --private --explain --help"},
	"repo clone":        {"<URL> [DIR] [flags]", "--allow-insecure-http --explain --help"},
	"repo fork":         {"[flags]", forgeHelpFlags},
	"repo contributors": {"[flags]", forgeHelpFlags},
	"repo mirror":       {"--url <url> [flags]", "--url <url> " + forgeHelpFlags},
	"repo search":       {"--search <text> [flags]", "--search <text> " + forgeHelpFlags},
	"repo transfer":     {"--target-namespace <namespace> [flags]", "--target-namespace <namespace> --yes " + forgeHelpFlags},
	"repo delete":       {"[flags]", "--yes " + forgeHelpFlags},
	"repo edit":         {"[flags]", "--description <text> --public --private " + forgeHelpFlags},
	"repo rename":       {"[<new-name>] [flags]", "--yes " + forgeHelpFlags},
	"repo sync":         {"[flags]", "--branch <branch> --source <repository> --force " + forgeHelpFlags},
	"repo set-default":  {"[flags]", "--unset --view --repo <URL> --remote <name> --help"},
	"repo commit":       {"[git args]", "--help"},
	"repo pull":         {"[git args]", "--help"},
	"repo push":         {"[git args]", "--help"},
	"repo filter-repo":  {"[flags]", "--path <path> --invert-paths --path-rename <old:new> --replace-text <regex==>replacement|file> --mailmap <file> --force --dry-run --help"},

	"issue list":           {"[flags]", "--state <open|closed|all> --limit <N> " + forgeHelpFlags},
	"issue status":         {"[flags]", forgeHelpFlags},
	"issue view":           {"<number> [flags]", forgeHelpFlags},
	"issue create":         {"[flags]", "--title <text> --body <text> " + forgeHelpFlags},
	"issue edit":           {"<number> [flags]", "--title <text> --body <text> " + forgeHelpFlags},
	"issue comment":        {"<number> [flags]", "--body <text> " + forgeHelpFlags},
	"issue comment list":   {"<number> [flags]", forgeHelpFlags},
	"issue comment edit":   {"<number> <comment-id> [flags]", "--body <text> " + forgeHelpFlags},
	"issue comment delete": {"<number> <comment-id> [flags]", forgeHelpFlags},
	"issue close":          {"<number> [flags]", forgeHelpFlags},
	"issue reopen":         {"<number> [flags]", forgeHelpFlags},
	"issue pin":            {"<number> [flags]", forgeHelpFlags},
	"issue unpin":          {"<number> [flags]", forgeHelpFlags},
	"issue delete":         {"<number> [flags]", "--yes " + forgeHelpFlags},
	"issue lock":           {"<number> [flags]", "--reason <reason> " + forgeHelpFlags},
	"issue unlock":         {"<number> [flags]", forgeHelpFlags},
	"issue develop":        {"<number> [--list | --name <branch>] [flags]", "--list --name <branch> --base <branch> --checkout " + forgeHelpFlags},
	"issue transfer":       {"<number> <destination-repository> [flags]", forgeHelpFlags},
	"issue sub-issue":      {"<number> --parent <parent> [flags]", "--parent <number> " + forgeHelpFlags},
	"issue blocked-by":     {"<number> --blocker <blocker> [flags]", "--blocker <number> " + forgeHelpFlags},
	"issue type":           {"<number> --name <name> [flags]", "--name <name> " + forgeHelpFlags},
	"issue subscribe":      {"<number> [flags]", forgeHelpFlags},
	"issue unsubscribe":    {"<number> [flags]", forgeHelpFlags},

	"pr list":           {"[flags]", "--state <open|closed|all> --limit <N> " + forgeHelpFlags},
	"pr view":           {"<number> [flags]", forgeHelpFlags},
	"pr checkout":       {"<number> [flags]", forgeHelpFlags},
	"pr checks":         {"<number> [flags]", forgeHelpFlags},
	"pr update-branch":  {"<number> [flags]", forgeHelpFlags},
	"pr diff":           {"<number> [flags]", forgeHelpFlags},
	"pr create":         {"[flags]", "--title <text> --body <text> --base <branch> --head <branch> --draft " + forgeHelpFlags},
	"pr edit":           {"<number> [flags]", "--title <text> --body <text> " + forgeHelpFlags},
	"pr comment":        {"<number> [flags]", "--body <text> " + forgeHelpFlags},
	"pr comment list":   {"<number> [flags]", forgeHelpFlags},
	"pr comment edit":   {"<number> <comment-id> [flags]", "--body <text> " + forgeHelpFlags},
	"pr comment delete": {"<number> <comment-id> [flags]", forgeHelpFlags},
	"pr status":         {"<number> [flags]", forgeHelpFlags},
	"pr ready":          {"<number> [flags]", "--undo " + forgeHelpFlags},
	"pr merge":          {"<number> [flags]", "--merge --squash --rebase --delete-branch --auto " + forgeHelpFlags},
	"pr close":          {"<number> [flags]", forgeHelpFlags},
	"pr delete":         {"<number> [flags]", forgeHelpFlags},
	"pr clean":          {"<number> [flags]", forgeHelpFlags},
	"pr reopen":         {"<number> [flags]", forgeHelpFlags},
	"pr lock":           {"<number> [flags]", "--reason <reason> " + forgeHelpFlags},
	"pr unlock":         {"<number> [flags]", forgeHelpFlags},
	"pr rebase":         {"<number> [flags]", "--skip-ci " + forgeHelpFlags},
	"pr review":         {"<number> (--approve | --request-changes | --comment) [flags]", "--approve --request-changes --comment --body <text> " + forgeHelpFlags},
	"pr approvers":      {"<number> [flags]", forgeHelpFlags},
	"pr revoke":         {"<number> [flags]", forgeHelpFlags},
	"pr todo":           {"<number> [flags]", forgeHelpFlags},
	"pr subscribe":      {"<number> [flags]", forgeHelpFlags},
	"pr unsubscribe":    {"<number> [flags]", forgeHelpFlags},

	"label clone":  {"<source-repository> [flags]", "--force " + forgeHelpFlags},
	"label list":   {"[flags]", "--limit <N> " + forgeHelpFlags},
	"label create": {"[flags]", "--name <text> --color <hex> --description <text> " + forgeHelpFlags},
	"label edit":   {"<name> [flags]", "--name <text> --color <hex> --description <text> " + forgeHelpFlags},
	"label delete": {"<name> [flags]", "--yes " + forgeHelpFlags},

	"release list":         {"[flags]", "--limit <N> " + forgeHelpFlags},
	"release view":         {"[<tag>] [flags]", forgeHelpFlags},
	"release create":       {"<tag> [asset...] [flags]", "--title <text> --notes <text> --ref <ref> --draft --prerelease " + forgeHelpFlags},
	"release edit":         {"<tag> [flags]", "--title <text> --notes <text> --draft --prerelease " + forgeHelpFlags},
	"release delete":       {"<tag> [flags]", "--yes --cleanup-tag " + forgeHelpFlags},
	"release download":     {"[<tag>] [flags]", "--pattern <glob> --dir <dir> " + forgeHelpFlags},
	"release upload":       {"<tag> <asset>... [flags]", forgeHelpFlags},
	"release delete-asset": {"<tag> <asset> [flags]", "--yes " + forgeHelpFlags},

	"ci list":     {"[flags]", "--limit <N> --branch <branch> " + forgeHelpFlags},
	"ci view":     {"[<id>] [flags]", forgeHelpFlags},
	"ci watch":    {"[<id>] [flags]", forgeHelpFlags},
	"ci retry":    {"<id> [flags]", forgeHelpFlags},
	"ci cancel":   {"<id> [flags]", forgeHelpFlags},
	"ci delete":   {"<id> [flags]", forgeHelpFlags},
	"ci download": {"<id> [flags]", "--pattern <glob> --dir <dir> " + forgeHelpFlags},
	"ci lint":     {"[flags]", forgeHelpFlags},
	"ci run":      {"[--branch <branch>] [flags]", "--branch <branch> " + forgeHelpFlags},
	"ci status":   {"[--branch <branch>] [flags]", "--branch <branch> " + forgeHelpFlags},
	"ci trigger":  {"<job-id> [flags]", forgeHelpFlags},

	"workflow list":    {"[flags]", forgeHelpFlags},
	"workflow view":    {"<name-or-id> [flags]", "--ref <ref> " + forgeHelpFlags},
	"workflow run":     {"<name-or-id> [flags]", "--ref <ref> " + forgeHelpFlags},
	"workflow enable":  {"<name-or-id> [flags]", forgeHelpFlags},
	"workflow disable": {"<name-or-id> [flags]", forgeHelpFlags},
	"auth status":      {"", "--help"},
	"config list":      {"", "--help"},
	"config set":       {"<host> <gh|glab|tea>", "--help"},
	"config unset":     {"<host>", "--help"},
}

var resourceHelpContracts = map[string]helpContract{
	"repo":     {"<command> [args]", "--help"},
	"issue":    {"<command> [flags]", forgeHelpFlags},
	"pr":       {"<command> [flags]", forgeHelpFlags},
	"label":    {"<command> [flags]", forgeHelpFlags},
	"release":  {"<command> [flags]", forgeHelpFlags},
	"ci":       {"<command> [flags]", forgeHelpFlags},
	"workflow": {"<command> [flags]", forgeHelpFlags},
	"auth":     {"<command>", "--help"},
	"config":   {"<command>", "--help"},
}

// 이 목록도 제품의 passthrough registry와 별개로 유지한다. 도움말을 외부
// CLI에 전달하는 명령도 resource의 Commands 목록에는 모두 나타나야 한다.
var relayedHelpActions = map[string][]string{
	"auth": strings.Fields("login logout refresh setup-git switch token"),
	"repo": strings.Fields(`
		add am archive bisect branch bundle checkout cherry-pick citool clean describe diff fetch format-patch gc grep gui
		init log merge mv notes range-diff rebase reset restore revert rm shortlog show sparse-checkout stash status submodule switch tag worktree
		annotate blame bugreport count-objects diagnose difftool fsck instaweb maintenance merge-tree mergetool prune-packed rerere scalar
		archimport cvsexportcommit cvsimport cvsserver imap-send p4 quiltimport request-pull send-email svn
		apply cat-file check-attr check-ignore check-mailmap check-ref-format checkout-index column commit-graph commit-tree credential credential-cache
		credential-store daemon diff-files diff-index diff-tree fast-export fast-import fetch-pack for-each-ref for-each-repo hash-object http-backend
		http-fetch http-push index-pack ls-files ls-remote ls-tree mailinfo mailsplit merge-base merge-file merge-index mktag mktree multi-pack-index
		name-rev pack-objects pack-redundant pack-refs patch-id prune read-tree receive-pack reflog remote repack replace rev-list rev-parse send-pack
		show-branch show-index show-ref stripspace symbolic-ref unpack-file unpack-objects update-index update-ref update-server-info upload-archive
		upload-pack var verify-commit verify-pack verify-tag write-tree`),
}

func helpSectionLines(t *testing.T, help, section string) []string {
	t.Helper()
	found := false
	var lines []string
	for _, line := range strings.Split(help, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == section+":" {
			found = true
			continue
		}
		if !found || trimmed == "" {
			continue
		}
		if strings.TrimLeft(line, " \t") == line {
			break
		}
		lines = append(lines, trimmed)
	}
	if !found {
		t.Fatalf("help에 %s 섹션 없음:\n%s", section, help)
	}
	return lines
}

var helpColumnSeparator = regexp.MustCompile(`\s{2,}|\t`)

func assertHelpLabels(t *testing.T, help, section string, want []string) {
	t.Helper()
	var got []string
	for _, line := range helpSectionLines(t, help, section) {
		label := helpColumnSeparator.Split(line, 2)[0]
		got = append(got, strings.Join(strings.Fields(label), " "))
	}
	expected := slices.Clone(want)
	slices.Sort(got)
	slices.Sort(expected)
	if !slices.Equal(got, expected) {
		t.Errorf("help %s = %q, want %q", section, got, expected)
	}
}

func assertHelpContract(t *testing.T, help, path string, contract helpContract) {
	t.Helper()
	wantUsage := strings.TrimSpace("gg " + path + " " + contract.args)
	var usages []string
	for _, line := range helpSectionLines(t, help, "Usage") {
		usages = append(usages, strings.Join(strings.Fields(line), " "))
	}
	if !slices.Contains(usages, wantUsage) {
		t.Errorf("help usage = %q, want %q", usages, wantUsage)
	}
	var flags []string
	for _, part := range strings.Split(contract.flags, "--") {
		if part = strings.TrimSpace(part); part != "" {
			flags = append(flags, "--"+part)
		}
	}
	assertHelpLabels(t, help, "Flags", flags)
}

func assertResourceHelpContract(t *testing.T, help, resource string, contract helpContract) {
	t.Helper()
	assertHelpContract(t, help, resource, contract)
	commands := slices.Clone(relayedHelpActions[resource])
	for path := range actionHelpContracts {
		if name, ok := strings.CutPrefix(path, resource+" "); ok {
			commands = append(commands, name)
		}
	}
	assertHelpLabels(t, help, "Commands", commands)
}

func TestActionHelpShowsOnlyDefinedFlags(t *testing.T) {
	// 신규 명령이 추가되어도 계약 검사를 조용히 건너뛰지 않는다.
	for resource, rd := range commandDefs {
		if _, ok := resourceHelpContracts[resource]; !ok {
			t.Errorf("resource %s의 독립적인 도움말 계약이 없음", resource)
		}
		for _, action := range rd.actions {
			path := resource + " " + action.name
			if _, ok := actionHelpContracts[path]; !ok && !slices.Contains(relayedHelpActions[resource], action.name) {
				t.Errorf("action %s의 독립적인 도움말 계약이 없음", path)
			}
		}
	}
	for path, contract := range actionHelpContracts {
		t.Run(path, func(t *testing.T) {
			help, ok := nestedHelp(append(strings.Fields(path), "--help"))
			if !ok {
				t.Fatal("명령 도움말을 찾지 못함")
			}
			assertHelpContract(t, help, path, contract)
		})
	}
}

func TestResourceHelpSharedFlags(t *testing.T) {
	for resource, contract := range resourceHelpContracts {
		t.Run(resource, func(t *testing.T) {
			help, ok := nestedHelp([]string{resource, "--help"})
			if !ok {
				t.Fatal("resource 도움말을 찾지 못함")
			}
			assertResourceHelpContract(t, help, resource, contract)
		})
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
	// repo resource는 가변 Git 인자를 명시하고, Git passthrough은 --help까지 Git에 전달한다.
	if help, ok := nestedHelp([]string{"repo", "--help"}); !ok || !strings.Contains(help, "gg repo <command> [args]") {
		t.Errorf("nestedHelp(repo --help) = %q, %v", help, ok)
	}
	for _, name := range commandOrder {
		rd := commandDefs[name]
		for i := range rd.actions {
			ad := &rd.actions[i]
			path := append(append([]string{name}, strings.Fields(ad.name)...), "--help")
			help, ok := nestedHelp(path)
			if name == "repo" && isGitPassthroughAction(ad.name) {
				if ok {
					t.Errorf("nestedHelp(%v)가 git passthrough help를 가로챔: %q", path, help)
				}
				continue
			}
			if name == "auth" && isAuthRelayAction(ad.name) {
				if ok {
					t.Errorf("nestedHelp(%v)가 gh auth relay help를 가로챔: %q", path, help)
				}
				continue
			}
			if !ok || !strings.Contains(help, "Usage:") || !strings.Contains(help, "gg "+name+" "+ad.name) {
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
	// help 경로가 아닌 입력 (commit 및 보조 git 명령 생략형은 --help를 git에 전달한다)
	for _, action := range commandDefs["repo"].actions {
		if action.passthrough && !helpAliases[action.name] {
			path := []string{action.name, "--help"}
			if help, ok := nestedHelp(path); ok {
				t.Errorf("nestedHelp(%v)가 help를 반환함: %q", path, help)
			}
		}
	}
	for _, path := range [][]string{
		{"unknown", "--help"},
		{"issue", "freeze", "--help"},
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
		"gg [flags] <command>", "gg <supported-git-command> [git args]", "gg repo --help",
		"gg config --help", "gg issue --help", "gg issue list --help", "gg pr create --help", "gg pr ready --help", "gg pr merge --help", "gg pr close --help", "gg pr reopen --help",
		"--repo <URL>", "--remote <name>", "--explain", "-h, --help", "--version",
		"-v, -verison",
	}
	commands := append(slices.Clone(relayedHelpActions["repo"]), "commit", "pull", "push", "version", "help")
	for resource := range resourceHelpContracts {
		commands = append(commands, resource)
	}
	assertHelpLabels(t, help, "Commands", commands)
	for _, want := range wants {
		if !strings.Contains(help, want) {
			t.Errorf("topLevelHelp에 %q 없음:\n%s", want, help)
		}
	}
	if strings.Contains(help, "gg <command> --help") {
		t.Errorf("topLevelHelp가 일반형 help를 광고함:\n%s", help)
	}
}
