# gg forge 라우팅 설계

날짜: 2026-08-18
상태: 승인됨

## 목표

`gg`는 GitHub, GitLab, Gitea 저장소에서 같은 명령 문법을 제공하는 Go 단일 실행 파일이다.

`gg`는 저장소 URL의 host를 보고 다음 CLI 중 하나를 고른다.

- GitHub: `gh`
- GitLab: `glab`
- Gitea: `tea`

`gg`는 forge API를 직접 호출하지 않는다. 명령과 option만 바꾸고, 선택한 CLI 또는 Git을 실행한다. 자식 process의 stdin, stdout, stderr, exit code는 그대로 전달한다.

## 범위

지원 자원:

- repository
- issue
- pull request / merge request

지원 repository 동작:

- `list`
- `view`
- `create`
- `clone`
- `pull`
- `push`

지원 issue와 pull request 동작:

- `list`
- `view`
- `create`

범위 밖:

- forge API 직접 호출
- 출력 형식 통합
- token 저장 또는 token 주입
- provider 전용 option 전달
- fork, delete, merge, review, CI 기능

## 배포

- 언어: Go
- 외부 Go dependency: 없음
- 결과물: Windows, Linux, macOS용 단일 실행 파일
- binary 이름: `gg`

## 명령 문법

기본 문법:

```text
gg [--repo <URL>] <resource> <action> [args] [flags]
```

Repository는 `repo`를 생략할 수 있다.

```text
gg repo list = gg list
gg repo view = gg view
gg repo create = gg create
gg repo clone = gg clone
gg repo pull = gg pull
gg repo push = gg push
```

Issue와 pull request는 자원 이름을 생략하지 않는다.

```text
gg issue list [--state open|closed|all] [--limit N]
gg issue view <number>
gg issue create [--title TEXT] [--body TEXT]

gg pr list [--state open|closed|all] [--limit N]
gg pr view <number>
gg pr create [--title TEXT] [--body TEXT] [--base BRANCH] [--head BRANCH] [--draft]
```

Repository 명령:

```text
gg list [--limit N]        # 로그인한 내 계정의 repository 목록
gg view                    # 현재 remote(또는 --repo) repository 정보
gg --repo <new-repository-URL> create (--public|--private) [--description TEXT]
gg clone <repository-URL> [DIR]
gg pull [git-args...]
gg push [git-args...]
```

규칙:

- `--repo`가 없으면 현재 Git remote URL을 쓴다.
- `create`는 새 repository URL이 필요하므로 `--repo`가 필수다.
- `create`는 `--public` 또는 `--private` 중 하나가 필수다. forge마다 다른 기본 공개 범위를 쓰지 않는다.
- `clone`은 첫 인자의 URL로 provider를 판별한다.
- `pull`과 `push`는 남은 인자를 `git pull`과 `git push`에 그대로 전달한다.
- 알 수 없는 공통 flag는 오류다.
- provider 전용 기능은 `gh`, `glab`, `tea`를 직접 사용한다.

## Remote 선택

`--repo` 또는 `clone` URL이 없을 때 다음 순서로 Git remote를 고른다.

1. 현재 branch의 upstream remote
2. `origin`
3. remote가 하나뿐이면 그 remote
4. 그 외에는 오류

지원 URL 형식:

- `https://host/owner/repo.git`
- `ssh://git@host[:port]/owner/repo.git`
- `git@host:owner/repo.git`

URL parser는 다음 값을 만든다.

- 정규화한 lowercase host (port 제외; forge 식별과 설정 key로 사용)
- owner 또는 namespace path
- repository 이름
- 끝의 `.git`을 뺀 repository slug

잘못된 URL, 빈 host, repository path가 없는 URL은 오류다.

## Provider 판별

기본 domain은 바로 판별한다.

- `github.com` → `gh`
- `gitlab.com` → `glab`
- `gitea.com` → `tea`

Self-hosted host는 다음 순서로 판별한다.

