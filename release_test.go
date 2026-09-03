package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestModulePathInGoMod(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("go.mod 읽기 실패: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		t.Fatalf("go.mod 내용이 비어 있습니다")
	}
	firstLine := strings.TrimSpace(lines[0])
	want := "module github.com/dungsil-ai/gg"
	if firstLine != want {
		t.Errorf("go.mod 첫 줄 = %q; want %q", firstLine, want)
	}
}

func TestReleaseArchiveNaming(t *testing.T) {
	cases := []struct {
		version      string
		goos         string
		goarch       string
		wantArchive  string
		wantChecksum string
	}{
		{"v0.1.0", "windows", "amd64", "gg_0.1.0_windows_amd64.zip", "gg_0.1.0_windows_amd64.zip.sha256"},
		{"v0.1.0", "windows", "arm64", "gg_0.1.0_windows_arm64.zip", "gg_0.1.0_windows_arm64.zip.sha256"},
		{"v0.1.0", "linux", "amd64", "gg_0.1.0_linux_amd64.tar.gz", "gg_0.1.0_linux_amd64.tar.gz.sha256"},
		{"v0.1.0", "linux", "arm64", "gg_0.1.0_linux_arm64.tar.gz", "gg_0.1.0_linux_arm64.tar.gz.sha256"},
		{"v0.1.0", "darwin", "amd64", "gg_0.1.0_darwin_amd64.tar.gz", "gg_0.1.0_darwin_amd64.tar.gz.sha256"},
		{"v0.1.0", "darwin", "arm64", "gg_0.1.0_darwin_arm64.tar.gz", "gg_0.1.0_darwin_arm64.tar.gz.sha256"},
	}

	for _, tc := range cases {
		gotArchive := ReleaseArchiveName(tc.version, tc.goos, tc.goarch)
		if gotArchive != tc.wantArchive {
			t.Errorf("ReleaseArchiveName(%q, %q, %q) = %q; want %q", tc.version, tc.goos, tc.goarch, gotArchive, tc.wantArchive)
		}
		gotChecksum := ReleaseChecksumName(gotArchive)
		if gotChecksum != tc.wantChecksum {
			t.Errorf("ReleaseChecksumName(%q) = %q; want %q", gotArchive, gotChecksum, tc.wantChecksum)
		}
	}
}

func extractBinaryFromArchive(t *testing.T, archiveData []byte, ext, binName string) ([]byte, os.FileMode) {
	t.Helper()
	if ext == "zip" {
		zr, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
		if err != nil {
			t.Fatalf("zip 열기 실패: %v", err)
		}
		for _, f := range zr.File {
			if f.Name == binName {
				rc, err := f.Open()
				if err != nil {
					t.Fatalf("zip 파일 열기 실패 (%s): %v", binName, err)
				}
				defer rc.Close()
				data, err := io.ReadAll(rc)
				if err != nil {
					t.Fatalf("zip 파일 읽기 실패 (%s): %v", binName, err)
				}
				return data, f.Mode()
			}
		}
		t.Fatalf("zip 내부에 %s 파일이 없습니다", binName)
	} else if ext == "tar.gz" {
		gr, err := gzip.NewReader(bytes.NewReader(archiveData))
		if err != nil {
			t.Fatalf("gzip 열기 실패: %v", err)
		}
		defer gr.Close()
		tr := tar.NewReader(gr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("tar 읽기 실패: %v", err)
			}
			if hdr.Name == binName {
				data, err := io.ReadAll(tr)
				if err != nil {
					t.Fatalf("tar 파일 읽기 실패 (%s): %v", binName, err)
				}
				return data, os.FileMode(hdr.Mode)
			}
		}
		t.Fatalf("tar.gz 내부에 %s 파일이 없습니다", binName)
	}
	t.Fatalf("지원하지 않는 확장자: %s", ext)
	return nil, 0
}

