# auth 릴레이의 provider 선택

## 문제

`gg auth`의 릴레이 action(ADR 0006)은 GitHub CLI 계정 작업을 `gh`에 그대로 전달한다. self-hosted GitLab 사용자의 온보딩 첫 단계인 `glab auth login`은 gg 밖에서만 실행 가능하며, ADR 0007은 이를 "provider를 고르는 방식 설계 후 별도 ADR"로 보류해 왔다. 이 문서가 그 설계다.

## 결정

릴레이 action(login, logout, refresh, setup-git, switch, token)에 `--provider <gh|glab>` flag를 추가한다.

- 위치는 action 뒤 어디든 좋고(`gg auth login --provider glab ...`, `gg auth token --provider=glab`), `"--provider x"`와 `"--provider=x"` 형태를 모두 받는다. gg가 해석해 제거한 나머지 인자는 검사 없이 대상 CLI에 전달된다(passthrough 계약 유지).
- 생략하면 `gh`다. 기존 호출은 전혀 바뀌지 않는다.
- `--provider glab`은 glab에 실제로 존재하는 하위 명령만 중계한다: `login`, `logout`, `token`. `status`는 gg-native 명령이고, `refresh`·`setup-git`·`switch`는 glab에 없으므로 사용법 오류로 거부한다.
- `--provider tea`는 거부한다. tea logins는 tea CLI 자신의 로컬 인증 저장소라는 ADR 0007의 미지원 확정을 따른다.
- `auth status`는 gg-native이므로 `--provider`를 받지 않는다(파서가 passthrough 밖에서 이미 거부한다).

## 기각한 대안

1. **Provider 설정에 기본 auth provider를 추가하는 방식** — ADR 0002의 Provider 설정은 host→provider 매핑만 가지며, auth는 host가 없는 질문이다. host 없는 기본값을 설정에 넣으면 상태가 늘고 의미가 모호해진다. 또한 로그인·로그아웃은 부작용이 큰 작업이라 설정에 숨겨진 기본값으로 대상 CLI가 바뀌는 것은 위험하다.
2. **현재 저장소 문맥이나 기존 로그인으로 provider를 자동 감지하는 방식** — auth는 저장소 문맥이 없는 명령이다(ADR 0006). 감지는 암묵 선택이며, gh에 로그인된 상태에서 `gg auth login --hostname gitlab.example.com`이 실수로 gh를 고르는 사고를 만든다.
3. **glab 전용 최상위 명령을 새로 만드는 방식** — auth 어휘는 이미 `gg auth`에 모여 있고 namespace를 늘 이유가 없다(ADR 0005가 `gg prepare`를 기각한 것과 같은 이유).

## 미해결 질문

없다. tea v1.x(리디자인 CLI)가 자체 인증 표면을 새로 제공하면 이 ADR을 확장해 다룬다.
