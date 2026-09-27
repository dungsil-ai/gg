# Forge 작업의 차이점

현재 설치된 `gg`의 도움말을 기준으로 지원 여부를 확인한다. 이 문서는 모든 명령을 열거하지 않고 기존 CLI에서 전환할 때 판단이 달라지는 부분을 설명한다.

## 이슈와 PR/MR

GitHub PR, GitLab MR, Gitea PR에는 공통으로 `gg pr`를 사용한다. 조회는 다음과 같이 실행한다.

```sh
gg pr list --state open --limit 20
gg pr view 42
gg pr diff 42
gg pr status 42
gg pr comment list 42
gg issue view 7
gg issue comment list 7
```

번호는 실제 조회 결과로 바꾼다. GitLab에서도 `--body`, `--base`, `--head`를 사용하며 `gg`가 Provider별 인자로 변환한다. 생성·댓글·리뷰 요청이 있는 경우의 예시는 다음과 같다.

```sh
gg pr create --title "<title>" --body "<body>" --base main --head <branch> --draft
gg issue create --title "<title>" --body "<body>"
gg pr comment 42 --body "<body>"
gg pr review 42 --request-changes --body "<reason>"
```

- `gg pr status 42`는 `Draft`, `Approval`, `CI`, `Conflict`, `Mergeable` 값을 출력한다. 종료 코드 `0`은 조회 성공일 뿐 CI 통과나 병합 가능을 뜻하지 않는다. `unknown`을 성공으로 해석하지 않는다.
- 번호 없는 `gg pr status`는 이를 지원하는 버전에서 GitHub 개인 PR 현황을 조회한다. 단일 PR의 병합 가능성 판단에는 번호를 지정한다.
- `gg pr checkout 42`는 현재 작업 트리를 변경한다. `--repo`는 PR을 선택할 뿐 작업 디렉터리를 바꾸지 않는다.
- `gg pr merge 42`에는 저장소 정책에 맞는 `--merge`, `--squash`, `--rebase` 중 하나를 선택한다. `--auto`와 `--delete-branch`는 요청에 포함된 경우에만 추가한다. `--auto` 성공을 병합 완료로 보고하지 않는다.
- `gg pr review`에는 `--approve`, `--request-changes`, `--comment` 중 정확히 하나를 지정한다. 변경 요청과 댓글 리뷰에는 본문이 필요하다. GitLab은 승인만, Gitea는 승인과 변경 요청만 지원한다.
- `pr checks`는 GitHub 전용이다. Gitea에서는 `pr diff`, `pr status`, `pr ready`, 댓글 수정·삭제 등 일부 기능을 지원하지 않는다.
- `gg pr comment list`는 일반 PR 댓글을 조회한다. 모든 인라인 리뷰 스레드를 포함한다고 가정하지 않는다. 인라인 댓글이 필요하면 해당 Provider API를 사용한다.
- 댓글 수정·삭제에는 PR/이슈 번호와 댓글 ID가 모두 필요하다. 목록에서 ID를 확인하여 `gg pr comment edit 42 1234 --body "<body>"`처럼 실행한다.

## CI와 workflow

`gg ci`는 실행 단위를 다루고 `gg workflow`는 GitHub Actions workflow를 다룬다. `gg actions`는 `gg ci`의 별칭이다.

```sh
gg ci list --branch main --limit 10
gg ci view 123456
gg workflow list
gg workflow view build.yml --yaml
```

- `gg ci view`에서 ID를 생략하면 현재 브랜치의 최신 실행을 조회한다. 특정 실행을 검증할 때는 ID를 명시한다.
- GitHub의 `gg ci watch <id>`와 `gg ci retry <id>`는 run ID를 받는다. GitLab에서는 두 명령 모두 job ID를 사용하고, `gg ci view <id>`와 `gg ci cancel <id>`는 pipeline ID를 사용한다.
- `gg ci watch`는 계속 실행될 수 있다. 현재 환경에서 허용하는 실행 시간과 대기 방식에 맞게 사용한다.
- `gg ci retry`, `gg ci cancel`, `gg workflow run`은 외부 상태를 변경하므로 조회 요청만으로 실행하지 않는다.
- `gg workflow run build.yml --ref main`은 GitHub 전용이다. `gh workflow run`의 `-f`나 `-F` 입력 플래그까지 지원한다고 가정하지 않는다. 입력값이 필요하면 도움말을 확인하고, 미지원이면 같은 요청 범위에서 API 사용을 검토한다.
- Gitea에서는 `gg ci`와 `gg workflow`를 지원하지 않는다.

