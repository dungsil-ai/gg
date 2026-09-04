# release prepare 계약

`gg release prepare <version>`은 ADR 0001이 정한 릴리즈 의식 — 깨끗한 트리에서의 전용 빈 릴리즈 커밋, 그 `HEAD`의 annotated tag, 커밋과 tag의 같은 원격 transaction atomic push — 를 검사하고 실행하는 gg 고유 명령이다. 이 절차를 손으로 실행하면 커밋과 tag를 만들어 두고 push를 잊은 채 하루를 보내거나, `--atomic` 없이 push해 커밋만 원격에 올라간 partial push를 남기거나, release 커밋이 아닌 `HEAD`에 tag를 찍거나, 이미 공개한 tag를 덮어쓰는 되돌리기 어려운 사고가 모두 가능하다. 계약의 모든 검사점은 기계적으로 판정 가능하므로 명령으로 흡수해 maintainer의 실수 여지를 없앤다. 이 ADR은 계약을 확정하는 스파이크의 결론이며, 실구현은 이 결론을 받은 별도 계획으로 진행한다.

## 계약

`prepare`는 변경을 하나도 일으키기 전에 아래 사전조건을 순서대로 모두 검사하고, 하나라도 실패하면 아무 것도 만들지 않고 실패한다(fail-closed). 검사 순서는 사용법 → tag 형식 → working tree → 현재 branch → 로컬 tag 부재 → 원격 대조다. 원격 조회는 로컬 검사가 모두 통과한 뒤에만 실행한다.

1. 사용법: positional 인자는 tag 문자열 하나뿐이다. `version` 인자는 완전한 tag 그 자체이고 `v` 접두사를 생략하거나 자동 보완하지 않는다.
2. tag 형식: `^v[0-9]+\.[0-9]+\.[0-9]+$`와 일치해야 한다. 이것은 release workflow의 dispatch·verify 단계가 이미 요구하는 형식과 같다. prepare가 만든 tag는 workflow 게이트를 통과할 수 있어야 하므로 그보다 넓은 형식을 받지 않는다.
3. working tree가 깨끗하다: `git status --porcelain`의 출력이 빈 문자열이다.
4. 현재 branch가 default branch다: `git symbolic-ref refs/remotes/origin/HEAD`로 얻은 branch와 현재 branch가 같다. 이 조회가 실패하면(detached HEAD, origin/HEAD 미설정) 검사 실패로 처리하고 `git remote set-head origin --auto` 실행을 안내한다.
5. 로컬에 같은 tag가 없다: `git show-ref --verify --quiet refs/tags/<tag>`가 실패해야 한다.
6. `HEAD`가 원격 default branch의 현재 tip과 같다: `git rev-parse HEAD`와 `git ls-remote origin refs/heads/<default>`의 값이 같다. prepare는 이 tip 위에 빈 커밋을 올리므로 이 검사가 있어야 ADR 0001의 workflow 게이트(태그 커밋 == 원격 default branch tip)를 보증할 수 있다.
7. 원격에 같은 tag가 없다: `git ls-remote --exit-code --tags origin refs/tags/<tag>`가 실패해야 한다. 조회 자체의 실패(네트워크 오류 등)는 tag 부재로 취급하지 않고 실패한다.

모든 검사를 통과하면 gg는 자식 프로세스로 순서대로 실행하고 각 단계의 stdout, stderr, exit code를 그대로 보존한다. 이는 Git 전달 명령의 실행 계약(ADR 0004)을 재사용한다.

```text
git commit --no-gpg-sign --allow-empty -m "release: <tag>"
git tag -a <tag> -m "Release <tag>"
git push --atomic origin HEAD:refs/heads/<default> refs/tags/<tag>
```

커밋 서명은 ADR 0004의 non-signing 정책을 따라 `--no-gpg-sign`을 넣는다. 이는 `gg commit`이 내부적으로 하는 일과 같으므로, prepare의 커밋은 기존 수동 경로의 커밋과 동일하다. 원격이 atomic push를 지원하지 않으면 명령은 실패하며 non-atomic fallback이나 force push는 쓰지 않는다(ADR 0001). 자식이 중간에 실패하면 그 자리에서 멈추고 자동 rollback(commit·tag 삭제)을 하지 않는다. 대신 어디까지 성공했는지 stderr에 한 줄로 남기고, push 실패 시 같은 refspec의 atomic push를 다시 실행하거나 로컬 tag와 커밋을 지운 뒤 prepare를 다시 실행하는 복구 방법을 안내한다.

