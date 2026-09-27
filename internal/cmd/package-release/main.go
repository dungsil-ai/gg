// Command package-release builds the release files from the repository root.
package main

import (
	"fmt"
	"os"

	"github.com/dungsil-ai/gg/internal/cli"
)

func main() {
	tag, outDir := os.Getenv("GG_RELEASE_VERSION"), os.Getenv("GG_RELEASE_OUT_DIR")
	if tag == "" || outDir == "" {
		fmt.Fprintln(os.Stderr, "GG_RELEASE_VERSION and GG_RELEASE_OUT_DIR are required")
		os.Exit(2)
	}
	if err := cli.BuildAndPackageRelease(".", outDir, tag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
