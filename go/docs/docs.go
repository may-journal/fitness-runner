// Package docs holds the runner's Markdown, which people edit and the
// fitness binary embeds: the topics, the llms page and local-branch's rule.
package docs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"github.com/may-journal/fitness-runner/go/pkg/mdx"
	"strings"
)

// files holds each topic, named <topic>.md, and the runner's own pages
// under runner/.
//
//go:embed *.md runner/*.md
var files embed.FS

// RunnerFolder holds the runner's own pages, inside this folder.
const RunnerFolder = "runner"

// Topic is one Markdown file: a `fitness docs` topic or a runner page.
type Topic struct {
	// Name is what `fitness docs <name>` asks for: the file name without .md.
	Name string
	// Summary is the topic's opening paragraph, one sentence on one line.
	Summary string
	// Body is the whole Markdown file, its title first.
	Body string
}

// Prose is the file after its title line, as `fitness explain` prints a
// check's documentation below its own heading.
func (t Topic) Prose() string {
	_, prose, _ := strings.Cut(t.Body, "\n\n")
	return prose
}

// Topics returns every topic the binary embeds, in name order.
func Topics() []Topic {
	// The generator fails a bad file with Read, so none is embedded.
	// If one slips in, say so.
	topics, err := Read(files, ".")
	if err != nil {
		panic(err)
	}
	return topics
}

// Runner returns the runner page of a name the binary embeds: "fitness",
// the page `fitness llms` prints, or "local-branch", that check's rule.
func Runner(name string) Topic {
	t, _ := readTopic(files, path.Join(RunnerFolder, name+".md"))
	return t
}

// Read returns each Markdown file in dir of folder, such as go/docs in a
// checkout, in name order. It fails a missing folder, and a file that does
// not open with its name as the title and then one summary sentence.
func Read(folder fs.FS, dir string) ([]Topic, error) {
	entries, err := fs.ReadDir(folder, dir)
	topics := make([]Topic, 0, len(entries))
	for _, entry := range entries {
		if path.Ext(entry.Name()) != ".md" {
			continue
		}
		t, readErr := readTopic(folder, path.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		topics = append(topics, t)
	}
	return topics, err
}

// readTopic reads one Markdown file, with Windows line endings made Unix
// ones.
func readTopic(folder fs.FS, file string) (Topic, error) {
	raw, err := fs.ReadFile(folder, file)
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	t := Topic{Name: strings.TrimSuffix(path.Base(file), ".md"), Summary: summary(body), Body: body}
	if err == nil && (!strings.HasPrefix(body, "# "+t.Name+"\n\n") || len(mdx.Sentences(t.Summary)) != 1 || strings.HasPrefix(t.Summary, "#")) {
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