1. `config.json`의 저장된 host 선택
2. 설치된 `gh`, `glab`, `tea`의 로그인 정보
3. 후보가 하나면 선택
4. 후보가 여러 개면 사용자 선택
5. 후보가 없으면 로그인 안내와 함께 오류

저장된 provider의 실행 파일이 없거나 해당 host 로그인이 사라졌으면 저장값을 무시하고 다시 판별한다.

### 로그인 정보 확인

- `gh`: `gh auth status`의 JSON host 정보
- `glab`: `glab auth status --hostname <host>` 결과
- `tea`: `tea logins list --output json` 결과

후보 확인 과정에서 token을 출력하거나 저장하지 않는다.

### 여러 후보 선택

stdin이 터미널이면 번호 선택 메뉴를 표시한다.

```text
Multiple providers match git.example.com:
1. gh
2. glab
Choose provider: 
```

선택한 값은 host별로 저장한다. 다음 실행부터 저장값을 먼저 쓴다.

stdin이 터미널이 아니면 기다리지 않는다. 후보 목록, 설정 파일 경로, JSON 수정 예시를 포함한 오류를 낸다.

## 설정 파일

설정 파일 이름은 `config.json`이다.

설정 root fallback 순서:

1. `$GG_HOME`
2. `$XDG_CONFIG_HOME/gg`
3. `~/.gg`

최종 경로 예:

```text
$GG_HOME/config.json
$XDG_CONFIG_HOME/gg/config.json
~/.gg/config.json
```

형식:

```json
{
  "hosts": {
    "git.example.com": "glab"
  }
}
```

허용 provider 값:

- `gh`
- `glab`
- `tea`

설정 저장 규칙:

- parent folder가 없으면 만든다.
- temp 파일에 먼저 쓴 뒤 rename한다.
- Unix에서는 folder `0700`, file `0600` mode를 요청한다.
- 손상된 JSON은 덮어쓰지 않는다. 파일 경로와 parse 오류를 낸다.
- token, username, repository URL은 저장하지 않는다.

## 명령 변환

### GitHub

- repository: `gh repo`
- issue: `gh issue`
- pull request: `gh pr`
- repository 대상: URL 또는 `[HOST/]OWNER/REPO`
- issue와 PR 대상: `-R [HOST/]OWNER/REPO`

### GitLab

- repository: `glab repo`
- issue: `glab issue`
- pull request: `glab mr`
- repository 대상: full URL
- issue와 MR 대상: `--repo <full-URL>`
- host만 필요한 명령은 `GITLAB_HOST=<host>`를 child environment에 넣는다.

### Gitea

- repository: `tea repos`
- issue: `tea issues`
- pull request: `tea pulls`
- clone: `tea clone`
- 선택한 login 이름과 `owner/repo`를 명령 option으로 넣는다.
- `view` subcommand가 없는 자원은 number 또는 slug를 바로 positional argument로 넣는다.

### 공통 flag 변환

| 공통 flag | GitHub | GitLab | Gitea |
|---|---|---|---|
| `--title TEXT` | `--title TEXT` | `--title TEXT` | `--title TEXT` |
| `--body TEXT` | `--body TEXT` | `--description TEXT` | `--description TEXT` |
| `--base BRANCH` | `--base BRANCH` | `--target-branch BRANCH` | `--base BRANCH` |
| `--head BRANCH` | `--head BRANCH` | `--source-branch BRANCH` | `--head BRANCH` |
| `--draft` | `--draft` | `--draft` | `--draft` |
| `--state open` | `--state open` | flag 없음 (glab 기본값) | `--state open` |
| `--state closed` | `--state closed` | `--closed` | `--state closed` |
| `--state all` | `--state all` | `--all` | `--state all` |
| `--limit N` | `--limit N` | `--per-page N` | `--limit N` |

GitLab list 명령에는 `--state`와 `--limit` flag가 없다. `gg`는 위 표대로
`--closed`/`--all`과 `--per-page`로 변환한다. 이 표는 `gh`/`glab` 로컬
`--help` 출력과 `tea` main branch 소스(`cmd/flags`)로 검증했다.