라우팅은 `gg config`의 선례를 따른다. `run.go`에서 forge resolve 이전에 우회하는 gg-native action이고 provider를 조회하지 않으며 로컬 git과 원격 git transport만 쓴다. `release prepare`는 기존 `gg release` resource의 action으로 파싱하지만 provider builder table에는 두지 않는다. CONTEXT.md의 용어로 이 명령은 "Git 전달 명령"이 아니다 — git 인자를 그대로 전달하지 않고 검사를 추가하고 여러 git 명령을 합성하므로 README의 "gg 고유 기능" 분류에 둔다. 전역 flag 계약은 Git 전달 명령과 다르다: `--repo`와 `--remote`는 UsageError와 exit code 2로 거부하고 `--explain`은 허용한다. 명령은 현재 작업 트리의 `HEAD`를 커밋하고 push하므로 저장소 문맥을 다른 URL이나 remote로 바꾸는 것은 무의미하고 위험하다. ADR 0004의 flag 거부는 Git 전달 명령 한정이므로 이 차이는 충돌이 아니다.

`--explain`은 기존 관례("이 조회를 실행하지 않는다")대로 아무 검사도 실행하지 않는 정적 예고다. 선택된 tag, 검사 목록, 실행할 git 명령 세 줄을 stdout에 출력하고 exit 0으로 끝낸다.

### 종료 코드와 메시지 계약

`gg pr status`의 exit code 계약 문서화 스타일을 따른다. 검사 실패와 usage 오류는 stderr에 `gg: ` 접두사의 한 줄 메시지를 내고, 메시지는 검사 이름과 실패한 구체적 값, 가능하면 다음 행동을 함께 넣는다(`유효하지 않은 release tag: %q` 선례).

| 상황 | exit code | 메시지 계약 |
|---|---|---|
| 인자가 0개 또는 2개 이상, 알 수 없는 flag | 2 | `usage: gg release prepare <tag>` |
| 명령 앞 `--repo`/`--remote` | 2 | `--repo is not supported for release prepare` 형식 (ADR 0004의 거부 문구 선례) |
| tag 형식 불일치 | 1 | 받은 tag와 기대 형식 `vMAJOR.MINOR.PATCH`를 함께 출력 |
| working tree가 지저분함 | 1 | `git status`로 확인하라는 안내 포함 |
| 현재 branch가 default branch가 아님 | 1 | 현재 branch와 기대 branch를 함께 출력 |
| origin/HEAD 조회 실패 (detached HEAD 포함) | 1 | `git remote set-head origin --auto` 안내 포함 |
| HEAD가 원격 default tip과 다름 | 1 | 두 SHA를 함께 출력 |
| 로컬·원격에 같은 tag가 이미 있음 | 1 | tag 재사용·이동 금지를 명시 (ADR 0001) |
| 원격 tag·tip 조회 자체가 실패 | 1 | 조회 실패를 tag 부재로 취급하지 않는다고 명시 |
| `git commit`/`git tag`/`git push` 실패 | 자식의 exit code | 자식의 stdout/stderr를 그대로 통과하고 gg는 진행 상황과 복구 방법을 stderr에 한 줄 더한다 |
| git 미설치 | 127 | `gg: git is not installed or not on PATH` (execChild 선례) |
| 자식이 신호로 종료 | 128+신호 | execChild 선례 |
| 성공 | 0 | 검사와 실행 진행을 stdout에 순서대로 한 줄씩, 마지막에 push한 refspec을 출력 |

## 검토했다

