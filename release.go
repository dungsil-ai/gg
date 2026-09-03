package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type ReleaseTarget struct {
	OS   string
	Arch string
}

var ReleaseTargets = []ReleaseTarget{
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
}

func ReleaseArchiveName(version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	cleanVersion := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("gg_%s_%s_%s.%s", cleanVersion, goos, goarch, ext)
}

func ReleaseChecksumName(archiveName string) string {
	return archiveName + ".sha256"
}

// releaseTagPattern은 ldflags에 주입하는 tag 형식을 제한해, go build -ldflags의
// 공백 기준 인자 분리를 통한 링커 인자 주입을 막는다.
var releaseTagPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)

func validReleaseTag(tag string) bool {
	return releaseTagPattern.MatchString(tag)
}

func BuildAndPackageRelease(srcDir, outDir, tag string) (err error) {
	if !validReleaseTag(tag) {
		return fmt.Errorf("유효하지 않은 release tag: %q", tag)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("outDir 생성 실패: %w", err)
	}
	if err := ensureReleaseAssetsAbsent(outDir, tag); err != nil {
		return fmt.Errorf("기존 release asset을 덮어쓸 수 없습니다: %w", err)
	}

	created := make([]string, 0, len(ReleaseTargets)*2)
	defer func() {
		if err == nil {
			return
		}
		for _, path := range created {
			_ = os.Remove(path)
		}
	}()

	for _, target := range ReleaseTargets {
		binName := "gg"
		if target.OS == "windows" {
			binName = "gg.exe"
		}

		tmpDir, err := os.MkdirTemp("", "gg-build-*")
		if err != nil {
			return fmt.Errorf("임시 디렉터리 생성 실패: %w", err)
		}

		binPath := filepath.Join(tmpDir, binName)

		cmd := exec.Command("go", "build", "-ldflags", "-X main.version="+tag, "-o", binPath, srcDir)
		cmd.Env = append(os.Environ(),
			"GOOS="+target.OS,
			"GOARCH="+target.Arch,
			"CGO_ENABLED=0",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmpDir)
			return fmt.Errorf("%s/%s 빌드 실패: %w\n%s", target.OS, target.Arch, err, string(out))
		}

		binData, err := os.ReadFile(binPath)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return fmt.Errorf("바이너리 읽기 실패: %w", err)
		}
		_ = os.RemoveAll(tmpDir)

		archiveName := ReleaseArchiveName(tag, target.OS, target.Arch)
		archivePath := filepath.Join(outDir, archiveName)

		if target.OS == "windows" {
			if err := writeZip(archivePath, binName, binData); err != nil {
				return fmt.Errorf("zip 생성 실패 (%s): %w", archiveName, err)
			}
		} else {
			if err := writeTarGz(archivePath, binName, binData); err != nil {
				return fmt.Errorf("tar.gz 생성 실패 (%s): %w", archiveName, err)
			}
		}
		created = append(created, archivePath)

		archiveData, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("archive 읽기 실패: %w", err)
		}

		h := sha256.Sum256(archiveData)
		hexHash := hex.EncodeToString(h[:])
		checksumContent := fmt.Sprintf("%s  %s\n", hexHash, archiveName)

		checksumName := ReleaseChecksumName(archiveName)
		checksumPath := filepath.Join(outDir, checksumName)
		if err := writeNewFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			return fmt.Errorf("checksum 파일 쓰기 실패 (%s): %w", checksumName, err)
		}
		created = append(created, checksumPath)
	}

	return nil
}

func ensureReleaseAssetsAbsent(outDir, tag string) error {
	for _, target := range ReleaseTargets {
		archiveName := ReleaseArchiveName(tag, target.OS, target.Arch)
		for _, name := range [...]string{archiveName, ReleaseChecksumName(archiveName)} {
			path := filepath.Join(outDir, name)
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("release asset이 이미 있습니다: %s", path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("release asset 확인 실패 (%s): %w", path, err)
			}
		}
	}
	return nil
}

func writeNewFile(path string, data []byte, perm os.FileMode) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	_, err = f.Write(data)
	return err
}

func writeZip(zipPath, filename string, data []byte) (err error) {
	f, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(zipPath)
		}
	}()

	zw := zip.NewWriter(f)
	defer func() {
		if closeErr := zw.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	w, err := zw.Create(filename)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func writeTarGz(tarGzPath, filename string, data []byte) (err error) {
	f, err := os.OpenFile(tarGzPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(tarGzPath)
		}
	}()

	gw := gzip.NewWriter(f)
	defer func() {
		if closeErr := gw.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	tw := tar.NewWriter(gw)
	defer func() {
		if closeErr := tw.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	hdr := &tar.Header{
		Name: filename,
		Mode: 0o755,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tw.Write(data)
	return err
}
