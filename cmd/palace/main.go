package main

import (
	"fmt"
	"os"

	"mind-palace/internal/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.SetBuildInfo(version, commit, date)
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "palace: %v\n", err)
		os.Exit(1)
	}
}
