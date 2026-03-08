package main

import (
	"fmt"
	"os"

	"github.com/chronick/bosun/internal/cli"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	if err := cli.Run(os.Args[1:], Version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