func TestBuildAndPackageRelease(t *testing.T) {
	outDir := os.Getenv("GG_RELEASE_OUT_DIR")
	if outDir == "" {
		outDir = t.TempDir()
	}
	version := os.Getenv("GG_RELEASE_VERSION")
	if version == "" {
		version = "v0.1.0"
	}

	err := BuildAndPackageRelease(".", outDir, version)
	if err != nil {
		t.Fatalf("BuildAndPackageRelease 실패: %v", err)
	}

	// 1. 공통 checksums.txt가 없는지 확인
	commonChecksumPath := filepath.Join(outDir, "checksums.txt")
	if _, err := os.Stat(commonChecksumPath); !os.IsNotExist(err) {
		t.Errorf("공통 checksums.txt가 생성되었습니다; 생성하지 않아야 합니다")
	}

	// 2. 6개 대상 archive 및 checksum 파일 확인
	targets := []struct {
		goos   string
		goarch string
		ext    string
		bin    string
	}{
		{"windows", "amd64", "zip", "gg.exe"},
		{"windows", "arm64", "zip", "gg.exe"},
		{"linux", "amd64", "tar.gz", "gg"},
		{"linux", "arm64", "tar.gz", "gg"},
		{"darwin", "amd64", "tar.gz", "gg"},
		{"darwin", "arm64", "tar.gz", "gg"},
	}

	if len(targets) != 6 {
		t.Fatalf("target 개수 = %d; want 6", len(targets))
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("outDir 읽기 실패: %v", err)
	}
	// 6 archives + 6 checksum files = 12 files
	if len(entries) != 12 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("생성된 파일 수 = %d (%v); want 12 (6 archives + 6 checksums)", len(entries), names)
	}

	for _, tg := range targets {
		archiveName := ReleaseArchiveName(version, tg.goos, tg.goarch)
		archivePath := filepath.Join(outDir, archiveName)
		checksumName := ReleaseChecksumName(archiveName)
		checksumPath := filepath.Join(outDir, checksumName)

		archiveData, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("archive 파일 읽기 실패: %s: %v", archiveName, err)
		}

		// archive 내용 및 파일 모드 검증
		binData, mode := extractBinaryFromArchive(t, archiveData, tg.ext, tg.bin)
		if len(binData) == 0 {
			t.Errorf("%s 내부 %s 크기가 0입니다", archiveName, tg.bin)
		}
		if tg.ext == "tar.gz" && mode&0o111 == 0 {
			t.Errorf("%s 내부 %s 에 실행 권한이 없습니다 (mode: %o)", archiveName, tg.bin, mode)
		}

		// checksum 검증
		checksumData, err := os.ReadFile(checksumPath)
		if err != nil {
			t.Fatalf("checksum 파일 읽기 실패: %s: %v", checksumName, err)
		}

		h := sha256.Sum256(archiveData)
		wantHex := hex.EncodeToString(h[:])
		wantChecksumContent := fmt.Sprintf("%s  %s\n", wantHex, archiveName)

		if string(checksumData) != wantChecksumContent {
			t.Errorf("%s 내용 = %q; want %q", checksumName, string(checksumData), wantChecksumContent)
		}

		// 현재 OS/Arch인 경우 바이너리 실행하여 gg v0.1.0 출력 확인
		if tg.goos == runtime.GOOS && tg.goarch == runtime.GOARCH {
			binPath := filepath.Join(t.TempDir(), tg.bin)
			if err := os.WriteFile(binPath, binData, 0o755); err != nil {
				t.Fatalf("바이너리 쓰기 실패: %v", err)
			}

			wantVersionOutput := fmt.Sprintf("gg %s\n", version)
			for _, args := range [][]string{{"version"}, {"--version"}} {
				stdout, stderr, code := runGGStreams(t, binPath, t.TempDir(), args...)
				if code != 0 || stdout != wantVersionOutput || stderr != "" {
					t.Errorf("release binary %v = stdout %q, stderr %q, exit %d; want stdout %q, exit 0", args, stdout, stderr, code, wantVersionOutput)
				}
			}
		}
	}
}

