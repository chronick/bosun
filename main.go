package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("bosun: agent entrypoint and lifecycle coordinator")
	fmt.Println("usage: bosun <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  run                  full lifecycle loop")
	fmt.Println("  register             announce identity to agent-mail")
	fmt.Println("  claim                pick highest-priority ready task")
	fmt.Println("  lease <task-id>      acquire file lease")
	fmt.Println("  release <task-id>    release file lease")
	fmt.Println("  heartbeat            ping agent-mail with status")
	fmt.Println("  complete <task-id>   close task, create PR, release lease")
	return nil
}
