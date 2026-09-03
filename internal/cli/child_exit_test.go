package cli

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

// TestChildExitCodePassthroughExited는 정상 exit code가 그대로 반환됨을 검증한다.
func TestChildExitCodePassthroughExited(t *testing.T) {
	const helperEnv = "GG_TEST_CHILD_EXIT_CODE"
	if v := os.Getenv(helperEnv); v != "" {
		code, _ := strconv.Atoi(v)
		os.Exit(code)
	}

	for _, want := range []int{0, 1, 42} {
		want := want
		t.Run("exit"+strconv.Itoa(want), func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestChildExitCodePassthroughExited$")
			cmd.Env = append(os.Environ(), helperEnv+"="+strconv.Itoa(want))
			err := cmd.Run()
			if want == 0 {
				if err != nil {
					t.Fatalf("exit 0: got err %v", err)
				}
				return
			}
			var ee *exec.ExitError
			if !errors.As(err, &ee) {
				t.Fatalf("exit %d: got %v, want *exec.ExitError", want, err)
			}
			if got := childExitCode(ee); got != want {
				t.Errorf("childExitCode = %d, want %d", got, want)
			}
		})
	}
}
