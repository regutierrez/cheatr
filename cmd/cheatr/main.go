package main

import (
	"cheatr/internal/backend"
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
	b, err := backend.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 0 {
		if err := b.Update(); err != nil {
			fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("updated all sources")
		return
	}

	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "usage: cheatr update [source]")
		os.Exit(2)
	}

	if err := b.UpdateSource(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "update failed for %q: %v\n", args[0], err)
		os.Exit(1)
	}

	fmt.Printf("updated source %q\n", args[0])
}

func runDocsCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: cheatr docs <slug> [search]")
		return
	}

	b, err := backend.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	res, err := b.Resolve(append([]string{"docs"}, args...))
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs resolve failed: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(res.Content) != "" {
		fmt.Println(res.Content)
		return
	}

	if len(res.Candidates) == 0 {
		fmt.Println("No matching docs entries.")
		return
	}

	for _, candidate := range res.Candidates {
		fmt.Printf("- %s (%s)\n", candidate.Title, candidate.Path)
	}
}

func runDirectMode(args []string) {
	fmt.Printf("direct mode: resolve '%s' (placeholder)\n", strings.Join(args, " "))
}
