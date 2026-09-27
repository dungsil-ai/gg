//go:build unix

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestE2EChildSignalExitCodes(t *testing.T) {
	bin := buildGG(t)
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "git"), []byte(fmt.Sprintf("#!/bin/sh\nkill -%d $$\n", sig)), 0o755); err != nil {
				t.Fatal(err)
			}
			if out, code := runGG(t, bin, dir, t.TempDir(), "status"); code != 128+int(sig) {
				t.Fatalf("signal %v: exit %d: %s", sig, code, out)
			}
		})
	}
}
