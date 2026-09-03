# TODO

## git 기능 목록

### 주요 작업 및 저장소 조작 (Main Porcelain)
- [x] `git add` (대응: `gg repo add`, `gg add`)
- [x] `git am` (대응: `gg repo am`, `gg am`)
- [x] `git archive` (대응: `gg repo archive`, `gg archive`)
- [x] `git bisect` (대응: `gg repo bisect`, `gg bisect`)
- [x] `git branch` (대응: `gg repo branch`, `gg branch`)
- [x] `git bundle` (대응: `gg repo bundle`, `gg bundle`)
- [x] `git checkout` (대응: `gg repo checkout`, `gg checkout`)
- [x] `git cherry-pick` (대응: `gg repo cherry-pick`, `gg cherry-pick`)
- [x] `git citool` (대응: `gg repo citool`, `gg citool`)
- [x] `git clean` (대응: `gg repo clean`, `gg clean`)
- [x] `git clone` (대응: `gg repo clone`, `gg clone`)
- [x] `git commit` (대응: `gg repo commit`, `gg commit`; 커밋 서명 비활성화)
- [x] `git describe` (대응: `gg repo describe`, `gg describe`)
- [x] `git diff` (대응: `gg repo diff`, `gg diff`)
- [x] `git fetch` (대응: `gg repo fetch`, `gg fetch`)
- [x] `git format-patch` (대응: `gg repo format-patch`, `gg format-patch`)
- [x] `git gc` (대응: `gg repo gc`, `gg gc`)
- [x] `git grep` (대응: `gg repo grep`, `gg grep`)
- [x] `git gui` (대응: `gg repo gui`, `gg gui`)
- [x] `git init` (대응: `gg repo init`, `gg init`)
- [x] `git log` (대응: `gg repo log`, `gg log`)
- [x] `git merge` (대응: `gg repo merge`, `gg merge`)
- [x] `git mv` (대응: `gg repo mv`, `gg mv`)
- [x] `git notes` (대응: `gg repo notes`, `gg notes`)
- [x] `git pull` (대응: `gg repo pull`, `gg pull`)
- [x] `git push` (대응: `gg repo push`, `gg push`)
- [x] `git range-diff` (대응: `gg repo range-diff`, `gg range-diff`)
- [x] `git rebase` (대응: `gg repo rebase`, `gg rebase`)
- [x] `git reset` (대응: `gg repo reset`, `gg reset`)
- [x] `git restore` (대응: `gg repo restore`, `gg restore`)
- [x] `git revert` (대응: `gg repo revert`, `gg revert`)
- [x] `git rm` (대응: `gg repo rm`, `gg rm`)
- [x] `git shortlog` (대응: `gg repo shortlog`, `gg shortlog`)
- [x] `git show` (대응: `gg repo show`, `gg show`)
- [x] `git sparse-checkout` (대응: `gg repo sparse-checkout`, `gg sparse-checkout`)
- [x] `git stash` (대응: `gg repo stash`, `gg stash`)
- [x] `git status` (대응: `gg repo status`, `gg status`)
- [x] `git submodule` (대응: `gg repo submodule`, `gg submodule`)
- [x] `git switch` (대응: `gg repo switch`, `gg switch`)
- [x] `git tag` (대응: `gg repo tag`, `gg tag`)
- [x] `git worktree` (대응: `gg repo worktree`, `gg worktree`)

