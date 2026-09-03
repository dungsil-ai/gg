//go:build unix

package main

import (
	"errors"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
)

func TestChildExitCodeMapsSignalsToShellStatus(t *testing.T) {
	cases := []struct {
		name string
		sig  syscall.Signal
	}{
		{name: "SIGINT", sig: syscall.SIGINT},
		{name: "SIGTERM", sig: syscall.SIGTERM},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("/bin/sh", "-c", "kill -"+strconv.Itoa(int(tt.sig))+" $$")
			err := cmd.Run()

			var ee *exec.ExitError
			if !errors.As(err, &ee) {
				t.Fatalf("child error = %v, want signal termination", err)
			}
			status, ok := ee.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != tt.sig {
				t.Fatalf("child status = %#v, want %v signal termination", ee.Sys(), tt.sig)
			}
			want := 128 + int(tt.sig)
			if got := childExitCode(ee); got != want {
				t.Errorf("childExitCode = %d, want %d", got, want)
			}
		})
	}
}
