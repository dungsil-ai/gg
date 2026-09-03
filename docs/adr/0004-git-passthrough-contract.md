# Git 전달 명령 계약

`gg`의 Git 전달 명령은 `gg <action>`과 `gg repo <action>`을 `git <action> [args...]`로 실행한다. registry에 등록한 Main Porcelain 37개와 ancillary 14개는 `--help`를 포함한 action 뒤 인자를 그대로 Git에 전달한다. Git을 실행하는 경우 저장소 문맥이나 Provider 설정을 조회하지 않으며, Git의 stdin, stdout, stderr, 종료 코드를 보존한다. `gg` 부모는 SIGINT를 소비하며 자식을 기다리고, fork/exec 자식은 default SIGINT disposition으로 터미널 Ctrl+C를 직접 받으며, Unix 신호 종료는 shell 관례 `128+signal`으로 반환한다.

`commit`만 기존의 non-signing 정책을 유지하기 위해 Git 인자 앞에 `--no-gpg-sign`을 넣어 `git commit --no-gpg-sign [args...]`로 실행한다. 다른 Git 전달 명령은 action과 action 뒤 Git 인자를 바꾸지 않는다.

명령 앞의 `--repo`, `--remote`, `--explain`은 Git 전달 명령에서 지원하지 않으며 UsageError와 exit code `2`로 거절한다. action 뒤의 같은 토큰은 `gg` flag로 파싱하지 않고 Git 인자(`GitArgs`)로 보존해 Git에 전달한다. `pull`과 `push`는 기존 help alias이므로 `gg pull --help`, `gg push --help`, `gg repo pull --help`, `gg repo push --help`에서 `gg` action help를 출력한다. `gg repo commit --help`도 `gg` action help를 출력하지만, `gg commit --help`는 `--no-gpg-sign`을 넣어 Git에 전달한다.

이 결정은 ADR 0003의 저장소 문맥 선택 범위를 forge 명령으로 한정한다. 0003의 pull/push가 남은 인자를 Git에 그대로 전달한다는 설명은 action 뒤 인자에는 계속 적용되지만, 명령 앞 `--remote`에는 적용되지 않는다.

모든 repo action에 저장소 문맥 및 Provider를 해석한 뒤 Git을 실행하는 방안과 앞의 `gg` context flag를 Git에 넘기는 방안을 검토했다. 전자는 직접 Git 실행의 I/O와 종료 코드 계약에 provider 조회를 끼워 넣고, 후자는 action 앞뒤의 flag 경계를 숨긴다. Git 전달 명령을 Forge 대상 선택과 독립시키고 앞의 context flag를 명시적으로 거절한다.