1. **명령 표면**: 검사·커밋·tag·push를 한 명령으로 묶고 push 분리 flag은 두지 않기로 했다. 검사+커밋+tag까지만 하고 push를 분리하면 "로컬에는 커밋과 tag가 있고 원격에는 없는" 중간 상태가 생긴다. 이 상태는 partial push만큼 위험한 미완료 릴리즈이고, 다음 실행은 로컬 tag 부재 검사에 막혀 복구 절차가 오히려 복잡해진다. `--push`를 두어 push를 선택으로 만드는 방안도 같은 이유로 기각했다 — 기본값이 조용한 부분 상태가 된다. ADR 0001이 3단계를 하나의 재현 가능한 사용자 경로로 묘사하므로 명령도 한 번에 끝낸다. 실패 지점이 늦어지는 비용은 모든 검사가 변경 전에 실행되므로 흡수된다.
2. **tag 형식**: `v` 접두사를 강제하고 접미사는 받지 않기로 했다. `internal/cli/release.go`의 `validReleaseTag`는 `^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`로 `v`를 선택으로 받는데, 이 정규식의 목적은 ldflags 주입으로부터 링커 인자를 보호하는 것이므로 넓게 받아도 안전하다. 반면 prepare의 목적은 공개 tag 계약의 집행이고 release workflow의 dispatch·verify 단계는 `^v[0-9]+\.[0-9]+\.[0-9]+$`로 엄격하다. `validReleaseTag`를 재사용하면 workflow가 거절할 접미사 tag를 만들어 게이트에서 반드시 실패하는 릴리즈를 만들 수 있으므로 기각했다. `validReleaseTag` 자체를 줄이는 방안도 기각했다 — 빌드 경로의 동작 변경은 이 스파이크 범위를 넘고 ldflags 보호 목적에는 불필요하다. 실구현은 별도의 엄격한 검증을 둔다. README 42행의 "선택적 접미사 허용"과 이 결정의 간극은 미해결 질문 2에 기록했다.
3. **사전조건 순서와 방법**: 위 계약의 순서를 선택했다. default branch 확인은 로컬 조회인 `git symbolic-ref refs/remotes/origin/HEAD`로 한다. `git remote show origin`은 정확하지만 네트워크 왕복이 필요하고, forge API의 default branch 조회는 provider 중계를 gg-native 명령에 끼워 넣으므로 기각했다. `HEAD` == 원격 tip 검사는 계획의 4개 검사에 추가한 것이다 — 이것이 없으면 뒤처진 branch에서 만든 tag 커밋이 workflow 게이트에서 반드시 실패하고, 앞선 branch에서는 maintainer가 의도하지 않은 커밋들이 릴리즈 커밋과 함께 공개된다. 생략하고 push의 non-fast-forward 거부에 의존하는 방안은 실패 메시지가 릴리즈 맥락을 설명하지 못하므로 기각했다. exit code는 기존 관례(usage 오류=2, 조회·검사 실패=1, 하위 CLI 미설치=127, 신호 종료=128+신호)를 그대로 따른다.
4. **실행 동작**: 자식 프로세스 실행은 `execChild`의 stdio·exit code 보존 계약을 재사용한다. 서명은 ADR 0004를 따라 `--no-gpg-sign`을 넣는다. 사용자의 `commit.gpgSign` 설정을 존중하는 방안은 같은 의식의 두 경로(`gg commit`과 prepare)가 환경에 따라 다르게 성공하는 불일치를 만드므로 기각했고, `-S`로 명시 서명을 요구하는 방안은 ADR 0001이 v0.x 범위에서 GPG를 제외한 것과 맞지 않아 기각했다. 실패 시 자동 rollback은 하지 않기로 했다 — ref 자동 삭제 자체가 또 다른 변경이고 실패 모드를 늘린다. 부분 상태는 로컬에만 남고 원격 partial push는 `--atomic`이 막으므로, 성공 단계를 명시하고 복구 방법을 안내하는 것이 fail-closed에 더 가깝다.
5. **gg-native 라우팅**: `config`처럼 forge resolve 이전에 우회한다. config의 우회 지점은 explain 블록 앞이므로 prepare 핸들러가 `req.Explain`을 직접 처리한다(위 계약 참고). `--remote`로 push 대상 remote를 고르게 하는 방안은 기각했다 — ADR 0001의 의식은 `origin`에 고정되어 있고 검사·메시지·문서를 origin 기준으로 단순하게 유지한다. multi-remote 수요가 생기면 별도 계획으로 다룬다. 새 최상위 명령 `gg prepare`도 기각했다 — release 어휘는 이미 `gg release` resource에 모여 있고 최상위 namespace를 늘 이유가 없다.
6. **workflow dispatch 경로와의 관계**: 현재 release.yml의 dispatch 경로는 tag 형식·원격 tag 부재·HEAD==default tip·커밋 제목·빈 커밋을 검사한 뒤 `github-actions[bot]` 명의로 tag를 만들고 push한다. 두 이벤트 모두 이어지는 verify 단계(annotated tag, Release 부재, Immutable Releases)를 통과하므로 불변성 검증 자체가 우회되지는 않는다. 그래도 경로가 둘이면 같은 계약의 두 구현이 서로 drift할 수 있고 tagger 신원이 달라진다. 제거를 권하기에는 prepare가 아직 없고, dispatch는 로컬 clone 없이 유일하게 릴리즈할 수 있는 수동 복구 경로다. 따라서 prepare 정착까지는 dispatch를 유지하고, 그동안 workflow의 공유 verify 단계를 약화하지 않기로 권고한다. 존폐의 최종 결정은 미해결 질문 1으로 남긴다. ADR 0001은 "이 기존 gg commit/gg tag/gg push 경로가 재현 가능한 사용자 경로다"라고 말하는데 dispatch는 그 경로 밖에서 tag를 만드는 두 번째 경로이므로, 이 충돌을 숨기지 않고 여기에 밝힌다.
7. **dry-run**: `--explain` 정적 예고만 두기로 했다. 실제 검사만 수행하는 별도 모드(`--check`, `--dry-run`)는 기각했다 — 모든 검사가 변경 앞에 있고 하나라도 실패하면 아무 것도 변경되지 않으므로 실패한 prepare 실행 자체가 검사 전용 실행이며, explain이 실행 예고를 담당한다. 표면적을 늘릴 사용성 근거가 구현 후에 생기면 그때 별도로 검토한다.