func TestBuildAndPackageReleaseRefusesExistingAssets(t *testing.T) {
	tag := "v0.1.0"
	target := ReleaseTargets[0]
	archiveName := ReleaseArchiveName(tag, target.OS, target.Arch)

	for _, assetName := range []string{archiveName, ReleaseChecksumName(archiveName)} {
		t.Run(assetName, func(t *testing.T) {
			outDir := t.TempDir()
			assetPath := filepath.Join(outDir, assetName)
			original := []byte("existing release asset")
			if err := os.WriteFile(assetPath, original, 0o644); err != nil {
				t.Fatalf("기존 asset 쓰기 실패: %v", err)
			}

			err := BuildAndPackageRelease(".", outDir, tag)
			if err == nil {
				t.Fatal("기존 release asset이 있는데 빌드가 성공했습니다")
			}
			if !strings.Contains(err.Error(), assetName) {
				t.Errorf("오류 %q에 기존 asset 이름 %q이 없습니다", err, assetName)
			}

			got, err := os.ReadFile(assetPath)
			if err != nil {
				t.Fatalf("기존 asset 읽기 실패: %v", err)
			}
			if !bytes.Equal(got, original) {
				t.Errorf("기존 asset이 변경되었습니다: got %q, want %q", got, original)
			}
			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("outDir 읽기 실패: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != assetName {
				t.Errorf("기존 asset 외 파일이 생성되었습니다: %v", entries)
			}
		})
	}
}

func TestREADMEContent(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md 읽기 실패: %v", err)
	}
	content := string(data)

	requiredSnippets := []string{
		"# TODO",
		"# TOBE",
		"git 기능 목록",
		"gh (GitHub CLI) 기능 목록",
		"glab (GitLab CLI) 기능 목록",
		"tea (Gitea CLI) 기능 목록",
		"gg 고유 설정 기능 목록",
		"- [x] `git clone` (대응: `gg repo clone`, `gg clone`)",
		"- [x] `git commit` (대응: `gg repo commit`, `gg commit`; 커밋 서명 비활성화)",
		"- [x] `git pull` (대응: `gg repo pull`, `gg pull`)",
		"- [x] `git push` (대응: `gg repo push`, `gg push`)",
		"- [x] `gh issue create`",
		"- [x] `gh pr create`",
		"- [x] `glab mr create`",
		"- [x] `tea pulls create`",
		"- [x] `gg config list`",
	}
	for _, action := range gitPassthroughActionNames {
		requiredSnippets = append(requiredSnippets,
			"- [x] `git "+action+"` (대응: `gg repo "+action+"`, `gg "+action+"`)",
		)
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("README.md에 필수 문구 %q가 누락되었습니다", snippet)
		}
	}

	forbiddenSnippets := []string{
		"대상 저장소",
		"Repo Context",
		"Provider Profile",
		"Login 설정",
		"기본 host",
		"Release Binary",
		"Build File",
	}

	for _, snippet := range forbiddenSnippets {
		if strings.Contains(content, snippet) {
			t.Errorf("README.md에 금지된 용어 %q가 포함되었습니다", snippet)
		}
	}
}