Repository create는 URL path를 provider 형식으로 바꾼다.

- GitHub: `gh repo create owner/repo`
- GitLab: `glab repo create <full-URL>`
- Gitea: `tea repos create --owner owner --name repo`

## Clone, pull, push

Clone:

- GitHub: `gh repo clone <URL> [DIR]`
- GitLab: `glab repo clone <URL> [DIR]`
- Gitea: `tea clone <URL> [DIR]`

Pull과 push:

- `gg pull [args...]` → `git pull [args...]`
- `gg push [args...]` → `git push [args...]`

Git 인증은 Git이 관리한다.

- SSH remote는 SSH key를 쓴다.
- HTTPS remote는 Git credential helper를 쓴다.
- `gg`는 `gh`, `glab`, `tea` token을 꺼내거나 Git에 주입하지 않는다.
- `gg`는 사용자의 global 또는 local Git 인증 설정을 바꾸지 않는다.

## Process 계약

자식 process는 현재 terminal에 직접 연결한다.

- stdin 상속
- stdout 상속
- stderr 상속
- 종료 signal 전달
- 자식 exit code 반환

`gg` 자체 exit code:

- `0`: 성공
- `1`: URL, remote, provider, 설정 read/write 등 실행 전 오류
- `2`: 잘못된 명령, flag, 필수 인자 등 사용법 오류
- `127`: 필요한 `git`, `gh`, `glab`, `tea` 실행 파일 없음
- 자식 process를 시작한 뒤에는 자식 exit code를 그대로 반환

## 오류 처리

주요 오류:

- Git 저장소가 아니고 `--repo`도 없음
- 사용할 remote를 하나로 고를 수 없음
- 지원하지 않는 URL
- provider 후보 없음
- 여러 provider 후보가 있으나 터미널 입력 없음
- 필요한 CLI 또는 Git이 PATH에 없음
- 손상된 설정 파일
- 지원하지 않는 명령 또는 flag
- 필수 인자 누락

오류는 다음을 포함한다.

- 실패한 위치
- 짧은 원인
- 가능한 경우 한 줄 수정 명령

자식 CLI와 Git의 stderr와 exit code는 바꾸지 않는다.

## 파일 구조

최소 구조:

```text
main.go
route.go
command.go
config.go
*_test.go
go.mod
docs/superpowers/specs/2026-08-18-gg-forge-routing-design.md
```

파일 역할:

- `main.go`: entrypoint, help, process exit
- `route.go`: remote 선택, URL parser, provider 판별
- `command.go`: 공통 command parser와 argv 변환
- `config.go`: 설정 경로, JSON read/write, 선택 저장

필요하지 않으면 package 분리는 하지 않는다.

## 테스트와 검증

단위 테스트:

1. HTTPS, SSH, SCP형 URL
2. port, `.git`, namespace path
3. 잘못된 URL과 빈 path
4. remote fallback 순서
5. 설정 root fallback 순서
6. 설정 JSON read, atomic write, 손상 파일 보호
7. 기본 domain 판별
8. 저장된 provider 판별
9. 로그인 후보 1개
10. 로그인 후보 여러 개 선택과 저장
11. 비대화형 여러 후보 오류
12. 모든 공통 명령의 provider별 argv 변환
13. Git pull/push 인자 전달
14. 자식 exit code 전달

End-to-end smoke test:

- temp PATH에 fake `git`, `gh`, `glab`, `tea` 실행 파일을 둔다.
- build한 `gg`를 실제 process로 실행한다.
- 선택된 실행 파일과 argv를 확인한다.
- stdout, stderr, exit code 전달을 확인한다.
- 사용자 선택 후 `config.json` 저장과 다음 실행 재사용을 확인한다.

검증 명령:

```text
go test ./...
go build ./...
GOOS=windows go build ./...
GOOS=linux go build ./...
GOOS=darwin go build ./...
```

현재 개발 PC에는 Go toolchain이 없다. 구현과 검증 전에 Go 설치가 필요하다.
