---
name: gg
description: Git 저장소와 GitHub·GitLab·Gitea의 이슈, PR/MR, CI, 릴리즈를 git, gh, glab, tea 대신 gg로 다룬다. 저장소 상태 조회, 브랜치·커밋·원격 작업이나 Forge 작업을 실행할 때 사용한다. Git 개념 설명이나 CLI 구현 자체만 수정하는 작업에는 적용하지 않는다.
---

# gg

저장소 작업에는 `git`, `gh`, `glab`, `tea`를 직접 호출하지 않고 `gg`를 우선 사용한다. `gg`는 독립적인 Git·Forge 구현이 아니라 설치된 Git과 Provider CLI를 호출하는 도구이므로, 해당 실행 파일과 인증은 여전히 필요하다.

사용자가 도구를 명시했거나 상위 지침이 전용 도구를 요구하면 그 선택을 따른다. 이 스킬은 커밋, push, PR 생성·병합, 설정 변경을 별도로 허가하지 않는다. 요청 범위 안에서만 실행한다.

## 실행 전 확인

- 처음 사용할 때 `gg --version`과 `gg --help`로 실행 가능 여부와 지원 명령을 확인한다. 의존 CLI 버전까지 필요하면 `gg -v`를 사용한다.
- 사용하려는 Forge 명령의 플래그는 `gg pr create --help`처럼 해당 명령의 도움말에서 확인한다. 설치된 버전이 이 문서보다 오래되면 도움말과 실제 오류를 기준으로 판단한다.
- 실행 도구의 작업 디렉터리를 로컬 저장소로 지정한다. `git -C <path>`를 `gg -C <path>`로 바꾸지 않는다.
- `gg`나 의존 CLI가 없으면 누락된 도구를 알린다. 설치 요청이 없는데 자동으로 설치하거나 원래 CLI로 조용히 전환하지 않는다.

## 명령 선택

아래 표는 명령 계열의 대응 관계다. Forge 명령의 나머지 인자까지 동일하다는 뜻은 아니다.

| 기존 작업 | 사용할 명령 | 주의점 |
| --- | --- | --- |
| `git status`, `diff`, `log`, `show` | `gg status`, `diff`, `log`, `show` | action 뒤 Git 인자를 그대로 전달한다. |
| `git add`, `commit`, `switch`, `fetch`, `push` | `gg add`, `commit`, `switch`, `fetch`, `push` | `commit`은 기본적으로 서명을 비활성화한다. |
| `git worktree`, `rev-parse`, `ls-files` | `gg worktree`, `rev-parse`, `ls-files` | 상위 지침이 관리형 worktree 도구를 요구하면 그 도구를 사용한다. |
| `gh issue`, `glab issue`, `tea issues` | `gg issue` | `list`, `view`, `create` 등 실제 하위 명령을 확인한다. |
| `gh pr`, `glab mr`, `tea pulls` | `gg pr` | `gg mr`도 같은 명령 계열이다. |
| `gh run`, `glab ci` | `gg ci` | `gh run rerun`에 대응하는 명령은 `gg ci retry`다. |
| `gh workflow` | `gg workflow` | GitHub 전용이며 기존 플래그를 모두 지원하지는 않는다. |
| Forge 저장소·라벨·릴리즈 작업 | `gg repo`, `gg label`, `gg release` | Provider별 지원 범위를 확인한다. |
| `gh api`, `glab api`, `tea api` | `gg api` | API 경로와 인자는 Provider별 형식을 유지한다. |

## Git 전달 명령

지원되는 Git 전달 명령은 `gg <action> [args...]` 또는 `gg repo <action> [args...]`으로 실행한다. action 뒤 인자의 순서와 값, 표준 입출력, 자식 프로세스의 종료 코드를 유지한다.

```sh
gg status --short
gg diff --stat
gg diff --cached
gg log -5 --oneline
gg remote -v
gg rev-parse --show-toplevel
```

작업 요청에 커밋이나 push가 포함되어 있다면 필요한 파일만 선택하여 실행한다.

```sh
gg add -- path/to/file
gg diff --cached
gg commit -m "<repository-convention-message>"
gg push -u origin <branch>
```

인자를 옮길 때 다음 예외를 지킨다.

