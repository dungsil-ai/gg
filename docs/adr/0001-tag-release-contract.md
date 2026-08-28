# Tag release 계약

`gg`는 Semantic Versioning tag `vMAJOR.MINOR.PATCH`를 쓰고 첫 release를 `v0.1.0`으로 시작한다. `v*` tag push를 공개 승인으로 보며, test와 Windows, Linux, macOS의 `amd64`/`arm64` build가 모두 성공하면 GitHub Release를 바로 공개하고 하나라도 실패하면 공개하지 않는다.

Go module path는 첫 release 전에 `github.com/dungsil-ai/gg`로 바꾼다. Release 파일과 함께 `go install github.com/dungsil-ai/gg@latest`를 지원하고 README에 두 설치 방법을 모두 적는다.

Release archive 이름은 `gg_<version>_<os>_<arch>.<ext>`로 한다. Windows는 `.zip`, Linux와 macOS는 `.tar.gz`를 쓴다. 각 archive는 같은 이름 뒤에 `.sha256`을 붙인 checksum 파일을 따로 가진다. 공통 `checksums.txt`는 만들지 않는다.

Release binary는 `gg version`과 `gg --version`에서 `gg v0.1.0`처럼 tag version을 보여준다. 일반 local build는 `gg dev`를 보여준다. `v0.x` 범위에서는 SHA-256만 제공하고 Windows code signing, macOS notarization, GPG, Sigstore는 인증서와 secret 관리 범위를 늘리므로 제외한다.

Draft release와 공통 checksum 파일도 검토했다. Tag 자체를 공개 승인으로 쓰고 각 파일 옆에서 checksum을 바로 찾을 수 있게 하기 위해 선택하지 않았다.
