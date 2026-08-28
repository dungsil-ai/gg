# gg

`gg`는 Git 저장소 host를 보고 같은 명령을 provider CLI로 보내는 도구다. 이 문서는 `gg`가 쓰는 고유 용어를 정한다.

## Terms

**저장소 문맥 (Repository Context)**:
Forge 명령이 대상으로 삼는 하나의 저장소다. 명시한 URL, remote 이름, 또는 자동으로 고른 Git remote에서 얻는다.
_Avoid_: 대상 저장소, Repo Context

**Provider 설정 (Provider Setting)**:
Self-hosted host와 이를 처리할 `gh`, `glab`, `tea` 중 하나의 연결이다. Host는 port 없는 lowercase 값이며 token, login 정보, 저장소 URL은 포함하지 않는다.
_Avoid_: Provider Profile, Login 설정

**기본 domain (Default Domain)**:
`github.com`, `gitlab.com`, `gitea.com`처럼 provider가 `gg`에 고정된 host다. Provider 설정으로 바꾸지 않는다.
_Avoid_: 기본 host

**Release 파일 (Release File)**:
Tag version과 하나의 OS, CPU 종류에 맞춰 공개한 archive와 그 archive의 SHA-256 파일이다.
_Avoid_: Release Binary, Build File