### 보조 명령 및 유틸리티 (Ancillary Commands)
- [x] `git annotate` (대응: `gg repo annotate`, `gg annotate`)
- [x] `git blame` (대응: `gg repo blame`, `gg blame`)
- [x] `git bugreport` (대응: `gg repo bugreport`, `gg bugreport`)
- [x] `git count-objects` (대응: `gg repo count-objects`, `gg count-objects`)
- [x] `git diagnose` (대응: `gg repo diagnose`, `gg diagnose`)
- [x] `git difftool` (대응: `gg repo difftool`, `gg difftool`)
- [x] `git fsck` (대응: `gg repo fsck`, `gg fsck`)
- [x] `git help` (대응: `gg help`, `gg --help`, `gg -h`)
- [x] `git instaweb` (대응: `gg repo instaweb`, `gg instaweb`)
- [x] `git maintenance` (대응: `gg repo maintenance`, `gg maintenance`)
- [x] `git merge-tree` (대응: `gg repo merge-tree`, `gg merge-tree`)
- [x] `git mergetool` (대응: `gg repo mergetool`, `gg mergetool`)
- [x] `git prune-packed` (대응: `gg repo prune-packed`, `gg prune-packed`)
- [x] `git rerere` (대응: `gg repo rerere`, `gg rerere`)
- [x] `git scalar` (대응: `gg repo scalar`, `gg scalar`)
- [x] `git version` (대응: 단독 `gg -verison`, 단독 `gg -v`)

<details>
<summary>외부 시스템 연동 (Interacting with Others)</summary>

- [ ] `git archimport`
- [ ] `git cvsexportcommit`
- [ ] `git cvsimport`
- [ ] `git cvsserver`
- [ ] `git imap-send`
- [ ] `git p4`
- [ ] `git quiltimport`
- [ ] `git request-pull`
- [ ] `git send-email`
- [ ] `git svn`

</details>

<details>
<summary>저수준 제어 (Low-level / Plumbing)</summary>

- [ ] `git apply`
- [ ] `git cat-file`
- [ ] `git check-attr`
- [ ] `git check-ignore`
- [ ] `git check-mailmap`
- [ ] `git check-ref-format`
- [ ] `git checkout-index`
- [ ] `git column`
- [ ] `git commit-graph`
- [ ] `git commit-tree`
- [ ] `git credential`
- [ ] `git credential-cache`
- [ ] `git credential-store`
- [ ] `git daemon`
- [ ] `git diff-files`
- [ ] `git diff-index`
- [ ] `git diff-tree`
- [ ] `git fast-export`
- [ ] `git fast-import`
- [ ] `git fetch-pack`
- [ ] `git for-each-ref`
- [ ] `git for-each-repo`
- [ ] `git hash-object`
- [ ] `git http-backend`
- [ ] `git http-fetch`
- [ ] `git http-push`
- [ ] `git index-pack`
- [ ] `git ls-files`
- [ ] `git ls-remote`
- [ ] `git ls-tree`
- [ ] `git mailinfo`
- [ ] `git mailsplit`
- [ ] `git merge-base`
- [ ] `git merge-file`
- [ ] `git merge-index`
- [ ] `git mktag`
- [ ] `git mktree`
- [ ] `git multi-pack-index`
- [ ] `git name-rev`
- [ ] `git pack-objects`
- [ ] `git pack-redundant`
- [ ] `git pack-refs`
- [ ] `git patch-id`
- [ ] `git prune`
- [ ] `git read-tree`
- [ ] `git receive-pack`
- [ ] `git reflog`
- [ ] `git remote`
- [ ] `git repack`
- [ ] `git replace`
- [ ] `git rev-list`
- [ ] `git rev-parse`
- [ ] `git send-pack`
- [ ] `git show-branch`
- [ ] `git show-index`
- [ ] `git show-ref`
- [ ] `git stripspace`
- [ ] `git symbolic-ref`
- [ ] `git unpack-file`
- [ ] `git unpack-objects`
- [ ] `git update-index`
- [ ] `git update-ref`
- [ ] `git update-server-info`
- [ ] `git upload-archive`
- [ ] `git upload-pack`
- [ ] `git var`
- [ ] `git verify-commit`
- [ ] `git verify-pack`
- [ ] `git verify-tag`
- [ ] `git write-tree`

</details>

---

## gh (GitHub CLI) 기능 목록

