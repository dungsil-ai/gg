package main

import (
	"os"

	"github.com/dungsil-ai/gg/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	os.Exit(cli.Run(os.Args[1:]))
}
