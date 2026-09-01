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
gg pr create --help
```

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