### 공통 핵심 명령 (Core Commands)
- `auth`
  - [ ] `gh auth login`
  - [ ] `gh auth logout`
  - [ ] `gh auth refresh`
  - [ ] `gh auth setup-git`
  - [ ] `gh auth status`
  - [ ] `gh auth switch`
  - [ ] `gh auth token`
- `issue`
  - [x] `gh issue close` (대응: `gg issue close`)
  - [x] `gh issue comment` (대응: `gg issue comment`)
  - [x] `gh issue create` (대응: `gg issue create`)
  - [ ] `gh issue delete`
  - [ ] `gh issue develop`
  - [ ] `gh issue edit`
  - [x] `gh issue list` (대응: `gg issue list`)
  - [ ] `gh issue lock`
  - [ ] `gh issue pin`
  - [x] `gh issue reopen` (대응: `gg issue reopen`)
  - [ ] `gh issue status`
  - [ ] `gh issue transfer`
  - [ ] `gh issue unlock`
  - [ ] `gh issue unpin`
  - [x] `gh issue view` (대응: `gg issue view`)
- `label`
  - [ ] `gh label clone`
  - [ ] `gh label create`
  - [ ] `gh label delete`
  - [ ] `gh label edit`
  - [ ] `gh label list`
- `pr`
  - [ ] `gh pr checkout`
  - [ ] `gh pr checks`
  - [ ] `gh pr close`
  - [ ] `gh pr comment`
  - [x] `gh pr create` (대응: `gg pr create`)
  - [ ] `gh pr diff`
  - [ ] `gh pr edit`
  - [x] `gh pr list` (대응: `gg pr list`)
  - [ ] `gh pr lock`
  - [x] `gh pr merge` (대응: `gg pr merge`)
  - [x] `gh pr ready` (대응: `gg pr ready`)
  - [ ] `gh pr reopen`
  - [ ] `gh pr review`
  - [ ] `gh pr status`
  - [ ] `gh pr unlock`
  - [ ] `gh pr update-branch`
  - [x] `gh pr view` (대응: `gg pr view`)
- `release`
  - [ ] `gh release create`
  - [ ] `gh release delete`
  - [ ] `gh release delete-asset`
  - [ ] `gh release download`
  - [ ] `gh release edit`
  - [ ] `gh release list`
  - [ ] `gh release upload`
  - [ ] `gh release view`
- `repo`
  - [ ] `gh repo archive`
  - [ ] `gh repo autolink`
  - [x] `gh repo clone` (대응: `gg repo clone`, `gg clone`)
  - [x] `gh repo create` (대응: `gg repo create`, `gg create`)
  - [ ] `gh repo delete`
  - [ ] `gh repo deploy-key`
  - [ ] `gh repo edit`
  - [ ] `gh repo fork`
  - [x] `gh repo list` (대응: `gg repo list`, `gg list`)
  - [ ] `gh repo rename`
  - [ ] `gh repo set-default`
  - [ ] `gh repo sync`
  - [ ] `gh repo unarchive`
  - [x] `gh repo view` (대응: `gg repo view`, `gg view`)
- [x] `gh help` (대응: `gg help`, `gg --help`, `gg -h`)
- [x] `gh version` (대응: 단독 `gg -verison`, 단독 `gg -v`)

<details>
<summary>기타 및 플랫폼 고유 명령 (GitHub Actions, Codespaces, Project 등)</summary>

- `browse`
  - [ ] `gh browse`
- `cache`
  - [ ] `gh cache delete`
  - [ ] `gh cache list`
- `codespace`
  - [ ] `gh codespace code`
  - [ ] `gh codespace cp`
  - [ ] `gh codespace create`
  - [ ] `gh codespace delete`
  - [ ] `gh codespace edit`
  - [ ] `gh codespace jupyter`
  - [ ] `gh codespace list`
  - [ ] `gh codespace logs`
  - [ ] `gh codespace ports`
  - [ ] `gh codespace rebuild`
  - [ ] `gh codespace ssh`
  - [ ] `gh codespace stop`
  - [ ] `gh codespace view`
