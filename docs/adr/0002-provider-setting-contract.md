# Provider 설정 계약

`gg`는 self-hosted host의 Provider 설정을 `gg config list`, `gg config set <host> <provider>`, `gg config unset <host>`로 관리한다. 기본 domain인 `github.com`, `gitlab.com`, `gitea.com`은 고정 mapping을 쓰며 `config set`으로 바꾸지 않는다.

Host 입력은 scheme과 path가 없는 hostname 또는 hostname과 port다. Host는 lowercase로 바꾸고 port는 빼서 저장하므로 같은 host의 여러 port는 하나의 Provider 설정을 함께 쓴다. Provider는 `gh`, `glab`, `tea`만 허용한다.

`config set`은 provider CLI 설치와 login이 없어도 값을 저장한다. 실제 forge 명령을 실행할 때 binary와 login 상태를 다시 확인한다. 같은 값이나 다른 값으로 다시 설정해도 확인 질문 없이 원자적으로 저장한다.

`config unset`은 설정이 없어도 성공한다. `config list`는 host 기준으로 정렬한 `HOST`와 `PROVIDER` 표를 보여주며 설정이 없으면 `No provider settings.`를 stdout에 쓴다. 공통 JSON 출력은 현재 범위 밖이다.

설정에는 token, login 정보, 저장소 URL을 넣지 않는다. 손상된 `config.json`은 모든 변경 명령이 덮어쓰지 않는다. 저장 전에 실제 login을 요구하는 방식과 기본 domain override도 검토했지만, 설치 전 자동 설정과 안전한 고정 domain 동작을 위해 선택하지 않았다.
