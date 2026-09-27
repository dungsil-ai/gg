package e2e

import (
	"testing"
)

func isolateCloneConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GG_HOME", t.TempDir())
	t.Setenv("GLAB_CONFIG_DIR", t.TempDir())
	for _, key := range []string{"GH_HOST", "GITLAB_HOST", "GITLAB_URI", "GL_HOST"} {
		t.Setenv(key, "")
	}
}