- `gist`
  - [ ] `gh gist clone`
  - [ ] `gh gist create`
  - [ ] `gh gist delete`
  - [ ] `gh gist edit`
  - [ ] `gh gist list`
  - [ ] `gh gist rename`
  - [ ] `gh gist view`
- `org`
  - [ ] `gh org list`
- `project`
  - [ ] `gh project close`
  - [ ] `gh project copy`
  - [ ] `gh project create`
  - [ ] `gh project delete`
  - [ ] `gh project edit`
  - [ ] `gh project field-create`
  - [ ] `gh project field-delete`
  - [ ] `gh project field-list`
  - [ ] `gh project item-add`
  - [ ] `gh project item-archive`
  - [ ] `gh project item-create`
  - [ ] `gh project item-delete`
  - [ ] `gh project item-edit`
  - [ ] `gh project item-list`
  - [ ] `gh project link`
  - [ ] `gh project list`
  - [ ] `gh project mark-template`
  - [ ] `gh project unlink`
  - [ ] `gh project view`
- `secret`
  - [ ] `gh secret delete`
  - [ ] `gh secret list`
  - [ ] `gh secret set`
- `variable`
  - [ ] `gh variable delete`
  - [ ] `gh variable get`
  - [ ] `gh variable list`
  - [ ] `gh variable set`
- `run`
  - [ ] `gh run cancel`
  - [ ] `gh run delete`
  - [ ] `gh run download`
  - [ ] `gh run list`
  - [ ] `gh run rerun`
  - [ ] `gh run view`
  - [ ] `gh run watch`
- `workflow`
  - [ ] `gh workflow disable`
  - [ ] `gh workflow enable`
  - [ ] `gh workflow list`
  - [ ] `gh workflow run`
  - [ ] `gh workflow view`
- `alias`
  - [ ] `gh alias delete`
  - [ ] `gh alias import`
  - [ ] `gh alias list`
  - [ ] `gh alias set`
- [ ] `gh api`
- `attestation`
  - [ ] `gh attestation download`
  - [ ] `gh attestation trusted-root`
  - [ ] `gh attestation verify`
- [ ] `gh completion`
- `config`
  - [ ] `gh config clear-cache`
  - [ ] `gh config get`
  - [ ] `gh config list`
  - [ ] `gh config set`
- `extension`
  - [ ] `gh extension browse`
  - [ ] `gh extension create`
  - [ ] `gh extension exec`
  - [ ] `gh extension install`
  - [ ] `gh extension list`
  - [ ] `gh extension remove`
  - [ ] `gh extension search`
  - [ ] `gh extension upgrade`
- `gpg-key`
  - [ ] `gh gpg-key add`
  - [ ] `gh gpg-key delete`
  - [ ] `gh gpg-key list`
- `ruleset`
  - [ ] `gh ruleset check`
  - [ ] `gh ruleset list`
  - [ ] `gh ruleset view`
- `search`
  - [ ] `gh search code`
  - [ ] `gh search commits`
  - [ ] `gh search issues`
  - [ ] `gh search prs`
  - [ ] `gh search repos`
- `ssh-key`
  - [ ] `gh ssh-key add`
  - [ ] `gh ssh-key delete`
  - [ ] `gh ssh-key list`
- [ ] `gh status`

</details>

---

## glab (GitLab CLI) 기능 목록

### 공통 핵심 명령 (Core Commands)
- `auth`
  - [ ] `glab auth login`
  - [ ] `glab auth logout`
  - [ ] `glab auth status`
- `issue`
  - [ ] `glab issue board`
  - [x] `glab issue close` (대응: `gg issue close`)
  - [x] `glab issue create` (대응: `gg issue create`)
  - [ ] `glab issue delete`
  - [x] `glab issue list` (대응: `gg issue list`)
  - [x] `glab issue note` (대응: `gg issue comment`)
  - [x] `glab issue reopen` (대응: `gg issue reopen`)
  - [ ] `glab issue subscribe`
  - [ ] `glab issue todo`
  - [ ] `glab issue unsubscribe`
  - [ ] `glab issue update`
  - [x] `glab issue view` (대응: `gg issue view`)
