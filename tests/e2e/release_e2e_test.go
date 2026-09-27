package e2e

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
	"sync"
	"testing"
)

var sharedReleasePackagerPath string

var buildSharedReleasePackager = sync.OnceFunc(func() {
	dir, err := os.MkdirTemp("", "gg-release-packager")
	if err != nil {
		panic(err)
	}
	bin := filepath.Join(dir, "package-release")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, "../../internal/cmd/package-release").CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic(fmt.Sprintf("package-release build: %v\n%s", err, out))
	}
	sharedReleasePackagerPath = bin
})

func runReleasePackager(t *testing.T, outDir, tag string) (string, int) {
	t.Helper()
	buildSharedReleasePackager()
	cmd := exec.Command(sharedReleasePackagerPath)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "GG_RELEASE_OUT_DIR="+outDir, "GG_RELEASE_VERSION="+tag)
	out, err := cmd.CombinedOutput()
	return string(out), processExitCode(t, err, string(out))
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

func TestE2EBuildAndPackageRelease(t *testing.T) {
	outDir := t.TempDir()
	const version = "v0.1.0"
	if out, code := runReleasePackager(t, outDir, version); code != 0 {
		t.Fatalf("package-release exit %d: %s", code, out)
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

func TestE2EReleasePackagingRefusesExistingAssets(t *testing.T) {
	tag := "v0.1.0"
	archiveName := "gg_0.1.0_windows_amd64.zip"

	for _, assetName := range []string{archiveName, archiveName + ".sha256"} {
		t.Run(assetName, func(t *testing.T) {
			outDir := t.TempDir()
			assetPath := filepath.Join(outDir, assetName)
			original := []byte("existing release asset")
			if err := os.WriteFile(assetPath, original, 0o644); err != nil {
				t.Fatalf("기존 asset 쓰기 실패: %v", err)
			}

			out, code := runReleasePackager(t, outDir, tag)
			if code != 1 {
				t.Fatal("기존 release asset이 있는데 빌드가 성공했습니다")
			}
			if !strings.Contains(out, assetName) {
				t.Errorf("오류 %q에 기존 asset 이름 %q이 없습니다", out, assetName)
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

func TestE2EReleasePackagingRejectsInvalidVersion(t *testing.T) {
	for _, tag := range []string{"vbroken", "v1.0.0 -X main.version=bad", "../v1.0.0"} {
		t.Run(tag, func(t *testing.T) {
			outDir := filepath.Join(t.TempDir(), "dist")
			if out, code := runReleasePackager(t, outDir, tag); code != 1 || !strings.Contains(out, "유효하지 않은 release tag") {
				t.Fatalf("exit %d: %s", code, out)
			}
			if _, err := os.Stat(outDir); !os.IsNotExist(err) {
				t.Fatalf("invalid version created output: %v", err)
			}
		})
	}
	for _, tc := range []struct{ out, tag string }{{"", "v1.0.0"}, {t.TempDir(), ""}} {
		if out, code := runReleasePackager(t, tc.out, tc.tag); code != 2 || !strings.Contains(out, "are required") {
			t.Fatalf("missing environment: exit %d: %s", code, out)
		}
	}
}
