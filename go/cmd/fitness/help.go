package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// usageEntry is one documented form of the command line, read from its
// comment by `go generate` into usage.go.
type usageEntry struct {
	// synopses are the forms, after `fitness`.
	synopses []string
	// summary says what the form does.
	summary string
}

// printUsage writes the command summary, generated from the comments on
// the forms that run checks and on each subcommands table entry.
func printUsage() int {
	fmt.Println("fitness — run the configured checks.")
	fmt.Println("\nRun checks:")
	writeEntries(os.Stdout, runnerUsage)
	fmt.Println("\nCommands:")
	writeEntries(os.Stdout, commandUsage)
	fmt.Println("\nNew here, or an AI? Run `fitness llms`: one page that explains the whole CLI.")
	return 0
}

// writeEntries writes each entry's forms, then its summary indented below.
func writeEntries(w io.Writer, entries []usageEntry) {
	for _, e := range entries {
		for _, s := range e.synopses {
			fmt.Fprintln(w, "  "+strings.TrimSpace("fitness "+s))
		}
		fmt.Fprintln(w, "      "+e.summary)
	}
}

// runLLMs handles `fitness llms`: the page that teaches an AI the whole CLI,
// the same one llms.txt holds.
func runLLMs(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "fitness llms: takes no arguments")
		return 2
	}
	fmt.Print(llmsPage)
	return 0
}
