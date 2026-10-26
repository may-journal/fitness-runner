package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// fitnessKeys are the front matter keys only fitness ever read. Both go
// from every block; any other key belongs to a tool that may still need it,
// such as GitHub's issue template fields, and stays.
var fitnessKeys = []string{"fitnessFunctions", "relatedConfigurations"}

var (
	// plainFront is a leading --- block and the blank lines after it. The
	// block is empty or opens on a non-blank line, so a rule followed by
	// a blank line never matches.
	plainFront = regexp.MustCompile(`\A---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n(?:\r?\n)*`)
	// commentFront is the same block inside an HTML comment that opens on
	// the first line, the form a PR template carries.
	commentFront = regexp.MustCompile(`\A<!--\r?\n---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n-->\r?\n(?:\r?\n)*`)
)

// runStripFrontMatter runs `fitness strip-front-matter`: it deletes the
// fitness keys from every tracked Markdown file's front matter, drops a
// block left with no key, and prints one line per file it changed.
func runStripFrontMatter(args []string) int {
	root, err := os.Getwd()
	if err == nil && len(args) > 0 {
		err = fmt.Errorf("takes no arguments, got %s", strings.Join(args, " "))
	}
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	stripped, err := stripAll(root)
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	fmt.Printf("fitness strip-front-matter: stripped %d files\n", stripped)
	return 0
}

// stripAll strips each Markdown file under root and counts the files it
// changed.
func stripAll(root string) (int, error) {
	stripped := 0
	for _, rel := range walkfs.FilesByExt(root, ".md") {
		changed, err := stripFile(root, rel)
		if err != nil {
			return stripped, err
		}
		if changed {
			stripped++
		}
	}
	return stripped, nil
}

// stripFile strips one file, prints what it did, and reports whether it
// changed the file.
func stripFile(root, rel string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	body, others := stripFront(string(data))
	if body == string(data) {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, err
	}
	line := "stripped " + rel
	if len(others) > 0 {
		line += ": front matter keeps " + strings.Join(others, ", ")
	}
	fmt.Println(line)
	return true, nil
}

// byteOrderMark is the UTF-8 BOM some editors put before a file's first line.
const byteOrderMark = "\ufeff"

// keyLine is a top-level front matter entry: a name, then a colon.
var keyLine = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\s*:`)

// stripFront returns content with the fitness keys gone from its front
// matter. A block left with no key, or empty to begin with, goes whole.
// When other keys stay, it returns the block without the fitness keys and
// names the keys that stay. Content without fitness keys, or whose leading
// --- opens a rule rather than front matter, comes back unchanged with no
// names. A BOM before the block stays at the top of the body.
func stripFront(content string) (string, []string) {
	bom := ""
	if strings.HasPrefix(content, byteOrderMark) {
		bom, content = byteOrderMark, strings.TrimPrefix(content, byteOrderMark)
	}
	for _, re := range []*regexp.Regexp{plainFront, commentFront} {
		if body, others, ok := stripWith(re, content); ok {
			return bom + body, others
		}
	}
	return bom + content, nil
}

// stripWith applies one block form to content. It reports false when the
// form does not hold front matter that is safe to decide on.
func stripWith(re *regexp.Regexp, content string) (string, []string, bool) {
	m := re.FindStringSubmatchIndex(content)
	if m == nil {
		return content, nil, false
	}
	start, end := innerSpan(m)
	entries, ok := frontEntries(content[start:end])
	kept, others, fitness := sortEntries(entries)
	switch {
	case !ok, !fitness && len(others) > 0:
		return content, nil, false
	case len(others) == 0:
		return content[m[1]:], nil, true
	}
	return content[:start] + kept + content[end:], others, true
}

// innerSpan is where the lines between a matched block's --- markers start
// and end; an empty block has none, so both sit at the block's start.
func innerSpan(m []int) (int, int) {
	if m[2] < 0 {
		return m[0], m[0]
	}
	return m[2], m[3]
}

// sortEntries joins the text of every entry that stays, names the other
// tools' keys among them, and reports whether a fitness key was there.
func sortEntries(entries []entry) (kept string, others []string, fitness bool) {
	for _, e := range entries {
		if slices.Contains(fitnessKeys, e.name) {
			fitness = true
			continue
		}
		kept += e.text
		if e.name != "" {
			others = append(others, e.name)
		}
	}
	return kept, others, fitness
}

// entry is one top-level front matter key and the text it spans, its own
// line through its last continuation line, line endings included.
type entry struct {
	name string
	text string
}

// frontEntries splits the lines between a block's --- markers into its
// top-level entries in order. Indented lines, list items, comments and
// blank lines belong to the entry above; those before the first key form an
// entry with no name. It reports false when a line is neither an entry nor
// a continuation, as with prose between two horizontal rules.
func frontEntries(inner string) ([]entry, bool) {
	var entries []entry
	for _, line := range strings.SplitAfter(inner, "\n") {
		var ok bool
		if entries, ok = addLine(entries, line); !ok {
			return nil, false
		}
	}
	return entries, true
}

// addLine adds one line to entries: a key opens a new entry, and any other
// line joins the entry above. It reports false for a line that can be
// neither.
func addLine(entries []entry, line string) ([]entry, bool) {
	bare := strings.TrimRight(line, "\r\n")
	switch {
	case keyLine.MatchString(bare):
		name, _, _ := strings.Cut(bare, ":")
		return append(entries, entry{name: strings.TrimSpace(name), text: line}), true
	case !continuesEntry(bare):
		return entries, false
	case len(entries) == 0:
		return append(entries, entry{text: line}), true
	}
	entries[len(entries)-1].text += line
	return entries, true
}

// continuesEntry reports whether a line belongs to the entry above it: it
// is blank, indented, a list item or a comment.
func continuesEntry(line string) bool {
	return line == "" || strings.ContainsAny(line[:1], " \t-#")
}
