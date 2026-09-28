# Git 전달 명령 계약

`gg`의 Git 전달 명령은 `gg <action>`과 `gg repo <action>`을 `git <action> [args...]`로 실행한다. registry에 등록한 Main Porcelain 37개와 ancillary 14개는 `--help`를 포함한 action 뒤 인자를 그대로 Git에 전달한다. Git을 실행하는 경우 저장소 문맥이나 Provider 설정을 조회하지 않으며, Git의 stdin, stdout, stderr, 종료 코드를 보존한다. `gg` 부모는 SIGINT를 소비하며 자식을 기다리고, fork/exec 자식은 default SIGINT disposition으로 터미널 Ctrl+C를 직접 받으며, Unix 신호 종료는 shell 관례 `128+signal`으로 반환한다.

`commit`만 기존의 non-signing 정책을 유지하기 위해 Git 인자 앞에 `--no-gpg-sign`을 넣어 `git commit --no-gpg-sign [args...]`로 실행한다. 다른 Git 전달 명령은 action과 action 뒤 Git 인자를 바꾸지 않는다.

명령 앞의 `--repo`, `--remote`, `--explain`은 Git 전달 명령에서 지원하지 않으며 UsageError와 exit code `2`로 거절한다. action 뒤의 같은 토큰은 `gg` flag로 파싱하지 않고 Git 인자(`GitArgs`)로 보존해 Git에 전달한다.

(2026-09-28 갱신) `--help`는 Git에 전달하지 않고 `gg` action help를 출력한다. `gg status --help`, `gg repo status --help`, `gg commit --help`가 모두 같은 규칙을 따르므로 이전의 help alias 구분(`pull`, `push`는 `gg` help, `gg commit --help`는 Git 전달)은 없어졌다. `--` 앞의 `--help`만 이 규칙을 따르고, `--` 뒤의 `--help`는 경로명 같은 위치 인자로 보아 그대로 전달한다. `auth` 릴레이와 `gg api`는 자식 CLI로 `--help`를 전달하므로(ADR 0007) 이 규칙은 Git 전달 명령에만 적용된다.

이유는 Git for Windows가 `help.format=html`을 쓰기 때문이다. `git status --help`는 Git 문서 HTML을 기본 브라우저로 연다. 자동화된 실행에서는 요청하지 않은 창이 뜨고 사용법도 얻지 못하므로 `--help`를 `gg` help로 흡수한다. Git 옵션의 짧은 사용법은 `gg status -h`로 확인한다. `-h`는 Git flag이므로 그대로 전달하며 Git의 usage 관례대로 exit code `129`를 보존한다. 다른 후보 두 가지는 기각했다. (1) `--help`를 `-h`로 바꿔 전달하는 방식은 `git grep -h`처럼 `-h`가 다른 뜻인 명령에서 동작을 바꾼다. (2) `help.format`을 `man`으로 바꾸는 방식은 사용자 Git 설정을 요구하고 man viewer가 없는 Windows에서 실패한다.

이 결정은 ADR 0003의 저장소 문맥 선택 범위를 forge 명령으로 한정한다. 0003의 pull/push가 남은 인자를 Git에 그대로 전달한다는 설명은 action 뒤 인자에는 계속 적용되지만, 명령 앞 `--remote`에는 적용되지 않는다.

모든 repo action에 저장소 문맥 및 Provider를 해석한 뒤 Git을 실행하는 방안과 앞의 `gg` context flag를 Git에 넘기는 방안을 검토했다. 전자는 직접 Git 실행의 I/O와 종료 코드 계약에 provider 조회를 끼워 넣고, 후자는 action 앞뒤의 flag 경계를 숨긴다. Git 전달 명령을 Forge 대상 선택과 독립시키고 앞의 context flag를 명시적으로 거절한다.
