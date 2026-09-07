# auth status 계약

`gg auth status`는 provider CLI의 로그인 상태를 한 표로 조회하는 읽기 전용 명령이다. 조회는 provider 자동 감지가 이미 쓰는 `hasLogin` 함수군(`ghHasLogin`, `glabHasLogin`, `teaLoginName`)을 그대로 노출하므로 새 파싱을 추가하지 않는다. 조회 함수가 오류를 조용히 삼키고 false를 반환하는 현재 계약도 이 ADR은 바꾸지 않는다.

## 조회 대상

조회 대상 host는 Provider 설정에 저장된 host와 기본 domain(`github.com`, `gitlab.com`, `gitea.com`)을 합친 목록이다. host의 Provider는 기본 domain이면 고정 mapping을, 저장된 host면 `gg config set`의 값을 쓰므로 host당 provider CLI 호출이 정확히 한 번이다. `gg config list`와 마찬가지로 host 기준으로 정렬한다. 기본 domain이 항상 목록에 있으므로 조회할 host가 없는 상태는 없고, `gg config list`의 `No provider settings.`에 해당하는 문구도 필요 없다.

기본 domain 3개만 고정 조회하는 방안은 self-hosted host의 로그인을 보여주지 못하고, 현재 저장소 문맥의 host만 조회하는 방안은 저장소 밖에서 아무것도 보여주지 못한다. 로그인 상태 진단은 저장소와 무관한 질문이므로 후자는 성립하지 않는다.

## 출력

`gg config list`의 `HOST`/`PROVIDER` 표 관례를 `LOGIN` 열로 확장한다.

```text
$ gg auth status
HOST             PROVIDER  LOGIN
git.example.com  tea       my-login
gitea.com        tea       no cli
github.com       gh        dungsil
gitlab.com       glab      yes
```

LOGIN 값 범위는 로그인 이름(`gh auth status --json hosts`와 `tea logins list --output json`에서 확인), `yes`(glab처럼 로그인은 확인되지만 이름을 제공하지 않을 때), `no`(CLI가 로그인 없음을 보고), `no cli`(provider CLI 미설치, `hasBin`으로 먼저 걸러 내어 조회를 시도하지 않음)다.

## exit code

`gg pr status`의 계약을 따른다. 조회 자체가 실패할 때만 0이 아닌 exit code를 내고, 행별 로그인 여부는 결과 값일 뿐이다.

| 상황 | exit code |
| --- | --- |
| 표 조회 성공 — 모든 행이 `no`나 `no cli`여도 포함 | 0 |
| 조회 자체 실패(예: 손상된 config.json 읽기) | 1 |

`gh auth status`처럼 미로그인을 non-zero로 보고하는 방식도 검토했지만, 표 조회가 성공해도 실패가 섞이므로 호출자가 `|| true`를 붙이게 되고, 미로그인도 상태 값으로 다루는 `gg pr status`의 기존 계약과 어긋난다. `gg pr status`는 조회 대상 provider가 하나라서 CLI 미설치를 127로 보고하지만, auth status는 행별 독립 조회이므로 한 행의 미설치가 전체를 실패시키지 않는다.

## 성능과 신뢰성

조회에 타임아웃을 두지 않는다. DetectProvider와 forge 명령의 자식 실행이 모두 타임아웃 없이 자식을 기다리므로 status만 다른 정책을 두면 같은 provider CLI가 문맥에 따라 다르게 실패한다. 느린 provider CLI 문제는 조회 경로 전체의 공통 정책으로 따로 다룬다. `glabHasLogin`이 exit code만 보고 `ghHasLogin`과 `teaLoginName`이 JSON을 파싱하는 구현 차이도 그대로 둔다.

## flag 경계

auth는 forge 명령이 아니므로 `--repo`, `--remote`, `--explain`을 받지 않는다. 저장소 문맥 flag는 Forge 대상 저장소를 고르는 수단인데 auth status의 조회 대상은 저장소가 아니라 Provider 설정과 기본 domain 목록이다. ADR 0004가 Git 전달 명령의 앞 context flag를 거절한 것과 같은 경계이고, `gg config list`도 같은 이유로 문맥 flag가 없다.

## 나머지 auth 명령

이 ADR은 `gg auth status`만 확정한다. README 체크리스트의 나머지 auth 명령은 두 갈래로 나뉜다. `logout`, `refresh`, `setup-git`, `switch`, `token`은 provider CLI 고유의 쓰기 동작이라 gg가 집계할 값이 없는 중계 후보다. `login`은 중계와 gg-native 집계 사이에 있다: 자식 stdio 보존으로 대화형 흐름의 중계 자체는 가능하지만, 어떤 provider CLI를 실행할지 고르려면 host와 provider의 규칙이 로그인보다 먼저 필요하다. DetectProvider는 로그인된 후보 가운데 provider를 고르므로 로그인 전에는 이 규칙이 성립하지 않는다. status는 이 문제 없이 기존 host 규칙(기본 domain 고정 + Provider 설정)만으로 동작하므로 status를 먼저 확정한다.

JSON 출력은 제공하지 않는다. ADR 0002가 공통 JSON 출력을 현재 범위 밖으로 유보했으므로 auth status가 개별 JSON 계약을 먼저 만들지 않고, 공통 JSON 계약이 정해지면 그때 따라간다.

## 미해결 질문

- `no`는 미로그인과 조회 실패(네트워크 오류, 출력 파싱 실패)를 구분하지 않는다. 구분하려면 `hasLogin`이 오류 원인을 반환하도록 계약을 바꿔야 하고 이는 DetectProvider에도 영향을 준다. 현 계약으로도 위의 표 계약은 성립하므로, 필요성이 확인되면 별도 계획으로 검토한다.
- glab의 로그인 이름은 `glab auth status`의 비안정 텍스트 출력을 파싱해야만 얻는다. `yes` 표기로 충분한지, 파싱을 도입할지는 실구현 계획에서 검토한다.
- `gg auth login` 중계에서 self-hosted host는 "먼저 `gg config set <host> <provider>`" 안내로 충분한지, login 전용 흐름이 필요한지는 별도 계획으로 결정한다. 기본 domain은 고정 mapping이 있어 이 문제가 없다.