## 기존 ADR과의 관계

prepare는 ADR 0001의 계약을 자동화할 뿐 대체하지 않는다. ADR 0001의 불변성 계약(같은 version 재작성 금지, tag 이동 금지, force push 금지, atomic push 필수)은 그대로이며 prepare의 검사는 그 집행 방법을 하나 늘린다. 실행 계약과 `--no-gpg-sign`, 전역 flag 거부는 ADR 0004의 Git 전달 명령 계약을 재사용하되, prepare는 Git 전달 명령이 아니므로 `--repo`/`--remote` 거부를 제외한 0004의 flag 계약은 적용되지 않는다. 저장소 문맥 선택(ADR 0003)도 prepare에는 적용되지 않는다 — 명령이 항상 현재 작업 트리와 `origin`을 대상으로 하기 때문이다.

## 미해결 질문

1. **workflow_dispatch 태그 생성 경로의 존폐**: 유지하면 릴리즈 경로가 둘이 되어 계약 구현이 drift할 수 있고 tagger 신원(github-actions[bot])이 달라진다. 제거하면 로컬 clone 없이는 릴리즈·복구할 방법이 사라지고 prepare 실구현 전에는 대체 경로가 없다. 권고는 prepare가 구현되고 실제 릴리즈로 안정이 확인될 때까지 유지한 뒤 제거를 재검토하는 것이다. 최종 결정은 maintainer의 몫으로 남긴다.
2. **tag 접미사 허용 여부**: README는 "선택적 접미사 허용"(`vMAJOR.MINOR.PATCH-rc.1` 등)이라 쓰지만 ADR 0001과 release workflow는 접미사 없는 `^v[0-9]+\.[0-9]+\.[0-9]+$`만 받는다. prepare는 현재 게이트 기준으로 엄격한 형식을 따른다. README의 접미사 문장을 고칠지 workflow 정규식을 접미사 허용으로 넓힐지는 maintainer가 정해야 하고, 넓히기로 하면 prepare의 형식 검사도 함께 넓힌다.
