package cli

import "slices"

// 이 파일은 forge 라우팅 없이 Git에 직접 전달할 지원 명령을 정의한다.
// Main Porcelain 37개, ancillary 14개, plumbing 70개를 하나의 registry에서 관리한다.
// clone, commit, pull, push는 cmd_repo.go의 별도 동작을 유지한다.

// gitPassthroughActionNames는 forge 라우팅 없이 Git에 직접 전달할 지원 명령이다.
var gitPassthroughActionNames = []string{
	"add", "am", "archive", "bisect", "branch", "bundle", "checkout", "cherry-pick",
	"citool", "clean", "describe", "diff", "fetch", "format-patch", "gc", "grep", "gui",
	"init", "log", "merge", "mv", "notes", "range-diff", "rebase", "reset", "restore",
	"revert", "rm", "shortlog", "show", "sparse-checkout", "stash", "status", "submodule",
	"switch", "tag", "worktree",
	"annotate", "blame", "bugreport", "count-objects", "diagnose", "difftool", "fsck",
	"instaweb", "maintenance", "merge-tree", "mergetool", "prune-packed", "rerere", "scalar",
	"apply", "cat-file", "check-attr", "check-ignore", "check-mailmap", "check-ref-format",
	"checkout-index", "column", "commit-graph", "commit-tree", "credential", "credential-cache",
	"credential-store", "daemon", "diff-files", "diff-index", "diff-tree", "fast-export",
	"fast-import", "fetch-pack", "for-each-ref", "for-each-repo", "hash-object", "http-backend",
	"http-fetch", "http-push", "index-pack", "ls-files", "ls-remote", "ls-tree", "mailinfo",
	"mailsplit", "merge-base", "merge-file", "merge-index", "mktag", "mktree", "multi-pack-index",
	"name-rev", "pack-objects", "pack-redundant", "pack-refs", "patch-id", "prune", "read-tree",
	"receive-pack", "reflog", "remote", "repack", "replace", "rev-list", "rev-parse", "send-pack",
	"show-branch", "show-index", "show-ref", "stripspace", "symbolic-ref", "unpack-file",
	"unpack-objects", "update-index", "update-ref", "update-server-info", "upload-archive",
	"upload-pack", "var", "verify-commit", "verify-pack", "verify-tag", "write-tree",
}

func isGitPassthroughAction(name string) bool {
	return slices.Contains(gitPassthroughActionNames, name)
}

func gitPassthroughActions() []actionDef {
	actions := make([]actionDef, len(gitPassthroughActionNames))
	for i, name := range gitPassthroughActionNames {
		actions[i] = actionDef{
			name: name, summary: "Run git " + name, usage: "gg " + name + " [git args]",
			passthrough: true, maxPos: -1,
		}
	}
	return actions
}
