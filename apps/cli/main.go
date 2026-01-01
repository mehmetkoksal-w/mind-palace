package main

import (
	"fmt"
	"os"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "palace: %v\n", err)
		os.Exit(1)
	}
}
