package main

import (
	"os"

	"github.com/sudokatie/api-key-rotate/internal/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	cli.SetVersionInfo(Version, Commit, BuildDate)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
