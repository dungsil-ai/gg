package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EUsageErrorsDoNotStartChildren(t *testing.T) {
	bin := buildGG(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"unknown"}, "unknown command unknown"},
		{[]string{"config"}, "config needs an action: list, set, unset"},
		{[]string{"issue"}, "issue needs an action: list, status, view, create, edit, comment, comment list, comment edit, comment delete, close, reopen, pin, unpin, delete, lock, unlock, develop, transfer, sub-issue, blocked-by, type, subscribe, unsubscribe"},
		{[]string{"label"}, "label needs an action: clone, list, create, edit, delete"},
		{[]string{"pr"}, "pr needs an action: list, view, checkout, checks, update-branch, diff, create, edit, comment, comment list, comment edit, comment delete, status, ready, merge, close, delete, clean, reopen, lock, unlock, rebase, review, approvers, revoke, todo, subscribe, unsubscribe"},
		{[]string{"repo"}, "repo needs an action: list, view, create, clone, fork, contributors, mirror, search, transfer, delete, edit, rename, sync, set-default, commit, pull, push, filter-repo, add, am, archive, bisect, branch, bundle, checkout, cherry-pick, citool, clean, describe, diff, fetch, format-patch, gc, grep, gui, init, log, merge, mv, notes, range-diff, rebase, reset, restore, revert, rm, shortlog, show, sparse-checkout, stash, status, submodule, switch, tag, worktree, annotate, blame, bugreport, count-objects, diagnose, difftool, fsck, instaweb, maintenance, merge-tree, mergetool, prune-packed, rerere, scalar"},
		{[]string{"issue", "freeze", "1"}, "issue does not support freeze"},
		{[]string{"label", "edit", "bug"}, "label edit needs --name, --color, or --description"},
		{[]string{"label", "edit"}, "usage: gg label edit <name>"},
		{[]string{"label", "edit", "bug", "extra"}, "usage: gg label edit <name>"},
		{[]string{"label", "delete"}, "usage: gg label delete <name>"},
		{[]string{"label", "delete", "1", "2"}, "usage: gg label delete <name>"},
		{[]string{"issue", "list", "--wat"}, "unknown flag --wat"},
		{[]string{"issue", "view"}, "usage: gg issue view <number>"},
		{[]string{"label", "create"}, "usage: gg label create --name <text>"},
		{[]string{"label", "create", "extra"}, "unexpected argument extra"},
		{[]string{"pr", "view"}, "usage: gg pr view <number>"},
		{[]string{"issue", "close"}, "usage: gg issue close <number>"},
		{[]string{"issue", "comment", "1"}, "usage: gg issue comment <number> --body <text>"},
		{[]string{"issue", "comment", "list"}, "usage: gg issue comment list <number>"},
		{[]string{"issue", "comment", "list", "1", "2"}, "usage: gg issue comment list <number>"},
		{[]string{"issue", "comment", "edit", "1"}, "usage: gg issue comment edit <number> <comment-id> --body <text>"},
		{[]string{"issue", "comment", "edit", "1", "2"}, "usage: gg issue comment edit <number> <comment-id> --body <text>"},
		{[]string{"issue", "comment", "delete", "1"}, "usage: gg issue comment delete <number> <comment-id>"},
		{[]string{"pr", "comment", "1"}, "usage: gg pr comment <number> --body <text>"},
		{[]string{"pr", "comment", "list"}, "usage: gg pr comment list <number>"},
		{[]string{"pr", "comment", "list", "1", "2"}, "usage: gg pr comment list <number>"},
		{[]string{"pr", "comment", "edit", "1"}, "usage: gg pr comment edit <number> <comment-id> --body <text>"},
		{[]string{"pr", "comment", "edit", "1", "2"}, "usage: gg pr comment edit <number> <comment-id> --body <text>"},
		{[]string{"pr", "comment", "delete", "1"}, "usage: gg pr comment delete <number> <comment-id>"},
		{[]string{"pr", "merge"}, "usage: gg pr merge <number>"},
		{[]string{"pr", "close"}, "usage: gg pr close <number>"},
		{[]string{"pr", "close", "1", "2"}, "usage: gg pr close <number>"},
		{[]string{"pr", "reopen"}, "usage: gg pr reopen <number>"},
		{[]string{"pr", "merge", "1", "--merge", "--squash"}, "--merge, --squash, --rebase are mutually exclusive; use at most one"},
		{[]string{"clone", "https://x.com/o/r", "d", "x"}, "usage: gg clone <URL|namespace/name> [DIR]"},
		{[]string{"create", "--public"}, "repo create needs --repo <new-repository-URL>"},
		{[]string{"create", "--repo", "https://x.com/o/r"}, "repo create needs exactly one of --public or --private"},
		{[]string{"list", "extra"}, "unexpected argument extra"},
		{[]string{"config", "list", "extra"}, "usage: gg config list"},
		{[]string{"config", "set", "only-host"}, "usage: gg config set <host> <provider>"},
		{[]string{"config", "unset"}, "usage: gg config unset <host>"},
		{[]string{"issue", "list", "--state", "merged"}, "--state must be open, closed, or all"},
		{[]string{"pr", "create", "--title"}, "--title needs a value"},
		{[]string{"--remote", "upstream", "clone", "https://github.com/o/r"}, "--remote is not supported for repo clone"},
		{[]string{"--explain", "pull"}, "--explain is not supported for repo pull"},
		{[]string{"--explain", "config", "list"}, "--explain is not supported for config list"},
		{[]string{"config", "list", "--remote", "origin"}, "--remote is not supported for config list"},
		{[]string{"config", "list", "--repo", "https://github.com/o/r"}, "--repo is not supported for config list"},
		{[]string{"--repo", "https://github.com/o/r", "config", "set", "git.example.com", "tea"}, "--repo is not supported for config set"},
		{[]string{"--repo", "https://github.com/o/r", "--remote", "upstream", "issue", "list"}, "--repo and --remote cannot be used together"},
		{[]string{"repo", "clone", "https://github.com/a/b", "--repo", "https://github.com/x/y"}, "--repo is not supported for repo clone"},
		{[]string{"--repo", "https://github.com/x/y", "clone", "https://github.com/a/b"}, "--repo is not supported for repo clone"},
		{[]string{"--repo", "https://github.com/b/b", "repo", "sync"}, "--repo/--remote are not supported for repo sync without --source"},
		{[]string{"repo", "sync", "--force", "--remote", "upstream"}, "--repo/--remote are not supported for repo sync without --source"},
		{[]string{"--repo"}, "--repo needs a URL"},
		{[]string{"--remote"}, "--remote needs a name"},
		{[]string{"release"}, "release needs an action: list, view, create, edit, delete, prepare, download, upload, delete-asset"},
		{[]string{"release", "publish"}, "release does not support publish"},
		{[]string{"release", "view", "a", "b"}, "usage: gg release view [<tag>]"},
		{[]string{"release", "create"}, "usage: gg release create <tag> [asset...]"},
		{[]string{"release", "edit"}, "usage: gg release edit <tag>"},
		{[]string{"release", "edit", "a", "b"}, "usage: gg release edit <tag>"},
		{[]string{"release", "edit", "v1.0.0", "--draft=maybe"}, "--draft must be true or false"},
		{[]string{"release", "edit", "v1.0.0", "--prerelease=yes"}, "--prerelease must be true or false"},
		{[]string{"release", "delete"}, "usage: gg release delete <tag>"},
		{[]string{"release", "download", "a", "b"}, "usage: gg release download [<tag>]"},
		{[]string{"release", "upload", "v1.0.0"}, "usage: gg release upload <tag> <asset>..."},
		{[]string{"release", "delete-asset", "v1.0.0"}, "usage: gg release delete-asset <tag> <asset>"},
		{[]string{"release", "delete-asset", "v1.0.0", "a", "b"}, "usage: gg release delete-asset <tag> <asset>"},
		{[]string{"release", "list", "--draft"}, "unknown flag --draft"},
		{[]string{"release", "create", "v1.0.0", "--notes"}, "--notes needs a value"},
		{[]string{"ci"}, "ci needs an action: list, view, watch, retry, cancel, delete, download, lint, run, status, trigger"},
		{[]string{"ci", "list", "extra"}, "unexpected argument extra"},
		{[]string{"ci", "view", "1", "2"}, "usage: gg ci view [<id>]"},
		{[]string{"ci", "watch", "1", "2"}, "usage: gg ci watch [<id>]"},
		{[]string{"ci", "retry"}, "usage: gg ci retry <id>"},
		{[]string{"ci", "retry", "1", "2"}, "usage: gg ci retry <id>"},
		{[]string{"ci", "cancel"}, "usage: gg ci cancel <id>"},
		{[]string{"ci", "list", "--wat"}, "unknown flag --wat"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			fakeDir, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls.log")
			for _, name := range []string{"git", "gh", "glab", "tea"} {
				writeFakeBin(t, fakeDir, name, logFile)
			}
			stdout, stderr, code := runGGStreamsWithFake(t, bin, fakeDir, t.TempDir(), c.args...)
			if code != 2 || stdout != "" || !strings.Contains(stderr, c.want) {
				t.Fatalf("gg %v: exit %d, stdout %q, stderr %q; want exit 2 and %q", c.args, code, stdout, stderr, c.want)
			}
			if calls := readLog(t, logFile); calls != "" {
				t.Fatalf("invalid command started children: %s", calls)
			}
		})
	}
}
