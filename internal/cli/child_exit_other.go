//go:build !unix

package cli

import "os/exec"

func childExitCode(ee *exec.ExitError) int {
	return ee.ExitCode()
}
