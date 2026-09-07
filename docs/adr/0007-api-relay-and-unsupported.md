# API 릴레이와 미중계 항목 방침

README TODO의 `gh api`, `glab api`, `tea api`는 각 provider CLI의 원시 API 접근 명령이고, `tea logins`는 tea CLI 자신의 로컬 인증 설정 명령이다. 이 문서는 이 항목들과 나머지 플랫폼 고유 명령(codespace, project, secret, gist 등)의 중계 방침을 정한다.

## gg api: 원시 passthrough

`gg api [args...]`는 action 뒤의 모든 인자를 검사 없이 provider의 `api` 하위 명령에 전달한다. 인자 모델은 git passthrough(ADR 0004)와 같고, provider 선택은 저장소 문맥(DetectProvider)으로 한다.

호스트 전달 방식은 기존 api 사용처(issue comment list/edit/delete)의 계약을 따른다. gh는 host가 `github.com`이 아니면 `GH_HOST` env를, glab은 host가 `gitlab.com`이 아니면 `GITLAB_HOST` env로 주입한다. tea는 다른 명령과 같이 저장소 문맥에서 얻은 `--login`과 `--repo`를 붙인다.

검토한 다른 방식 두 가지는 기각했다. (1) `--repo`/`--remote` 같은 저장소 문맥 flag를 api 인자 사이에서 파싱하는 방식은 원시 API 인자와 충돌할 수 있어 passthrough 계약을 깬다. (2) endpoint를 gg가 파싱해 표준화하는 방식은 세 provider의 API 차이를 gg가 추상화해야 하므로 범위가 과도하다.

`--help`를 포함한 모든 인자가 전달되므로 `gg api --help`는 `gh api --help`를 실행한다. 이는 git passthrough와 같은 예외다.

tea의 `api` 하위 명령은 `--login`이 필요하므로 저장소 문맥의 login 조회를 실패하면 다른 tea 명령과 같이 오류를 낸다.

## glab auth login/logout: 보류 유지

`gg auth` 릴레이(#103)는 ADR 0006에 따라 저장소 문맥 없이 동작하므로, glab 로그인 명령을 실행할 provider를 고를 수단이 없다. provider를 고르는 방식(예: `--provider` flag, Provider 설정의 기본 provider)은 auth 릴레이 확장 시 별도 ADR로 다룬다. 그 전까지 `glab auth login/logout`은 보류다.

## tea logins: 미지원 확정

`tea logins add/delete/list/view`는 tea CLI 자신의 로컬 인증 설정(`~/.config/tea/config.yml`)을 관리한다. gg는 provider 로그인 정보를 저장하지 않고(ADR 0002), Provider 설정은 host→provider 매핑만 가진다(`gg config`). auth status는 `tea logins list`를 조회 함수로 사용하지만, 설정 자체의 생성·삭제는 각 CLI의 고유 영역이다. 따라서 `gg logins` 자원을 만들지 않고 해당 항목은 미지원으로 확정한다.

## codespace·project·secret·gist 등 플랫폼 고유 자원

GitHub 고유 자원(codespace, project, secret, variable, gist, attestation 등)은 세 provider 간 공통 표면이 성립하지 않는다. gg는 공통 기능 추상화를 목표로 하므로(README TOBE), 자원별 gg 표면 설계는 개별 수요가 확인될 때 별도 ADR로 다룬다. 그 전까지 README의 해당 항목은 미체크를 유지한다.

## 영향

- `gg api`는 위 passthrough 계약대로 구현하고, README의 `gh api`·`glab api`·`tea api` 항목을 체크한다.
- `tea logins` 5개 항목은 README에 미지원 사유를 명시한다.
- `glab auth login/logout`과 플랫폼 고유 자원은 보류 사유를 README에 명시한다.