- `label`
  - [ ] `glab label create`
  - [ ] `glab label list`
- `mr`
  - [ ] `glab mr approve`
  - [ ] `glab mr approvers`
  - [ ] `glab mr checkout`
  - [ ] `glab mr close`
  - [x] `glab mr create` (대응: `gg pr create`)
  - [ ] `glab mr delete`
  - [ ] `glab mr diff`
  - [x] `glab mr list` (대응: `gg pr list`)
  - [x] `glab mr merge` (대응: `gg pr merge`)
  - [ ] `glab mr note`
  - [ ] `glab mr rebase`
  - [ ] `glab mr reopen`
  - [ ] `glab mr revoke`
  - [ ] `glab mr subscribe`
  - [ ] `glab mr todo`
  - [ ] `glab mr unsubscribe`
  - [x] `glab mr update` (대응: `gg pr ready`)
  - [x] `glab mr view` (대응: `gg pr view`)
- `release`
  - [ ] `glab release create`
  - [ ] `glab release delete`
  - [ ] `glab release download`
  - [ ] `glab release list`
  - [ ] `glab release upload`
  - [ ] `glab release view`
- `repo`
  - [ ] `glab repo archive`
  - [x] `glab repo clone` (대응: `gg repo clone`, `gg clone`)
  - [ ] `glab repo contributors`
  - [x] `glab repo create` (대응: `gg repo create`, `gg create`)
  - [ ] `glab repo delete`
  - [ ] `glab repo fork`
  - [x] `glab repo list` (대응: `gg repo list`, `gg list`)
  - [ ] `glab repo mirror`
  - [ ] `glab repo search`
  - [ ] `glab repo transfer`
  - [x] `glab repo view` (대응: `gg repo view`, `gg view`)
- [x] `glab help` (대응: `gg help`, `gg --help`, `gg -h`)
- [x] `glab version` (대응: 단독 `gg -verison`, 단독 `gg -v`)

<details>
<summary>기타 및 플랫폼 고유 명령 (CI/CD, Incident, Duo, Snippet 등)</summary>

- `alias`
  - [ ] `glab alias delete`
  - [ ] `glab alias list`
  - [ ] `glab alias set`
- [ ] `glab api`
- `ask`
  - [ ] `glab ask git`
- `changelog`
  - [ ] `glab changelog generate`
- [ ] `glab check-update`
- `ci`
  - [ ] `glab ci artifacts`
  - [ ] `glab ci delete`
  - [ ] `glab ci get`
  - [ ] `glab ci lint`
  - [ ] `glab ci list`
  - [ ] `glab ci retry`
  - [ ] `glab ci run`
  - [ ] `glab ci status`
  - [ ] `glab ci trace`
  - [ ] `glab ci trigger`
  - [ ] `glab ci view`
- `cluster`
  - [ ] `glab cluster agent`
- [ ] `glab completion`
- `config`
  - [ ] `glab config get`
  - [ ] `glab config init`
  - [ ] `glab config set`
- `duo`
  - [ ] `glab duo ask`
- `incident`
  - [ ] `glab incident close`
  - [ ] `glab incident list`
  - [ ] `glab incident note`
  - [ ] `glab incident reopen`
  - [ ] `glab incident subscribe`
  - [ ] `glab incident unsubscribe`
  - [ ] `glab incident view`
- `schedule`
  - [ ] `glab schedule create`
  - [ ] `glab schedule delete`
  - [ ] `glab schedule list`
  - [ ] `glab schedule run`
  - [ ] `glab schedule update`
  - [ ] `glab schedule variable`
- `snippet`
  - [ ] `glab snippet create`
  - [ ] `glab snippet delete`
  - [ ] `glab snippet list`
  - [ ] `glab snippet view`