func TestReleaseWorkflowGate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("ci.yml 읽기 실패: %v", err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	// 1. jobs 섹션 및 잡별 블록 추출
	jobsIdx := strings.Index(content, "\njobs:\n")
	if jobsIdx == -1 {
		t.Fatalf("ci.yml에 jobs 섹션이 없습니다")
	}
	headerBlock := content[:jobsIdx]
	jobsContent := content[jobsIdx+len("\njobs:\n"):]

	extractJobBlock := func(jobName string) (string, error) {
		lines := strings.Split(jobsContent, "\n")
		var jobLines []string
		inJob := false
		for _, line := range lines {
			if strings.HasPrefix(line, "  "+jobName+":") {
				inJob = true
				jobLines = append(jobLines, line)
				continue
			}
			if inJob {
				if len(line) > 0 && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
					break
				}
				if strings.HasPrefix(line, "  ") && len(line) > 2 && line[2] != ' ' && line[2] != '\t' {
					// 다른 top-level 잡 시작
					break
				}
				jobLines = append(jobLines, line)
			}
		}
		if !inJob {
			return "", fmt.Errorf("잡 %q을 찾을 수 없습니다", jobName)
		}
		return strings.Join(jobLines, "\n"), nil
	}

	verifyBlock, err := extractJobBlock("verify")
	if err != nil {
		t.Fatalf("verify 잡 추출 실패: %v", err)
	}
	releaseBlock, err := extractJobBlock("release")
	if err != nil {
		t.Fatalf("release 잡 추출 실패: %v", err)
	}
	// 2. tag push는 전달 신호이며, release 잡이 전용 release 커밋과 tag를 검증한다.
	if !strings.Contains(headerBlock, `tags:`) || !strings.Contains(headerBlock, `"v*"`) {
		t.Errorf("ci.yml push 트리거에 tags v* 항목이 없습니다")
	}

	// 3. verify 잡 계약: OS 매트릭스 및 테스트/vet 검증
	for _, expected := range []string{"ubuntu-latest", "windows-latest", "go vet ./...", "go test ./..."} {
		if !strings.Contains(verifyBlock, expected) {
			t.Errorf("verify 잡 블록에 필수 항목 %q가 없습니다", expected)
		}
	}

	// 4. release 잡은 필요한 실행 환경과 verify 의존성을 가져야 한다.
	for _, expected := range []string{
		"needs: verify",
		"if: startsWith(github.ref, 'refs/tags/v')",
		"contents: write",
		"fetch-depth: 0",
	} {
		if !strings.Contains(releaseBlock, expected) {
			t.Errorf("release 잡 블록에 필수 설정 %q가 없습니다", expected)
		}
	}

	// 5. 게시 전에 지켜야 할 release provenance 및 불변성 gate를 고정한다.
	gates := []struct {
		fragment string
		name     string
	}{
		{"does not match semantic versioning format", "Semantic Versioning tag 검증"},
		{"does not point to current HEAD commit", "tag 대상 commit 검증"},
		{"github.event.repository.default_branch", "default branch 이름 조회"},
		{"git ls-remote --exit-code --refs origin", "원격 default branch tip 조회"},
		{"is not the current tip of default branch", "default branch 현재 tip 검증"},
		{"does not match expected release commit format", "release commit 제목 검증"},
		{"is not an empty commit", "빈 release commit 검증"},
		{"must be an annotated tag", "annotated tag 검증"},
		{"releases/tags/", "기존 Release 조회"},
		{"already exists", "기존 Release 차단"},
		{"HTTP 404", "Release 부재의 404 확인"},
		{"Failed to verify release absence due to API error", "Release 조회 오류 차단"},
		{"Verify Immutable Releases setting", "Immutable Releases 설정 검증 스텝"},
		{"RELEASE_ADMIN_TOKEN: ${{ secrets.RELEASE_ADMIN_TOKEN }}", "Immutable Releases 관리자 secret"},
		{"GH_TOKEN=\"$RELEASE_ADMIN_TOKEN\" gh api --method GET", "Immutable Releases GET 요청의 관리자 token"},
		{"repos/${{ github.repository }}/immutable-releases", "Immutable Releases 설정 endpoint"},
		{"RELEASE_ADMIN_TOKEN is required", "관리자 secret 누락 차단"},
		{"Failed to verify GitHub Immutable Releases setting", "Immutable Releases API 오류 차단"},
		{`"$IMMUTABLE_RELEASES_ENABLED" != "true"`, "Immutable Releases 비활성 상태 차단"},
		{"must be enabled before publishing a release", "Immutable Releases 활성화 요구"},
	}
	publishStepIdx := strings.Index(releaseBlock, "gh release create")
	if publishStepIdx == -1 {
		t.Error("release 잡에 GitHub Release 발행 스텝(gh release create)이 없습니다")
	}
	for _, gate := range gates {
		idx := strings.Index(releaseBlock, gate.fragment)
		if idx == -1 {
			t.Errorf("release 잡에 %s gate가 없습니다", gate.name)
			continue
		}
		if publishStepIdx != -1 && idx >= publishStepIdx {
			t.Errorf("%s gate가 GitHub Release 발행 뒤에 있습니다", gate.name)
		}
	}

	// Immutable Releases는 GITHUB_TOKEN이 아닌 최소 Administration (read) 권한의
	// 전용 secret으로 조회하고, 누락·오류·비활성 상태를 모두 publish 전에 실패시켜야 한다.
	immutableGateStart := strings.Index(releaseBlock, "- name: Verify Immutable Releases setting")
	if immutableGateStart == -1 {
		t.Error("release 잡에 Immutable Releases 설정 검증 스텝이 없습니다")
	} else {
		immutableGate := releaseBlock[immutableGateStart:]
		if nextStepIdx := strings.Index(immutableGate, "\n      - name:"); nextStepIdx != -1 {
			immutableGate = immutableGate[:nextStepIdx]
		}
		for _, requirement := range []struct {
			fragment string
			name     string
		}{
			{"RELEASE_ADMIN_TOKEN: ${{ secrets.RELEASE_ADMIN_TOKEN }}", "전용 관리자 secret"},
			{"GH_TOKEN=\"$RELEASE_ADMIN_TOKEN\"", "관리자 token으로 API 호출"},
			{"--method GET", "GET method"},
			{"repos/${{ github.repository }}/immutable-releases", "Immutable Releases endpoint"},
			{"RELEASE_ADMIN_TOKEN is required", "secret 누락 실패"},
			{"Failed to verify GitHub Immutable Releases setting", "API 오류 실패"},
			{`"$IMMUTABLE_RELEASES_ENABLED" != "true"`, "enabled=true 외 상태 실패"},
			{"must be enabled before publishing a release", "비활성 설정 실패"},
		} {
			if !strings.Contains(immutableGate, requirement.fragment) {
				t.Errorf("Immutable Releases gate에 %s(%q)가 없습니다", requirement.name, requirement.fragment)
			}
		}
		if strings.Contains(immutableGate, "github.token") {
			t.Error("Immutable Releases gate가 관리자 secret 대신 github.token을 사용합니다")
		}
	}

	if publishStepIdx != -1 {
		publishStepStart := strings.LastIndex(releaseBlock[:publishStepIdx], "- name: Publish GitHub Release")
		if publishStepStart == -1 {
			t.Error("GitHub Release 발행 스텝의 시작을 찾을 수 없습니다")
		} else {
			publishBlock := releaseBlock[publishStepStart:]
			if !strings.Contains(publishBlock, "GH_TOKEN: ${{ github.token }}") {
				t.Error("GitHub Release 발행은 github.token을 사용해야 합니다")
			}
			if strings.Contains(publishBlock, "RELEASE_ADMIN_TOKEN") {
				t.Error("GitHub Release 발행에 관리자 token을 사용하면 안 됩니다")
			}
		}
	}

	// 6. Release 파일은 검증 뒤 빌드하고, 검증된 기존 tag로만 게시한다.
	buildStepIdx := strings.Index(releaseBlock, "TestBuildAndPackageRelease")
	if buildStepIdx == -1 {
		t.Error("release 잡에 6개 타겟 빌드/패키징 스텝(TestBuildAndPackageRelease)이 없습니다")
	} else if publishStepIdx != -1 && buildStepIdx >= publishStepIdx {
		t.Error("release 잡에서 빌드/패키징 스텝이 배포 스텝보다 먼저 실행되어야 합니다")
	}
	if !strings.Contains(releaseBlock, "GG_RELEASE_OUT_DIR: dist") || !strings.Contains(releaseBlock, "GG_RELEASE_VERSION: ${{ github.ref_name }}") {
		t.Errorf("release 잡 빌드 스텝에 환경 변수 설정(GG_RELEASE_OUT_DIR, GG_RELEASE_VERSION)이 누락되었습니다")
	}
	if !strings.Contains(releaseBlock, "--verify-tag") {
		t.Error("release 잡 배포 명령이 기존 tag 검증(--verify-tag)을 하지 않습니다")
	}

	// 7. 범위 외 요소 및 기존 release/asset을 바꾸는 옵션은 금지한다.

	forbiddenItems := []struct {
		pattern string
		reason  string
	}{
		{"--clobber", "기존 Release asset 덮어쓰기 플래그"},
		{"--draft", "Draft release 생성 플래그"},
		{"signtool", "Windows code signing 도구"},
		{"codesign", "macOS signing 도구"},
		{"notarize", "macOS notarization"},
		{"gon ", "macOS notarization 도구"},
		{"gpg", "GPG 서명"},
		{"cosign", "Sigstore cosign 서명"},
		{"sigstore", "Sigstore"},
	}

	for _, item := range forbiddenItems {
		if strings.Contains(strings.ToLower(content), strings.ToLower(item.pattern)) {
			t.Errorf("ci.yml에 허용되지 않는 항목(%s: %q)이 포함되어 있습니다", item.reason, item.pattern)
		}
	}
}
