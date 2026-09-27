# Repository Guidelines

## 테스트 원칙

- 자동 테스트는 `tests/e2e`의 E2E 테스트만 사용하며, 테스트 함수 이름은 `TestE2E`로 시작합니다. 내부 함수에 대한 유닛 테스트나 fuzz 테스트는 추가하지 않습니다.
- 빌드한 `gg`를 별도 프로세스로 실행해 검증합니다. Release 파일 생성은 `internal/cmd/package-release` 실행 파일을, 릴리즈 게시 절차는 실제 workflow의 Bash 스크립트를 사용합니다.
- 테스트에서 제품 패키지를 직접 가져오거나 내부 함수·전역 변수를 호출 또는 교체하지 않습니다. 기대값은 사용자에게 공개된 동작을 기준으로 정하고, 제품의 자료구조나 명령 레지스트리에서 가져오지 않습니다.
- 새 기능, 버그 수정, 입력 경계와 회귀 사례도 E2E 시나리오로 검증합니다. 표준 입출력과 종료 코드뿐 아니라 파일·Git ref의 변경 또는 보존, 실패 시 후속 CLI 호출 차단 등 필요한 부작용을 확인합니다.
- Git 동작은 실제 임시 저장소와 로컬 bare 원격 저장소로 검증합니다. Forge 연동은 별도 실행 파일인 CLI 대역으로 인자·환경 변수·표준 입출력·종료 코드를 검증하며, 실제 Forge 서버나 사용자 인증 계정을 사용하지 않습니다.
- 기존 E2E 도우미를 재사용해 임시 디렉터리, Git 설정, Provider 설정과 PATH를 격리합니다. 사용자 환경에 의존하거나 사용자 저장소를 변경하는 테스트를 만들지 않습니다.
- 공통 동작은 Windows와 Linux에서 검증합니다. 셸 대역의 제약만으로 공통 시나리오를 건너뛰지 않으며, 시그널처럼 OS에 종속된 동작만 해당 OS로 제한합니다. 교차 컴파일 성공을 실제 실행 성공으로 보고하지 않습니다.
- 동작에 영향을 주는 변경은 `go vet ./...`와 `go test -count=1 -timeout=20m ./...`로 검증합니다. 개발 중에는 특정 시나리오를 선택해 실행할 수 있지만, 이를 전체 검증 결과로 보고하지 않습니다. 문서만 변경한 경우에는 내용 정합성과 변경 내역을 확인하며, 전체 테스트를 불필요하게 반복하지 않습니다.
- 상세한 검증 범위와 실행 방법은 [테스트 안내](docs/testing.md)를 따릅니다.

## Commit Message Convention

- Follow the Conventional Commits format: `<type>(<scope>): <subject>`
- The allowed `type` values are `feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `build`, `ci`, `chore`, and `revert`.
- Write the `subject` in Korean as a declarative sentence, limit it to 80 characters, and do not end it with a period.
- The `scope` is optional. Use an English lowercase module or domain name (for example, `auth`, `api`, or `ui`).
- Do not include issue numbers in the commit title.
- In the `body`, explain why the change is necessary instead of listing files, and wrap lines at 120 characters. Omit the `body` when the `subject` is sufficient.
- For a breaking change, add `!` after the `type` (for example, `feat!:`) and explain the migration procedure in the `body`.
- A `revert` commit must include the original commit hash in the `body`.

## Pull Request Convention

- Follow [Commit Message Convention](#commit-message-convention) for the PR title.
- Write the PR body in Korean.
- Organize the PR body in this order: `요약`, `수정 내역`, `검증 사항`, `Ref`, and `Closes`.
- In `요약`, concisely explain the purpose of the PR and its main changes.
- In `수정 내역`, list the actual changes by item.
- In `검증 사항`, describe the verification methods performed and the results confirmed.
- In `Ref`, provide related issues, PRs, documents, or other references.
- In `Closes`, identify each issue closed by the PR using `Closes #<issue-number>`.
- Omit sections that have no applicable content.

### Example

PR title:

```text
feat(agent): PR 작성 규칙 추가
```

PR body:

```markdown
## 요약

공통 에이전트 지침에 PR 작성 규칙을 추가합니다.

## 수정 내역

- PR 제목 및 본문 작성 규칙 추가
- duninit 워크플로우에 PR 규칙 반영 단계 추가

## 검증 사항

- PR 규칙이 기존 지침과 중복되지 않음을 확인
- 문서가 마지막 줄바꿈으로 끝나는지 확인

## Ref

- #10

## Closes

Closes #11
```
