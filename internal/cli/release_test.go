package cli

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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestModulePathInGoMod(t *testing.T) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = "../.."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("module 조회 실패: %v", err)
	}
	if got, want := strings.TrimSpace(string(output)), "github.com/dungsil-ai/gg"; got != want {
		t.Errorf("module = %q; want %q", got, want)
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

	err := BuildAndPackageRelease("../..", outDir, version)
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

			err := BuildAndPackageRelease("../..", outDir, tag)
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
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("README.md 읽기 실패: %v", err)
	}
	content := string(data)

	requiredSnippets := []string{
		"https://github.com/dungsil-ai/gg/releases",
		"go install github.com/dungsil-ai/gg@latest",
		".sha256",
		"gg repo clone", "gg clone",
		"gg repo commit", "gg commit",
		"gg repo pull", "gg pull",
		"gg repo push", "gg push",
		"gh issue create", "gh pr create",
		"glab mr create", "tea pulls create",
		"gg config list",
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
