package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/may-journal/fitness-runner/go/docs"
)

// runDocs handles `fitness docs [topic]`: with no topic it lists them, and
// with one it prints that topic's Markdown from go/docs. An unknown topic,
// extra topics, or a flag such as --help exits 2, with the usage or the
// topics.
func runDocs(args []string) int {
	switch {
	case len(args) == 0:
		listTopics(os.Stdout)
		return 0
	case len(args) > 1 || strings.HasPrefix(args[0], "-"):
		fmt.Fprintln(os.Stderr, "fitness docs: name one topic, as `fitness docs [topic]`; `fitness docs` lists them")
		return 2
	}
	return printTopic(args[0])
}

// printTopic prints the named topic, or exits 2 listing the topics when no
// topic has that name.
func printTopic(name string) int {
	if t, ok := docs.Lookup(name); ok {
		fmt.Print(t.Body)
		return 0
	}
	fmt.Fprintf(os.Stderr, "fitness docs: no topic named %q; the topics are:\n", name)
	listTopics(os.Stderr)
	return 2
}

// listTopics writes one aligned line per topic: its name and summary.
func listTopics(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range docs.Topics() {
		fmt.Fprintf(tw, "%s\t%s\n", t.Name, t.Summary)
	}
	_ = tw.Flush()
}