- Git 전달 명령 앞에 `--repo`, `--remote`, `--explain`을 붙이지 않는다. action 뒤의 같은 토큰도 `gg`가 처리하지 않고 Git으로 전달한다. 원격 지정에는 `gg fetch upstream`, `gg push origin <branch>`처럼 Git 인자를 사용한다.
- `gg commit`은 Git 인자 앞에 `--no-gpg-sign`을 추가한다. 서명이 필요한 작업에서는 기존 서명 요구를 무시하지 말고 명시적인 서명 옵션과 환경을 확인한다.
- `gg config`는 Git 설정이 아니라 host와 Provider의 연결을 관리한다. `git config`나 Git 전역 옵션 `-c`에 대응한다고 가정하지 않는다.
- `gg clone <URL> [DIR]`은 별도 Forge 명령이다. 전체 URL을 사용하고 `--depth` 등 Git clone 옵션을 임의로 덧붙이지 않는다. HTTPS 또는 SSH를 사용하며 HTTP 허용은 별도의 사용자 선택이 필요하다.
- `gg repo archive`는 `git archive`에 대응한다. 호스팅된 저장소를 보관 처리하는 명령이 아니다.
- `gg repo filter-repo`는 `gg` 자체 히스토리 재작성 기능이다. 외부 `git-filter-repo`의 모든 옵션을 지원한다고 가정하지 않는다.

## Forge 명령과 저장소 문맥

명령이 대상으로 삼는 저장소를 **저장소 문맥**이라고 한다. 기본 선택 순서는 현재 브랜치의 upstream, `origin`, 유일한 remote다. 선택이 모호하거나 사용자 지정 저장소가 있으면 명시한다.

```sh
gg --remote upstream issue list --state open --limit 20
gg --repo https://github.com/owner/repo pr view 42
gg --repo https://gitlab.com/group/project pr view 42
```

- `--repo`에는 host가 포함된 저장소 URL을 사용한다. `gh -R owner/repo`를 그대로 옮기지 않는다.
- `--repo`와 `--remote`는 함께 사용할 수 없다. 일관성을 위해 명령 앞에 배치하되, 해당 명령이 저장소 문맥 플래그를 지원하는지 확인한다.
- 일반 Forge 명령은 임의의 Provider CLI 플래그를 전달하지 않는다. `--json`, `--jq`, `--body-file`, `--fill`, `--web` 등을 도움말 확인 없이 붙이지 않는다. 본문 파일은 셸이나 파일 도구로 읽어 `--body`에 하나의 인자로 전달한다.
- 단일 이슈·PR 작업에는 조회한 번호를 명시한다. PR URL이나 브랜치 이름, 번호 생략이 지원된다고 가정하지 않는다.
- 실행 전 저장소 문맥과 Provider를 확인해야 하면 해당 명령이 지원하는 `--explain`을 사용한다. 예: `gg --repo https://github.com/owner/repo --explain pr view 42`. 일반 Forge 명령에서는 본 작업을 실행하지 않고 저장소 문맥, Provider, CLI를 출력한다. 전체 인자나 원격 작업의 성공 여부를 검증하는 기능은 아니다.

PR/MR 상태 판단, CI 식별자, 인증·Self-hosted 설정 또는 원시 API가 필요하면 [Forge 작업의 차이점](references/forge.md)에서 해당 절을 읽는다.

## 미지원 기능과 실패 처리

명령이나 플래그가 거절되면 해당 도움말과 Provider 지원 범위를 확인한다. 같은 실패를 다른 플래그 조합으로 반복하거나 존재하지 않는 `gg git`, `gg gh`, `gg glab`, `gg tea` 명령을 만들지 않는다.

공통 명령으로 해결할 수 없는 Forge 작업은 같은 요청 범위에서 `gg api`로 수행할 수 있는지 검토한다. `gg api`도 불가능하거나 Git 설정처럼 대응 명령이 없으면 제약을 알리고 원래 CLI를 사용할지 묻는다. 사용자가 이미 해당 예외를 허용했다면 다시 묻지 않는다.

외부 변경이 도중에 실패하면 조회 명령으로 실제 반영 여부를 확인한 뒤 재시도를 판단한다. 생성·댓글·재실행 요청을 무조건 반복하지 않는다. `--explain` 결과만으로 변경 완료를 보고하지 않는다.