- `ssh-key`
  - [ ] `glab ssh-key add`
  - [ ] `glab ssh-key delete`
  - [ ] `glab ssh-key list`
- `stack`
  - [ ] `glab stack diff`
  - [ ] `glab stack first`
  - [ ] `glab stack init`
  - [ ] `glab stack last`
  - [ ] `glab stack navigate`
  - [ ] `glab stack next`
  - [ ] `glab stack prev`
  - [ ] `glab stack save`
  - [ ] `glab stack sync`
- `token`
  - [ ] `glab token create`
  - [ ] `glab token list`
  - [ ] `glab token revoke`
- `user`
  - [ ] `glab user events`
  - [ ] `glab user status`
- `variable`
  - [ ] `glab variable delete`
  - [ ] `glab variable export`
  - [ ] `glab variable get`
  - [ ] `glab variable list`
  - [ ] `glab variable set`
  - [ ] `glab variable update`

</details>

---

## tea (Gitea CLI) 기능 목록

### 공통 핵심 명령 (Core Commands)
- [x] `tea clone` (대응: `gg repo clone`, `gg clone`)
- `comment`
  - [x] `tea comment create` (대응: `gg issue comment`)
  - [ ] `tea comment list`
- `issues`
  - [x] `tea issues close` (대응: `gg issue close`)
  - [x] `tea issues create` (대응: `gg issue create`)
  - [ ] `tea issues delete`
  - [x] `tea issues list` (대응: `gg issue list`)
  - [x] `tea issues open` (대응: `gg issue view`)
  - [x] `tea issues reopen` (대응: `gg issue reopen`)
- `labels`
  - [ ] `tea labels create`
  - [ ] `tea labels delete`
  - [ ] `tea labels list`
  - [ ] `tea labels update`
- `logins`
  - [ ] `tea logins add`
  - [ ] `tea logins delete`
  - [ ] `tea logins edit`
  - [ ] `tea logins list`
  - [ ] `tea logins view`
- `pulls`
  - [ ] `tea pulls approve`
  - [ ] `tea pulls checkout`
  - [ ] `tea pulls clean`
  - [ ] `tea pulls close`
  - [x] `tea pulls create` (대응: `gg pr create`)
  - [x] `tea pulls list` (대응: `gg pr list`)
  - [ ] `tea pulls merge`
  - [x] `tea pulls open` (대응: `gg pr view`)
  - [ ] `tea pulls reject`
  - [ ] `tea pulls reopen`
- `releases`
  - [ ] `tea releases create`
  - [ ] `tea releases delete`
  - [ ] `tea releases download`
  - [ ] `tea releases edit`
  - [ ] `tea releases list`
- `repos`
  - [x] `tea repos create` (대응: `gg repo create`, `gg create`)
  - [ ] `tea repos delete`
  - [ ] `tea repos flags`
  - [ ] `tea repos fork`
  - [x] `tea repos list` (대응: `gg repo list`, `gg list`)
  - [ ] `tea repos search`
  - [x] `tea repos view` (대응: `gg repo view`, `gg view`)
- [x] `tea help` (대응: `gg help`, `gg --help`, `gg -h`)
- [x] `tea version` (대응: 단독 `gg -verison`, 단독 `gg -v`)

<details>
<summary>기타 및 플랫폼 고유 명령 (Admin, Times, Milestones 등)</summary>

- `actions`
  - [ ] `tea actions runs`
  - [ ] `tea actions secrets`
- `admin`
  - [ ] `tea admin orgs`
  - [ ] `tea admin repos`
  - [ ] `tea admin runners`
  - [ ] `tea admin users`
- [ ] `tea api`
- `branches`
  - [ ] `tea branches list`
- [ ] `tea logout`
- [ ] `tea man`
- `milestones`
  - [ ] `tea milestones create`
  - [ ] `tea milestones delete`
  - [ ] `tea milestones list`
  - [ ] `tea milestones update`
