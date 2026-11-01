package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// topic is one `fitness docs` topic: the doc comments of the package that
// names it, which `go generate` writes into topics.go.
type topic struct {
	// name is what `fitness docs <name>` asks for.
	name string
	// pkg is the package the topic documents, under go/.
	pkg string
	// summary is the first sentence of the package doc comment.
	summary string
	// body is the package's doc comments as Markdown.
	body string
}

// runDocs handles `fitness docs [topic]`. With no topic it lists them, and
// with one it prints that topic. An unknown topic, more than one, or a flag
// such as --help exits 2. A flag gets the usage, and a name gets the topics.
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
	for _, t := range topics {
		if t.name == name {
			fmt.Printf("# %s\n\nFrom the doc comments of go/%s.\n\n%s", t.name, t.pkg, t.body)
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "fitness docs: no topic named %q; the topics are:\n", name)
	listTopics(os.Stderr)
	return 2
}

// listTopics writes one aligned line per topic: its name and summary.
func listTopics(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range topics {
		fmt.Fprintf(tw, "%s\t%s\n", t.name, t.summary)
	}
	_ = tw.Flush()
}
