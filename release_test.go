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
		archiveName := fmt.Sprintf("gg_0.1.0_%s_%s.%s", tg.goos, tg.goarch, tg.ext)
		archivePath := filepath.Join(outDir, archiveName)
		checksumName := archiveName + ".sha256"
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

			for _, args := range [][]string{{"version"}, {"--version"}} {
				stdout, stderr, code := runGGStreams(t, binPath, t.TempDir(), args...)
				if code != 0 || stdout != "gg v0.1.0\n" || stderr != "" {
					t.Errorf("release binary %v = stdout %q, stderr %q, exit %d; want stdout %q, exit 0", args, stdout, stderr, code, "gg v0.1.0\n")
				}
			}
		}
	}
}

func TestREADMEContent(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md 읽기 실패: %v", err)
	}
	content := string(data)

	requiredSnippets := []string{
		"go install github.com/dungsil-ai/gg@latest",
		"gg_0.1.0_windows_amd64.zip",
		"gg_0.1.0_linux_amd64.tar.gz",
		"gg_0.1.0_darwin_amd64.tar.gz",
		".sha256",
		"gg help",
		"gg --help",
		"gg version",
		"gg --version",
		"gg config --help",
		"gg issue --help",
		"gg pr create --help",
		"--remote",
		"gg config list",
		"gg config set",
		"gg config unset",
		"저장소 문맥",
		"Provider 설정",
		"기본 domain",
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
