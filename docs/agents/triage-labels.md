# Labels

Skill은 canonical role을 쓴다. 실제 GitHub label은 아래 표를 쓴다.

## Triage 상태

| Canonical Role | GitHub Label | 뜻 |
| --- | --- | --- |
| `needs-triage` | `상태:분류필요` | maintainer가 요청을 분류해야 함 |
| `needs-info` | `상태:정보필요` | reporter의 추가 정보가 필요함 |
| `ready-for-agent` | `상태:에이전트작업` | spec이 끝났고 agent가 구현할 수 있음 |
| `ready-for-human` | `상태:사람작업` | 사람이 직접 구현해야 함 |
| `wontfix` | `상태:처리안함` | 구현하지 않음 |

## 계획 label

| Canonical Role | GitHub Label | 뜻 |
| --- | --- | --- |
| `map` | `상태:초안` | 실행 ticket이 아닌 decision map |
| `research` | `유형:조사` | source 조사로 답을 찾음 |
| `prototype` | `유형:프로토타입` | 버릴 수 있는 prototype으로 확인함 |
| `grilling` | `유형:인터뷰` | 질문을 한 번에 하나씩 물어 결정함 |
| `task` | `유형:작업` | 결정보다 먼저 해야 하는 수동 작업 |

## 두 축 규칙

`상태:`는 현재 상태를 뜻한다. `유형:`은 결정 방법을 뜻한다. 한 Issue에는 각 축에서 label을 최대 하나만 붙인다.

`상태:초안`은 다른 `상태:` label과 함께 쓰지 않는다. decision ticket은 `유형:` label만 쓴다. 계획이 끝나고 구현 ticket이 생기면 다시 triage 상태 label을 쓴다.
