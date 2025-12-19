package main

import (
	"fmt"
	"os"

	"mind-palace/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "palace: %v\n", err)
		os.Exit(1)
	}
}
