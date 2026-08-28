# Issue Tracker: GitHub

이 저장소의 Issue와 PRD는 GitHub Issues에서 관리한다. 모든 작업은 `gh` CLI를 사용한다.

## 기본 규칙

- Issue 생성: `gh issue create --title "..." --body "..."`
- Issue 읽기: `gh issue view <number> --comments`
- Issue 목록: `gh issue list --state open --json number,title,body,labels,blockedBy,subIssues`
- 댓글 작성: `gh issue comment <number> --body "..."`
- label 추가와 제거: `gh issue edit <number> --add-label "..."` 또는 `--remove-label "..."`
- Issue 닫기: `gh issue close <number> --comment "..."`

저장소는 `git remote -v`에서 찾는다. 이 clone 안에서 `gh`를 실행하면 현재 저장소가 자동으로 선택된다.

## PR 분류 범위

**PR을 요청 분류 surface로 사용하지 않는다.** 외부 PR을 자동 분류 목록에 넣지 않는다. 사용자가 PR을 직접 지정하면 해당 PR은 읽고 처리할 수 있다.

GitHub는 Issue와 PR이 같은 번호 영역을 쓴다. `#42`가 모호하면 먼저 `gh pr view 42`를 확인하고, 실패하면 `gh issue view 42`를 확인한다.

## Skill 용어

- Skill이 "Issue Tracker에 게시"하라고 하면 GitHub Issue를 만든다.
- Skill이 "관련 ticket을 읽기"라고 하면 `gh issue view <number> --comments`를 실행한다.

## Spec과 ticket 구조

- Spec은 parent Issue다.
- ticket이 둘 이상이면 각 ticket을 spec의 GitHub native sub-issue로 만든다.
- child Issue를 만든 뒤 `gh api repos/<owner>/<repo>/issues/<child> --jq .id`로 numeric database id를 구한다.
- `gh api --method POST repos/<owner>/<repo>/issues/<parent>/sub_issues -F sub_issue_id=<child-db-id>`로 parent에 연결한다.
- 연결 뒤 sub-issue 목록을 다시 읽고 child 번호가 있는지 확인한다.
- GitHub API가 sub-issue를 지원하지 않는다고 명확히 답할 때만 parent task list와 `Part of #<parent>` 문구를 대신 쓴다.

## Blocker

- GitHub native issue dependency를 쓴다.
- `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`로 blocker를 연결한다.
- blocker id는 Issue 번호나 `node_id`가 아니라 numeric database id다.
- native dependency를 쓸 수 없으면 child body의 `Prerequisites`에 `Blocked by: #<number>`를 적는다.
- 열린 blocker가 모두 닫혀야 ticket이 시작 가능하다.

## 계획 map

- `/vibe-deep-plan` map은 `상태:초안` label이 붙은 하나의 Issue다.
- decision ticket은 `유형:조사`, `유형:프로토타입`, `유형:인터뷰`, `유형:작업` 중 하나를 쓴다.
- map과 ticket 연결은 위 native sub-issue 규칙을 따른다.
- 조사 결과는 같은 조사 Issue의 전용 comment에 source와 함께 남긴다.
- 저장과 연결을 다시 확인하기 전에는 성공으로 보고하지 않는다.
