// Package docs holds the `fitness docs` topics as Markdown, one file each,
// which people edit directly and the fitness binary embeds.
package docs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// files holds each topic, named <topic>.md.
//
//go:embed *.md
var files embed.FS

// Topic is one `fitness docs` topic.
type Topic struct {
	// Name is what `fitness docs <name>` asks for: the file name without .md.
	Name string
	// Summary is the topic's opening paragraph, one sentence on one line.
	Summary string
	// Body is the whole Markdown file, its title first.
	Body string
}

// Topics returns every topic the binary embeds, in name order.
func Topics() []Topic {
	// The generator reads the same files with Read and fails a bad one, so
	// the binary never embeds a topic Read refuses.
	topics, _ := Read(files)
	return topics
}

// Read returns each topic in a folder of Markdown files, such as go/docs in
// a checkout, in name order. It fails a missing folder, and a file that does
// not open with its name as the title and then one summary sentence.
func Read(folder fs.FS) ([]Topic, error) {
	entries, err := fs.ReadDir(folder, ".")
	topics := make([]Topic, 0, len(entries))
	for _, entry := range entries {
		if path.Ext(entry.Name()) != ".md" {
			continue
		}
		t, readErr := readTopic(folder, entry.Name())
		if readErr != nil {
			return nil, readErr
		}
		topics = append(topics, t)
	}
	return topics, err
}

// readTopic reads one topic file, with Windows line endings made Unix ones.
func readTopic(folder fs.FS, file string) (Topic, error) {
	raw, err := fs.ReadFile(folder, file)
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	t := Topic{Name: strings.TrimSuffix(file, ".md"), Summary: summary(body), Body: body}
	if err == nil && (!strings.HasPrefix(body, "# "+t.Name+"\n\n") || strings.Contains(t.Summary, ". ")) {
		err = fmt.Errorf("go/docs/%s: open with the title %q, a blank line, then one summary sentence", file, "# "+t.Name)
	}
	return t, err
}

// Lookup returns the topic of a name, reporting whether there is one.
func Lookup(name string) (Topic, bool) {
	for _, t := range Topics() {
		if t.Name == name {
			return t, true
		}
	}
	return Topic{}, false
}

// summary is the first paragraph after a Markdown file's title, joined onto
// one line.
func summary(body string) string {
	_, rest, _ := strings.Cut(body, "\n\n")
	first, _, _ := strings.Cut(rest, "\n\n")
	return strings.Join(strings.Fields(first), " ")
}
