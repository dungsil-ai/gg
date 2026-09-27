# E2E 테스트

## 실행 경계

모든 테스트는 `tests/e2e`에서 실행합니다. 제품의 `internal/cli` 패키지를 가져오지 않으며, 내부 함수와 전역 변수를 직접 호출하거나 교체하지 않습니다. Go의 `testing` 패키지는 시나리오 실행과 결과 비교에만 사용합니다.

일반 명령은 실제 `gg` 실행 파일을 사용합니다. Release 파일 생성은 CI와 같은 `internal/cmd/package-release` 실행 파일을 사용하고, 릴리즈 게시 절차는 실제 workflow의 Bash 스크립트를 격리된 환경에서 실행합니다.

## 검증 범위

| 동작 | 관찰하는 결과 | 주요 테스트 파일 |
| --- | --- | --- |
| 명령어, 별칭, 도움말, 입력 오류 | stdout, stderr, 종료 코드, 잘못된 요청에서 자식 프로세스가 실행되지 않는지 확인합니다. | `commands_e2e_test.go`, `command_alias_e2e_test.go`, `usage_e2e_test.go` |
| 저장소 문맥과 Provider 판별 | 실제 Git remote의 우선순위, URL 형식, 저장된 Provider 설정, 로그인 후보, 미지원 버전을 확인합니다. | `e2e_test.go`, `routing_e2e_test.go` |
| Provider 설정 | 저장 위치 우선순위, 파일 재조회, 호스트 정규화, 손상된 파일 보존, 동시 변경을 확인합니다. | `config_e2e_test.go`, `e2e_test.go` |
| clone | 저장소 조회 결과, 환경 변수와 설정, HTTP 차단, 여러 계정, 인증정보가 오류에 노출되지 않는지 확인합니다. | `clone_e2e_test.go` |
| Forge 명령 | Provider CLI에 전달되는 인자와 환경 변수, 응답 변환, 실패 시 후속 호출 차단을 확인합니다. | `core_commands_e2e_test.go`와 명령별 `*_e2e_test.go` |
| Git 전달 명령 | 인자 경계, 표준 입출력, 종료 코드, Unix 시그널 전달을 확인합니다. | `e2e_test.go`, `child_exit_unix_test.go`, `child_sigint_unix_e2e_test.go` |
| 이력 재작성 | 실제 파일·커밋·태그·원격 추적 ref·백업, dry-run의 무변경, 잘못된 스트림과 자식 프로세스 실패를 확인합니다. | `filter_repo_e2e_test.go`, `filter_contract_e2e_test.go` |
| 릴리즈 | 실제 로컬 원격 저장소의 커밋·태그, 6종 Release 파일, 체크섬, 바이너리 버전, 게시 전 실패 조건을 확인합니다. | `release_prepare_e2e_test.go`, `release_e2e_test.go`, `release_workflow_e2e_test.go` |

## 격리와 외부 의존성

Git 전역 설정은 테스트 전용 파일을 사용합니다. 저장소, Provider 설정, 실행 파일은 임시 디렉터리에 만들며, Git의 원격 접근은 로컬 파일 프로토콜로 제한합니다. 공통 실행 도우미의 PATH에는 CLI 대역과 실제 Git으로 전달하는 실행 파일만 둡니다.

Forge CLI 대역은 별도 실행 파일이며 요청 인자를 JSON 배열로 기록합니다. 필요하면 호출별 응답과 실패를 지정합니다. 이 대역은 `gg`가 외부 CLI와 맺는 실행 계약을 검증하기 위한 것으로, 실제 GitHub·GitLab·Gitea 서버의 동작이나 특정 CLI 버전과의 호환성을 검증하지는 않습니다.

Release 파일 생성 테스트는 실제 교차 컴파일을 수행하므로 다른 시나리오보다 오래 걸립니다. Linux 전용 시그널 테스트와 Windows에서 사용할 수 없는 파일명 사례는 해당 OS에서만 실행합니다.

## 테스트 추가 원칙

새 동작은 사용자 명령과 관찰 가능한 결과를 기준으로 작성합니다. 내부 자료구조나 명령 레지스트리에서 기대값을 가져오지 않습니다. 오류 사례에서는 종료 코드와 오류 메시지뿐 아니라 파일·Git ref 보존, 후속 CLI 호출 차단 등 필요한 부작용도 확인합니다.

내부 함수에 대한 유닛 테스트나 fuzz 테스트는 추가하지 않습니다. 입력 경계와 회귀 사례도 실행 파일을 통과하는 E2E 시나리오로 작성합니다. 전체 검증 명령은 `go test -count=1 -timeout=20m ./...`이며, 특정 이름만 선택하는 필터를 CI의 전체 검증에 사용하지 않습니다.
