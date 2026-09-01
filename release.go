package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func BuildAndPackageRelease(srcDir, outDir, tag string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("outDir 생성 실패: %w", err)
	}

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

		archiveData, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("archive 읽기 실패: %w", err)
		}

		h := sha256.Sum256(archiveData)
		hexHash := hex.EncodeToString(h[:])
		checksumContent := fmt.Sprintf("%s  %s\n", hexHash, archiveName)

		checksumName := ReleaseChecksumName(archiveName)
		checksumPath := filepath.Join(outDir, checksumName)
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			return fmt.Errorf("checksum 파일 쓰기 실패 (%s): %w", checksumName, err)
		}
	}

	return nil
}

func writeZip(zipPath, filename string, data []byte) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(filename)
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := w.Write(data); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func writeTarGz(tarGzPath, filename string, data []byte) error {
	f, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0o755,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		_ = tw.Close()
		_ = gw.Close()
		return err
	}
	if _, err := tw.Write(data); err != nil {
		_ = tw.Close()
		_ = gw.Close()
		return err
	}
	if err := tw.Close(); err != nil {
		_ = gw.Close()
		return err
	}
	return gw.Close()
}
