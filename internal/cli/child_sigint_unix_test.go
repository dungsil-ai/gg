//go:build linux

package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestExecChildSIGINTDefaultDisposition verifies that execChild does not pass
// SIG_IGN to the git child. A process-group SIGINT must terminate the direct
// git child, while gg waits and exits with the shell-conventional 128+SIGINT.
//
// `gg mergetool` is a passthrough, so this exercises execChild without a
// repository lookup. The fake git records readiness and then execs sleep,
// ensuring SIGINT reaches a process with the default disposition rather than
// a shell wrapper.
func TestExecChildSIGINTDefaultDisposition(t *testing.T) {
	bin := buildGG(t)

	fakeDir := t.TempDir()
	readyPath := filepath.Join(t.TempDir(), "fake-git-ready")
	fakeGit := filepath.Join(fakeDir, "git")
	const fakeGitScript = "#!/bin/sh\n: > \"$GG_TEST_FAKE_GIT_READY\"\nexec sleep 30\n"
	if err := os.WriteFile(fakeGit, []byte(fakeGitScript), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "mergetool")
	cmd.Env = append(os.Environ(),
		"PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GG_HOME="+t.TempDir(),
		"GG_TEST_FAKE_GIT_READY="+readyPath,
	)
	cmd.Dir = t.TempDir()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start gg: %v", err)
	}

	// Setpgid with Pgid zero creates a group led by the child PID.
	pgid := cmd.Process.Pid
	var waitErr error
	waitDone := make(chan struct{})
	go func() {
		waitErr = cmd.Wait()
		close(waitDone)
	}()

	// Always kill the whole group and reap gg. This covers readiness failures,
	// SIGINT timeouts, and unexpected test failures without leaving sleep behind.
	t.Cleanup(func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		select {
		case <-waitDone:
		case <-time.After(3 * time.Second):
			t.Errorf("gg process group %d did not exit after SIGKILL", pgid)
		}
	})

	readyDeadline := time.NewTimer(2 * time.Second)
	defer readyDeadline.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat fake git readiness: %v", err)
		}
		select {
		case <-waitDone:
			t.Fatalf("gg exited before fake git was ready: %v", waitErr)
		case <-readyDeadline.C:
			t.Fatal("fake git did not become ready")
		case <-poll.C:
		}
	}

	// kill(-pgid, SIGINT) models terminal Ctrl+C delivery to the foreground job.
	if err := syscall.Kill(-pgid, syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT to process group: %v", err)
	}

	select {
	case <-waitDone:
		var ee *exec.ExitError
		if !errors.As(waitErr, &ee) {
			t.Fatalf("gg exit error = %v, want exit status 130", waitErr)
		}
		if got := ee.ExitCode(); got != 130 {
			t.Errorf("gg exit code = %d, want 130 (128+SIGINT)", got)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("gg did not exit after process-group SIGINT")
	}
}