## 원시 API와 구조화된 출력

`gg api`는 저장소 문맥으로 Provider를 고르고 `api` 뒤 인자를 그대로 전달한다. 일반 도움말에 표시되지 않는 버전도 있다. API 경로나 응답 형식을 Provider 사이에서 변환하지 않으며, 저장소 문맥을 지정해도 endpoint의 소유자·저장소·프로젝트 경로를 자동으로 채워 주지 않는다.

저장소 문맥 플래그는 반드시 `api` 앞에 둔다. `gg api ... --repo <URL>`은 저장소 문맥 선택이 아니라 원시 API 인자가 된다. `gg api`에는 `--explain`을 붙이지 않는다. `gg --repo <URL> api --help`는 선택된 Provider CLI의 API 도움말을 호출한다.

GitHub에서 PR의 JSON 데이터가 필요한 경우:

```sh
gg --repo https://github.com/owner/repo api repos/owner/repo/pulls/42
gg --repo https://github.com/owner/repo api repos/owner/repo/issues/42/comments --paginate
```

GitLab에서 MR의 JSON 데이터가 필요한 경우:

```sh
gg --repo https://gitlab.com/group/project api projects/group%2Fproject/merge_requests/42
```

GitLab 프로젝트 경로는 `/`를 `%2F`로 인코딩하며 하위 그룹도 포함한다. Provider별 API 경로와 지원 플래그를 확인하고, 필요한 페이지를 모두 조회한 뒤 결과를 해석한다. `--jq`, `--paginate` 등도 선택한 Provider가 지원할 때만 사용한다.

API로 전환해도 사용자의 요청 범위는 바뀌지 않는다. 조회에는 읽기 전용 endpoint와 메서드를 사용한다. 인증 토큰을 출력하거나 환경 변수 값을 덤프하여 문제를 진단하지 않는다.

## 인증과 Self-hosted Provider 설정

기본 domain은 `github.com` → `gh`, `gitlab.com` → `glab`, `gitea.com` → `tea`로 고정된다. Self-hosted host에만 **Provider 설정**을 추가한다. Provider 설정은 host와 CLI의 연결이며 인증 정보나 저장소 URL을 보관하지 않는다.

```sh
gg config list
gg auth status
```

설정 작업이 요청되었고 host의 종류를 확인했다면 다음처럼 연결한다.

```sh
gg config set gitlab.example.com glab
gg config set gitea.example.com tea
```

- host에 scheme이나 경로를 넣지 않는다. host는 소문자로 정규화되고 port는 제거되므로 같은 host의 여러 port는 Provider 설정을 공유한다. URL 전체는 Forge 명령의 `--repo`에 지정한다.
- 알 수 없는 host에서는 Provider 선택을 요구할 수 있다. 에이전트가 추측하여 Provider 설정을 저장하지 말고 사용자가 제공한 정보와 기존 설정을 확인한다.
- `gg auth status`의 `LOGIN` 열을 확인한다. 명령이 성공해도 모든 host에 로그인되었다는 뜻은 아니며, `no`와 `no cli`는 각각 미인증과 CLI 미설치를 뜻한다.
- 인증 작업이 요청되었을 때 `gg auth login --hostname github.example.com`은 기본적으로 `gh`를 사용한다. 저장소 문맥이나 hostname만 보고 GitLab으로 전환하지 않는다.
- `gg auth --help`에서 `--provider glab` 지원을 확인한 뒤 `gg auth login --provider glab --hostname gitlab.example.com`을 사용할 수 있다. GitLab 릴레이는 `login`, `logout`, `token`만 지원하며 `gg auth status`에는 `--provider`를 붙이지 않는다. 지원하지 않는 구버전에서는 원래 CLI로 조용히 전환하지 않는다.
- `gg auth token`은 비밀 값을 출력하므로 일반 인증 확인에 사용하지 않는다.
- `tea logins` 관리는 `gg`에서 지원하지 않는다. Gitea 로그인 설정이 없으면 사용자에게 필요한 설정을 알린다. 다른 host의 로그인을 대신 사용하지 않는다.
