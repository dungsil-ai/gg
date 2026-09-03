# gg

`gg`는 Git 저장소 host를 보고 같은 명령을 provider CLI(`gh`, `glab`, `tea`)로 보내는 도구입니다.

## 설치

### 1. Release 파일로 설치

GitHub Releases에서 사용하는 OS와 CPU에 맞는 Release 파일을 내려받습니다.

- **Windows** (zip):
  - `gg_0.1.0_windows_amd64.zip`
  - `gg_0.1.0_windows_arm64.zip`
- **Linux** (tar.gz):
  - `gg_0.1.0_linux_amd64.tar.gz`
  - `gg_0.1.0_linux_arm64.tar.gz`
- **macOS** (tar.gz):
  - `gg_0.1.0_darwin_amd64.tar.gz`
  - `gg_0.1.0_darwin_arm64.tar.gz`

각 archive 옆에 있는 `.sha256` 파일로 SHA-256 checksum을 확인합니다.

```bash
# Linux
sha256sum -c gg_0.1.0_linux_amd64.tar.gz.sha256

# macOS
shasum -a 256 -c gg_0.1.0_darwin_amd64.tar.gz.sha256

# Windows (PowerShell)
(Get-FileHash .\gg_0.1.0_windows_amd64.zip -Algorithm SHA256).Hash -eq ((Get-Content .\gg_0.1.0_windows_amd64.zip.sha256).Split(" ")[0]).ToUpper()
```

압축을 풀고 `gg` (Windows는 `gg.exe`) 실행 파일을 `PATH`에 등록된 경로로 이동합니다.

### 2. go install로 설치

Go 1.22 이상이 설치되어 있다면 다음 명령으로 설치할 수 있습니다.

```bash
go install github.com/dungsil-ai/gg@latest
```

## 첫 실행과 help

도움말을 보려면 다음 명령 중 하나를 실행합니다.

```bash
gg
gg help
gg --help
gg -h
```

버전을 확인하려면 다음 명령을 실행합니다.

```bash
gg version
gg --version
```

각 하위 명령의 도움말을 확인할 수도 있습니다.

```bash
gg config --help
gg issue --help
gg issue list --help
gg issue comment --help
gg issue close --help
gg issue reopen --help
gg pr create --help
gg pr merge --help

## 저장소 문맥 선택 (--remote)

`gg`는 현재 Git 저장소의 remote 정보를 읽어 저장소 문맥을 자동으로 선택합니다.

대상 remote를 직접 지정하려면 `--remote` flag를 사용합니다.

```bash
# upstream remote를 저장소 문맥으로 사용
gg --remote upstream issue list

# 명령 뒤에 --remote를 붙여도 동작합니다
gg pr create --remote origin
```

저장소 URL을 직접 지정하려면 `--repo` flag를 사용합니다.

```bash
gg --repo https://github.com/dungsil-ai/gg issue list
```

### PR 병합 준비 상태

`gg pr status <number>`는 한 PR의 병합 준비 상태를 한 화면에서 보여줍니다. GitHub와 GitLab에서 같은 필드 이름과 값 범위를 사용합니다.

```bash
gg pr status 42
```

```text
Draft: no
Approval: approved
CI: pass
Conflict: no
Mergeable: yes
```

### 출력 값의 뜻

| 필드 | 값 | 뜻 |
| --- | --- | --- |
| `Draft` | `yes`, `no`, `unknown` | Draft PR인지 |
| `Approval` | `approved`, `required`, `changes-requested`, `unknown` | 승인 상태 |
| `CI` | `pass`, `fail`, `pending`, `none`, `unknown` | CI 검사 상태. `none`은 실행된 검사가 없다는 뜻 |
| `Conflict` | `yes`, `no`, `unknown` | base 브랜치와 충돌이 있는지 |
| `Mergeable` | `yes`, `no`, `unknown` | 지금 병합할 수 있는지 |

`unknown`은 provider가 아직 계산 중이거나 값을 주지 않았다는 뜻이며, 임의의 안전한 값(`no`, `pass`)으로 바꿔 판단하지 않습니다.

상태 조회 자체가 성공하면 병합 불가, CI 실패, 승인 대기여도 exit code는 0입니다. 조회가 실패할 때만 0이 아닌 exit code를 냅니다.

`tea`(Gitea)는 이 명령을 지원하지 않습니다.

# Issue 관리

```bash
# Issue 댓글 작성
gg issue comment 18 --body "구현 완료되었습니다."

# Issue 닫기
gg issue close 18

# Issue 다시 열기
gg issue reopen 18
```

## PR 병합

```bash
# PR 병합 (provider 기본 방식)
gg pr merge 42

# 병합 방식 지정: --merge, --squash, --rebase 중 하나
gg pr merge 42 --squash

# 병합 뒤 source branch 삭제
gg pr merge 42 --squash --delete-branch

# 필수 승인과 CI가 끝난 뒤 자동 병합
gg pr merge 42 --auto
```

GitHub에서는 `gh pr merge`, GitLab에서는 `glab mr merge`로 보냅니다. Gitea(`tea`)는 아직 PR 병합을 지원하지 않습니다.

## Provider 설정

Self-hosted host에 사용할 provider(`gh`, `glab`, `tea`)를 지정할 수 있습니다.

기본 domain(`github.com`, `gitlab.com`, `gitea.com`)은 Provider 설정 없이 자동으로 연결됩니다.

### Provider 설정 목록 확인

```bash
gg config list
```

### Provider 설정 추가 및 변경

```bash
gg config set git.example.com glab
gg config set code.internal.net tea
```

### Provider 설정 삭제

```bash
gg config unset git.example.com
```
