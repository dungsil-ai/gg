//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// childExitCode는 자식 프로세스의 종료 코드를 반환한다.
// Unix에서 신호로 종료된 경우 shell 관례 128+signal로 매핑한다.
// ExitCode() == -1은 신호 종료를 나타내며 이 경우에만 WaitStatus를 참조한다.
func childExitCode(ee *exec.ExitError) int {
	code := ee.ExitCode()
	if code != -1 {
		return code
	}
	status, ok := ee.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return code
}
