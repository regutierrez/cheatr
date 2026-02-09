package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	switch {
	case len(args) == 0:
		runInteractiveMode()
	case args[0] == "update":
		runUpdateCommand(args[1:])
	case args[0] == "docs":
		runDocsCommand(args[1:])
	default:
		runDirectMode(args)
	}
}

func runInteractiveMode() {
	fmt.Println("interactive mode (placeholder)")
}

func runUpdateCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("update command: refresh all sources (placeholder)")
		return
	}

	fmt.Printf("update command: refresh sources [%s] (placeholder)\n", strings.Join(args, ", "))
}

func runDocsCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: cheatr docs <slug> [search]")
		return
	}

	slug := args[0]
	if len(args) == 1 {
		fmt.Printf("docs command: browse '%s' docs (placeholder)\n", slug)
		return
	}

	search := strings.Join(args[1:], " ")
	fmt.Printf("docs command: search '%s' in '%s' docs (placeholder)\n", search, slug)
}

func runDirectMode(args []string) {
	fmt.Printf("direct mode: resolve '%s' (placeholder)\n", strings.Join(args, " "))
}