- `notifications`
  - [ ] `tea notifications list`
  - [ ] `tea notifications pin`
  - [ ] `tea notifications read`
- [ ] `tea open`
- `organizations`
  - [ ] `tea organizations create`
  - [ ] `tea organizations delete`
  - [ ] `tea organizations edit`
  - [ ] `tea organizations list`
- `times`
  - [ ] `tea times add`
  - [ ] `tea times delete`
  - [ ] `tea times list`
  - [ ] `tea times reset`
- [ ] `tea whoami`

</details>
---

## gg 고유 설정 기능 목록

- `config`
  - [x] `gg config list`
  - [x] `gg config set`
  - [x] `gg config unset`

## gg 고유 기능 목록

- `pr`
  - [x] `gg pr status` (GitHub, GitLab 지원; Gitea 미지원)
  - [x] `gg pr ready` (GitHub, GitLab 지원; Gitea 미지원)
  - [x] `gg mr` (`gg pr`의 alias)
- `저장소 문맥`
  - [x] `--repo <URL>` (명시한 URL을 저장소 문맥으로 사용)
  - [x] `--remote <name>` (명시한 Git remote를 저장소 문맥으로 사용)
- `실행 설명`
  - [x] `--explain` (선택한 저장소 문맥, Provider, 실행할 CLI 출력)

---

모든 명령과 action은 `--help`를 지원합니다.

```bash
gg repo --help
gg repo list --help
gg issue --help
gg issue list --help
gg pr create --help
gg pr ready --help
gg config --help
gg config set --help
```

`gg <cmd> --help`와 `gg repo <cmd> --help`는 `list`, `view`, `create`, `clone`, `pull`, `push`에서 같은 gg help를 제공합니다. `gg repo commit --help`도 gg가 처리합니다.
`gg commit --help`는 `--no-gpg-sign`을 추가해 git에 전달합니다. Git Main Porcelain 37개와 ancillary 14개 registry 명령은 `gg <cmd> --help`와 `gg repo <cmd> --help` 모두 명령 뒤에 둔 `--help`를 포함한 모든 인자를 git에 그대로 전달합니다.
Git passthrough 명령에는 명령 앞의 gg 전역 flag를 사용할 수 없습니다.

### 사용 예시 (Usage Examples)

#### PR / MR Ready & Draft 전환 (`gg pr ready`)
- PR을 Ready 상태로 전환:
  ```bash
  gg pr ready 42
  ```
  - GitHub: `gh pr ready 42 -R <owner>/<repo>` 호출
  - GitLab: `glab mr update 42 --ready --repo <URL>` 호출
- PR을 Draft 상태로 되돌리기 (`--undo`):
  ```bash
  gg pr ready 42 --undo
  ```
  - GitHub: `gh pr ready 42 --undo -R <owner>/<repo>` 호출
  - GitLab: `glab mr update 42 --draft --repo <URL>` 호출
- Gitea (`tea`):
  - `tea` CLI는 PR Ready/Draft 전환 기능을 지원하지 않으므로 미지원 오류(`pr ready is not supported for tea`)가 반환됩니다.
- 저장소 문맥 플래그와 함께 사용:
  ```bash
  gg --repo https://github.com/owner/repo pr ready 42
  gg pr ready 42 --remote upstream --undo
  ```

# TOBE

- `gg`가 단일 CLI 인터페이스로 `git`, `gh`, `glab`, `tea`의 모든 공통 기능을 투명하게 중계하도록 구현 범위를 점진적으로 확장합니다.
- 체크리스트에서 `[x]`로 표시된 항목은 현재 `gg` 명령으로 실행 경로가 연결된 기능이며, `[ ]`로 표시된 항목은 향후 라우팅 및 플래그 매핑을 추가해야 하는 구현 대상입니다.
- Git Forge 간의 기능 차이를 추상화하여 저장소 원격지(GitHub, GitLab, Gitea, Self-hosted)에 구애받지 않고 일관된 개발자 경험을 제공하는 것을 목표로 합니다.
