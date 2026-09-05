# TODO

## 설치

`gg`는 두 가지 방법으로 설치할 수 있습니다.

### GitHub Release 파일

[Releases](https://github.com/dungsil-ai/gg/releases) 페이지에서 사용 중인 OS/CPU 종류에 맞는 Release 파일을 내려받습니다. Archive는 `gg_<version>_<os>_<arch>.<ext>` 이름을 쓰며 Windows는 `.zip`, Linux와 macOS는 `.tar.gz`입니다. 각 Release 파일 옆에는 같은 이름에 `.sha256`을 붙인 checksum 파일이 따로 있습니다. 압축을 풀기 전에 checksum을 검증합니다.

```bash
# 예시: Linux amd64
sha256sum -c gg_<version>_linux_amd64.tar.gz.sha256
```

### go install

```bash
go install github.com/dungsil-ai/gg@latest
```

---

## 릴리즈 절차

준비자는 저장소 default branch(`main`)에서 변경사항을 stage하지 않은 상태로 다음 순서를 실행합니다.

```bash
gg commit --allow-empty -m "release: vMAJOR.MINOR.PATCH"
gg tag -a vMAJOR.MINOR.PATCH -m "Release vMAJOR.MINOR.PATCH"
gg push --atomic origin HEAD:refs/heads/main refs/tags/vMAJOR.MINOR.PATCH
```

전용 빈 릴리즈 커밋을 만들고, 그 `HEAD`에 annotated tag를 만든 뒤, 커밋과 tag를 같은 원격 transaction으로 atomic push합니다. `--atomic`은 두 ref 중 하나라도 원격이 거부하면 나머지도 갱신하지 않아 partial push를 막습니다. 원격이 atomic push를 지원하지 않아도 non-atomic fallback이나 force push는 쓰지 않습니다.

release Workflow는 다음 조건을 모두 만족해야 GitHub Release를 게시합니다.

- 저장소에서 GitHub Immutable Releases 설정이 활성화되어 있어야 합니다.
- `RELEASE_ADMIN_TOKEN` secret(`Administration: read` 권한)이 등록되어 있어야 합니다.
- 같은 tag의 기존 Release가 존재하지 않아야 합니다.
- 같은 version을 다시 쓰거나 tag를 이동하거나 force push하는 것은 금지합니다.
- tag는 유의적 버전 형식(`vMAJOR.MINOR.PATCH`, 선택적 접미사 허용)이어야 하며, `BuildAndPackageRelease`가 ldflags 주입 전에 이를 검증합니다.

---

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

- [x] `git apply` (대응: `gg repo apply`, `gg apply`)
- [x] `git cat-file` (대응: `gg repo cat-file`, `gg cat-file`)
- [x] `git check-attr` (대응: `gg repo check-attr`, `gg check-attr`)
- [x] `git check-ignore` (대응: `gg repo check-ignore`, `gg check-ignore`)
- [x] `git check-mailmap` (대응: `gg repo check-mailmap`, `gg check-mailmap`)
- [x] `git check-ref-format` (대응: `gg repo check-ref-format`, `gg check-ref-format`)
- [x] `git checkout-index` (대응: `gg repo checkout-index`, `gg checkout-index`)
- [x] `git column` (대응: `gg repo column`, `gg column`)
- [x] `git commit-graph` (대응: `gg repo commit-graph`, `gg commit-graph`)
- [x] `git commit-tree` (대응: `gg repo commit-tree`, `gg commit-tree`)
- [x] `git credential` (대응: `gg repo credential`, `gg credential`)
- [x] `git credential-cache` (대응: `gg repo credential-cache`, `gg credential-cache`)
- [x] `git credential-store` (대응: `gg repo credential-store`, `gg credential-store`)
- [x] `git daemon` (대응: `gg repo daemon`, `gg daemon`)
- [x] `git diff-files` (대응: `gg repo diff-files`, `gg diff-files`)
- [x] `git diff-index` (대응: `gg repo diff-index`, `gg diff-index`)
- [x] `git diff-tree` (대응: `gg repo diff-tree`, `gg diff-tree`)
- [x] `git fast-export` (대응: `gg repo fast-export`, `gg fast-export`)
- [x] `git fast-import` (대응: `gg repo fast-import`, `gg fast-import`)
- [x] `git fetch-pack` (대응: `gg repo fetch-pack`, `gg fetch-pack`)
- [x] `git for-each-ref` (대응: `gg repo for-each-ref`, `gg for-each-ref`)
- [x] `git for-each-repo` (대응: `gg repo for-each-repo`, `gg for-each-repo`)
- [x] `git hash-object` (대응: `gg repo hash-object`, `gg hash-object`)
- [x] `git http-backend` (대응: `gg repo http-backend`, `gg http-backend`)
- [x] `git http-fetch` (대응: `gg repo http-fetch`, `gg http-fetch`)
- [x] `git http-push` (대응: `gg repo http-push`, `gg http-push`)
- [x] `git index-pack` (대응: `gg repo index-pack`, `gg index-pack`)
- [x] `git ls-files` (대응: `gg repo ls-files`, `gg ls-files`)
- [x] `git ls-remote` (대응: `gg repo ls-remote`, `gg ls-remote`)
- [x] `git ls-tree` (대응: `gg repo ls-tree`, `gg ls-tree`)
- [x] `git mailinfo` (대응: `gg repo mailinfo`, `gg mailinfo`)
- [x] `git mailsplit` (대응: `gg repo mailsplit`, `gg mailsplit`)
- [x] `git merge-base` (대응: `gg repo merge-base`, `gg merge-base`)
- [x] `git merge-file` (대응: `gg repo merge-file`, `gg merge-file`)
- [x] `git merge-index` (대응: `gg repo merge-index`, `gg merge-index`)
- [x] `git mktag` (대응: `gg repo mktag`, `gg mktag`)
- [x] `git mktree` (대응: `gg repo mktree`, `gg mktree`)
- [x] `git multi-pack-index` (대응: `gg repo multi-pack-index`, `gg multi-pack-index`)
- [x] `git name-rev` (대응: `gg repo name-rev`, `gg name-rev`)
- [x] `git pack-objects` (대응: `gg repo pack-objects`, `gg pack-objects`)
- [x] `git pack-redundant` (대응: `gg repo pack-redundant`, `gg pack-redundant`)
- [x] `git pack-refs` (대응: `gg repo pack-refs`, `gg pack-refs`)
- [x] `git patch-id` (대응: `gg repo patch-id`, `gg patch-id`)
- [x] `git prune` (대응: `gg repo prune`, `gg prune`)
- [x] `git read-tree` (대응: `gg repo read-tree`, `gg read-tree`)
- [x] `git receive-pack` (대응: `gg repo receive-pack`, `gg receive-pack`)
- [x] `git reflog` (대응: `gg repo reflog`, `gg reflog`)
- [x] `git remote` (대응: `gg repo remote`, `gg remote`)
- [x] `git repack` (대응: `gg repo repack`, `gg repack`)
- [x] `git replace` (대응: `gg repo replace`, `gg replace`)
- [x] `git rev-list` (대응: `gg repo rev-list`, `gg rev-list`)
- [x] `git rev-parse` (대응: `gg repo rev-parse`, `gg rev-parse`)
- [x] `git send-pack` (대응: `gg repo send-pack`, `gg send-pack`)
- [x] `git show-branch` (대응: `gg repo show-branch`, `gg show-branch`)
- [x] `git show-index` (대응: `gg repo show-index`, `gg show-index`)
- [x] `git show-ref` (대응: `gg repo show-ref`, `gg show-ref`)
- [x] `git stripspace` (대응: `gg repo stripspace`, `gg stripspace`)
- [x] `git symbolic-ref` (대응: `gg repo symbolic-ref`, `gg symbolic-ref`)
- [x] `git unpack-file` (대응: `gg repo unpack-file`, `gg unpack-file`)
- [x] `git unpack-objects` (대응: `gg repo unpack-objects`, `gg unpack-objects`)
- [x] `git update-index` (대응: `gg repo update-index`, `gg update-index`)
- [x] `git update-ref` (대응: `gg repo update-ref`, `gg update-ref`)
- [x] `git update-server-info` (대응: `gg repo update-server-info`, `gg update-server-info`)
- [x] `git upload-archive` (대응: `gg repo upload-archive`, `gg upload-archive`)
- [x] `git upload-pack` (대응: `gg repo upload-pack`, `gg upload-pack`)
- [x] `git var` (대응: `gg repo var`, `gg var`)
- [x] `git verify-commit` (대응: `gg repo verify-commit`, `gg verify-commit`)
- [x] `git verify-pack` (대응: `gg repo verify-pack`, `gg verify-pack`)
- [x] `git verify-tag` (대응: `gg repo verify-tag`, `gg verify-tag`)
- [x] `git write-tree` (대응: `gg repo write-tree`, `gg write-tree`)

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
  - [x] `gh issue comment` (대응: `gg issue comment`; 조회·수정·삭제는 `gg issue comment list|edit|delete`로 `gh api` 중계)
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
  - [x] `gh label create` (대응: `gg label create`)
  - [ ] `gh label delete`
  - [ ] `gh label edit`
  - [x] `gh label list` (대응: `gg label list`)
- `pr`
  - [ ] `gh pr checkout`
  - [ ] `gh pr checks`
  - [x] `gh pr close` (대응: `gg pr close`)
  - [x] `gh pr comment` (대응: `gg pr comment`; 조회·수정·삭제는 `gg pr comment list|edit|delete`로 `gh api` 중계)
  - [x] `gh pr create` (대응: `gg pr create`)
  - [ ] `gh pr diff`
  - [ ] `gh pr edit`
  - [x] `gh pr list` (대응: `gg pr list`)
  - [ ] `gh pr lock`
  - [x] `gh pr merge` (대응: `gg pr merge`)
  - [x] `gh pr ready` (대응: `gg pr ready`)
  - [x] `gh pr reopen` (대응: `gg pr reopen`)
  - [ ] `gh pr review`
  - [ ] `gh pr status`
  - [ ] `gh pr unlock`
  - [ ] `gh pr update-branch`
  - [x] `gh pr view` (대응: `gg pr view`)
- `release`
  - [x] `gh release create` (대응: `gg release create`)
  - [x] `gh release delete` (대응: `gg release delete`)
  - [x] `gh release delete-asset` (대응: `gg release delete-asset`; GitLab 미지원)
  - [x] `gh release download` (대응: `gg release download`)
  - [x] `gh release edit` (대응: `gg release edit`; GitLab 미지원)
  - [x] `gh release list` (대응: `gg release list`)
  - [x] `gh release upload` (대응: `gg release upload`)
  - [x] `gh release view` (대응: `gg release view`)
- `repo`
  - [ ] `gh repo archive` (`gg repo archive`와 `gg archive`는 git archive passthrough로 쓰는 이름이라 미지원)
  - [ ] `gh repo autolink` (하위 명령 그룹이라 미지원)
  - [x] `gh repo clone` (대응: `gg repo clone`, `gg clone`)
  - [x] `gh repo create` (대응: `gg repo create`, `gg create`)
  - [x] `gh repo delete` (대응: `gg repo delete`, `gg delete`)
  - [ ] `gh repo deploy-key` (하위 명령 그룹이라 미지원)
  - [x] `gh repo edit` (대응: `gg repo edit`, `gg edit`; GitLab·Gitea 미지원)
  - [x] `gh repo fork` (대응: `gg repo fork`, `gg fork`)
  - [x] `gh repo list` (대응: `gg repo list`, `gg list`)
  - [x] `gh repo rename` (대응: `gg repo rename`, `gg rename`; GitLab·Gitea 미지원)
  - [x] `gh repo set-default` (대응: `gg repo set-default`, `gg set-default`; GitLab·Gitea 미지원)
  - [x] `gh repo sync` (대응: `gg repo sync`, `gg sync`; GitLab·Gitea 미지원)
  - [ ] `gh repo unarchive` (archive 미지원과 함께 보류)
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
  - [x] `gh run cancel` (대응: `gg ci cancel`)
  - [ ] `gh run delete`
  - [ ] `gh run download`
  - [x] `gh run list` (대응: `gg ci list`)
  - [x] `gh run rerun` (대응: `gg ci retry`)
  - [x] `gh run view` (대응: `gg ci view`)
  - [x] `gh run watch` (대응: `gg ci watch`)
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
  - [x] `glab issue note` (대응: `gg issue comment`; `gg issue comment list|edit|delete`는 `glab api` 중계)
  - [x] `glab issue reopen` (대응: `gg issue reopen`)
  - [ ] `glab issue subscribe`
  - [ ] `glab issue todo`
  - [ ] `glab issue unsubscribe`
  - [ ] `glab issue update`
  - [x] `glab issue view` (대응: `gg issue view`)
- `label`
  - [x] `glab label create` (대응: `gg label create`)
  - [x] `glab label list` (대응: `gg label list`)
- `mr`
  - [ ] `glab mr approve`
  - [ ] `glab mr approvers`
  - [ ] `glab mr checkout`
  - [x] `glab mr close` (대응: `gg pr close`)
  - [x] `glab mr create` (대응: `gg pr create`)
  - [ ] `glab mr delete`
  - [ ] `glab mr diff`
  - [x] `glab mr list` (대응: `gg pr list`)
  - [x] `glab mr merge` (대응: `gg pr merge`)
  - [x] `glab mr note` (대응: `gg pr comment`; `gg pr comment list|edit|delete`는 `glab api` 중계)
  - [ ] `glab mr rebase`
  - [x] `glab mr reopen` (대응: `gg pr reopen`)
  - [ ] `glab mr revoke`
  - [ ] `glab mr subscribe`
  - [ ] `glab mr todo`
  - [ ] `glab mr unsubscribe`
  - [x] `glab mr update` (대응: `gg pr ready`)
  - [x] `glab mr view` (대응: `gg pr view`)
- `release`
  - [x] `glab release create` (대응: `gg release create`; `--draft`·`--prerelease` 미지원)
  - [x] `glab release delete` (대응: `gg release delete`)
  - [x] `glab release download` (대응: `gg release download`)
  - [x] `glab release list` (대응: `gg release list`)
  - [x] `glab release upload` (대응: `gg release upload`)
  - [x] `glab release view` (대응: `gg release view`)
- `repo`
  - [ ] `glab repo archive` (저장소 보관이 아니라 스냅샷 다운로드 명령이라 `gg repo archive`와 연결하지 않음)
  - [x] `glab repo clone` (대응: `gg repo clone`, `gg clone`)
  - [ ] `glab repo contributors`
  - [x] `glab repo create` (대응: `gg repo create`, `gg create`)
  - [x] `glab repo delete` (대응: `gg repo delete`, `gg delete`)
  - [x] `glab repo fork` (대응: `gg repo fork`, `gg fork`)
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
  - [x] `glab ci cancel` (대응: `gg ci cancel`)
  - [ ] `glab ci delete`
  - [x] `glab ci get` (대응: `gg ci view`)
  - [ ] `glab ci lint`
  - [x] `glab ci list` (대응: `gg ci list`)
  - [x] `glab ci retry` (대응: `gg ci retry`)
  - [ ] `glab ci run`
  - [ ] `glab ci status`
  - [x] `glab ci trace` (대응: `gg ci watch`)
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
  - [x] `tea comment create` (대응: `gg issue comment`, `gg pr comment`)
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
  - [x] `tea pulls close` (대응: `gg pr close`)
  - [x] `tea pulls create` (대응: `gg pr create`)
  - [x] `tea pulls list` (대응: `gg pr list`)
  - [ ] `tea pulls merge`
  - [x] `tea pulls open` (대응: `gg pr view`)
  - [ ] `tea pulls reject`
  - [x] `tea pulls reopen` (대응: `gg pr reopen`)
- `releases`
  - [ ] `tea releases create`
  - [ ] `tea releases delete`
  - [ ] `tea releases download`
  - [ ] `tea releases edit`
  - [ ] `tea releases list`
- `repos`
  - [x] `tea repos create` (대응: `gg repo create`, `gg create`)
  - [x] `tea repos delete` (대응: `gg repo delete`, `gg delete`)
  - [ ] `tea repos flags`
  - [x] `tea repos fork` (대응: `gg repo fork`, `gg fork`)
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
  - exit code 계약: 조회 자체가 실패할 때만 0이 아닌 exit code를 냅니다 — 하위 CLI(gh/glab) 미설치 시 127, 자식이 신호로 종료되면 128+신호 코드, 그 외 조회 실패 시 1. 조회 성공 시 병합 불가·CI 실패·승인 대기는 결과 값이며 exit 0입니다. CI 값 범위는 pass|fail|pending|none|unknown이고, NEUTRAL/SKIPPED 체크는 pass로 셉니다.
  - [x] `gg pr ready` (GitHub, GitLab 지원; Gitea 미지원)
  - [x] `gg pr comment` (PR 댓글 입력; GitHub, GitLab, Gitea 지원)
  - [x] `gg pr comment list` / `gg pr comment edit` / `gg pr comment delete` (PR 댓글 조회·수정·삭제; GitHub, GitLab 지원 — `gh api`/`glab api` 중계. Gitea 미지원)
  - [x] `gg mr` (`gg pr`의 alias)
- `issue` 댓글
  - [x] `gg issue comment` (이슈 댓글 입력; GitHub, GitLab, Gitea 지원)
  - [x] `gg issue comment list` / `gg issue comment edit` / `gg issue comment delete` (이슈 댓글 조회·수정·삭제; GitHub, GitLab 지원 — `gh api`/`glab api` 중계. Gitea 미지원)
- `issue` 관계 등록
  - [x] `gg issue sub-issue` (GitHub 지원; GitLab·Gitea 미지원) — 이슈를 다른 이슈의 native sub-issue로 등록
  - [x] `gg issue blocked-by` (GitHub 지원; GitLab·Gitea 미지원) — 이슈에 blocked-by 의존성을 등록
  - [x] `gg issue type` (GitHub 지원; GitLab·Gitea 미지원) — 이슈 종류(issue type)를 설정
  - sub-issue와 blocked-by는 GitHub API가 body로 numeric database id를 요구하므로 gg가 `gh api repos/<owner>/<repo>/issues/<number> --jq .id`로 번호→id를 미리 조회한 뒤 `gh api`를 호출합니다. `--explain`은 이 조회를 실행하지 않습니다.
- `ci`
  - [x] `gg ci list` (GitHub: `gh run list`, GitLab: `glab ci list`; Gitea 미지원)
  - [x] `gg ci view` (GitHub: `gh run view`, GitLab: `glab ci get`; id 생략 시 현재 branch의 최신 실행)
  - [x] `gg ci watch` (GitHub: `gh run watch`, GitLab: `glab ci trace`; GitLab id는 job id)
  - [x] `gg ci retry` (GitHub: `gh run rerun`, GitLab: `glab ci retry`; GitLab id는 job id)
  - [x] `gg ci cancel` (GitHub: `gh run cancel`, GitLab: `glab ci cancel pipeline`; Gitea 미지원)
  - [x] `gg actions` (`gg ci`의 alias)
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

#### PR / MR 닫기·다시 열기 (`gg pr close`, `gg pr reopen`)
- PR 닫기:
  ```bash
  gg pr close 42
  ```
  - GitHub: `gh pr close 42 -R <owner>/<repo>` 호출
  - GitLab: `glab mr close 42 --repo <URL>` 호출
  - Gitea: `tea pulls close 42 ...` 호출
- 닫은 PR 다시 열기:
  ```bash
  gg pr reopen 42
  ```
  - GitHub: `gh pr reopen 42 -R <owner>/<repo>` 호출
  - GitLab: `glab mr reopen 42 --repo <URL>` 호출
  - Gitea: `tea pulls reopen 42 ...` 호출
- 저장소 문맥 플래그와 함께 사용:
  ```bash
  gg --repo https://github.com/owner/repo pr close 42
  gg pr reopen 42 --remote upstream
  ```

#### PR 댓글 입력·조회·수정·삭제 (`gg pr comment`)
- PR에 댓글 달기:
  ```bash
  gg pr comment 42 --body "확인했습니다"
  ```
  - GitHub: `gh pr comment 42 --body "확인했습니다" -R <owner>/<repo>` 호출
  - GitLab: `glab mr note 42 --message "확인했습니다" --repo <URL>` 호출
  - Gitea: `tea comment 42 "확인했습니다" ...` 호출
- 댓글 목록 조회 (댓글 ID는 JSON 출력의 `id` 필드):
  ```bash
  gg pr comment list 42
  ```
  - GitHub: `gh api repos/<owner>/<repo>/issues/42/comments` 호출
  - GitLab: `glab api projects/<owner>%2F<repo>/merge_requests/42/notes` 호출
  - Gitea: 미지원 오류(`pr comment list is not supported for tea`)가 반환됩니다
- 댓글 수정:
  ```bash
  gg pr comment edit 42 1234 --body "수정했습니다"
  ```
  - GitHub: `gh api -X PATCH repos/<owner>/<repo>/issues/comments/1234 -f body="수정했습니다"` 호출
  - GitLab: `glab api -X PUT projects/<owner>%2F<repo>/merge_requests/42/notes/1234 -f body="수정했습니다"` 호출
- 댓글 삭제:
  ```bash
  gg pr comment delete 42 1234
  ```
  - GitHub: `gh api -X DELETE repos/<owner>/<repo>/issues/comments/1234` 호출
  - GitLab: `glab api -X DELETE projects/<owner>%2F<repo>/merge_requests/42/notes/1234` 호출

#### Issue 관계 등록 (`gg issue sub-issue`, `gg issue blocked-by`, `gg issue type`)
- 이슈를 parent의 sub-issue로 등록:
  ```bash
  gg issue sub-issue 42 --parent 7
  ```
  - GitHub: 번호 42의 numeric database id를 `gh api repos/<owner>/<repo>/issues/42 --jq .id`로 조회한 뒤 `gh api --method POST repos/<owner>/<repo>/issues/7/sub_issues -F sub_issue_id=<id>` 호출
- 이슈에 blocked-by 의존성 등록:
  ```bash
  gg issue blocked-by 42 --blocker 7
  ```
  - GitHub: 번호 7의 numeric database id를 조회한 뒤 `gh api --method POST repos/<owner>/<repo>/issues/42/dependencies/blocked_by -F issue_id=<id>` 호출
- 이슈 종류 설정:
  ```bash
  gg issue type 42 --name Bug
  ```
  - GitHub: `gh api --method PATCH repos/<owner>/<repo>/issues/42 -F type=Bug` 호출
- GitLab과 Gitea:
  - 관계 등록은 GitHub 고유 기능이라 미지원 오류(예: `issue sub-issue is not supported for glab`)가 반환됩니다.
- 저장소 문맥 플래그와 함께 사용:
  ```bash
  gg --repo https://github.com/owner/repo issue sub-issue 42 --parent 7
  gg issue blocked-by 42 --blocker 7 --remote upstream
  ```
- 같은 두 번호를 지정하는 자기 참조(`gg issue sub-issue 42 --parent 42`)는 usage 오류로 거부합니다.
#### CI 실행 조회·제어 (`gg ci`, alias: `gg actions`)
- CI 실행 목록:
  ```bash
  gg ci list
  gg ci list --branch main --limit 10
  ```
  - GitHub: `gh run list -R <owner>/<repo> [--branch main] [--limit 10]` 호출
  - GitLab: `glab ci list --repo <URL> [--ref main] [--per-page 10]` 호출
- 하나의 CI 실행 보기 (id 생략 시 현재 branch의 최신 실행):
  ```bash
  gg ci view 1234567890
  ```
  - GitHub: `gh run view 1234567890 -R <owner>/<repo>` 호출
  - GitLab: `glab ci get --pipeline-id 1234567890 --repo <URL>` 호출
- 실행 로그 실시간 추적 (GitLab id는 job id):
  ```bash
  gg ci watch 1234567890
  ```
  - GitHub: `gh run watch 1234567890 -R <owner>/<repo>` 호출
  - GitLab: `glab ci trace 1234567890 --repo <URL>` 호출
- 재실행과 취소:
  ```bash
  gg ci retry 1234567890
  gg ci cancel 1234567890
  ```
  - GitHub: `gh run rerun`, `gh run cancel` 호출
  - GitLab: `glab ci retry`, `glab ci cancel pipeline` 호출
- Gitea (`tea`):
  - `gg ci` 전체는 미지원 오류(`ci is not supported for tea`)가 반환됩니다.
- `gg actions`는 `gg ci`와 완전히 같은 동작을 합니다 (`gg actions list` 등).

# TOBE

- `gg`가 단일 CLI 인터페이스로 `git`, `gh`, `glab`, `tea`의 모든 공통 기능을 투명하게 중계하도록 구현 범위를 점진적으로 확장합니다.
- 체크리스트에서 `[x]`로 표시된 항목은 현재 `gg` 명령으로 실행 경로가 연결된 기능이며, `[ ]`로 표시된 항목은 향후 라우팅 및 플래그 매핑을 추가해야 하는 구현 대상입니다.
- Git Forge 간의 기능 차이를 추상화하여 저장소 원격지(GitHub, GitLab, Gitea, Self-hosted)에 구애받지 않고 일관된 개발자 경험을 제공하는 것을 목표로 합니다.
