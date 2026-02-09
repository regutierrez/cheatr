package main

import (
	"cheatr/internal/backend"
	"cheatr/internal/tui"
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

	runResolvedArgs(append([]string{"docs"}, args...))
}

func runDirectMode(args []string) {
	runResolvedArgs(args)
}

func runResolvedArgs(args []string) {
	b, err := backend.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	res, err := b.Resolve(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve failed: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(res.Content) != "" {
		query := strings.Join(args, " ")
		if err := tui.RunPager(query, res.Source, res.Content); err != nil {
			fmt.Fprintf(os.Stderr, "pager failed: %v\n", err)
			os.Exit(1)
		}
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
