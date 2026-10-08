package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// idLine is an ID and the line that declares it.
type idLine struct {
	id   string
	line int
}

// reqDoc is one parsed requirement doc.
type reqDoc struct {
	path        string
	id          string
	idLine      int
	acceptances []idLine
}

// ids lists the doc's own ID followed by its acceptance IDs.
func (d reqDoc) ids() []idLine {
	return append([]idLine{{d.id, d.idLine}}, d.acceptances...)
}

// line is a non-blank source line with its 1-based number.
type line struct {
	n    int
	text string
}

// section is a level-2 heading and the lines under it.
type section struct {
	name  string
	n     int
	lines []line
}

// sectionNames is every doc's exact section list, in order.
var sectionNames = []string{"Why", "Measurement", "Requirements"}

var (
	fileNamePattern = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	titlePattern    = regexp.MustCompile(`^# (\d{4}) \S`)
	commentPattern  = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// problems collects one doc's errors, each prefixed with its location.
type problems struct {
	path string
	errs []string
}

func (p *problems) add(n int, msg string) {
	p.errs = append(p.errs, at(p.path, n, msg))
}

// at formats a path:line message.
func at(file string, n int, msg string) string {
	return fmt.Sprintf("%s:%d: %s", file, n, msg)
}

// loadDocs reads and parses every requirement doc.
func loadDocs(root string, paths []string) ([]reqDoc, []string, error) {
	var docs []reqDoc
	var errs []string
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return nil, nil, err
		}
		d, docErrs := parseDoc(root, p, string(b))
		docs = append(docs, d)
		errs = append(errs, docErrs...)
	}
	return docs, errs, nil
}

// parseDoc validates one doc's name, title, sections, measurement, and
// acceptances.
func parseDoc(root, file, body string) (reqDoc, []string) {
	p := &problems{path: file}
	head, sections := splitSections(contentLines(body))
	d := reqDoc{path: file}
	d.id, d.idLine = checkNameAndTitle(p, file, head)
	checkSectionNames(p, sections)
	byName := map[string]section{}
	for _, s := range sections {
		byName[s.name] = s
	}
	checkMeasurement(p, root, byName["Measurement"])
	d.acceptances = parseRequirements(p, d.id, byName["Requirements"])
	return d, p.errs
}

// contentLines returns the non-blank lines with comments and front matter
// removed, keeping original line numbers.
func contentLines(body string) []line {
	body = commentPattern.ReplaceAllStringFunc(body, func(c string) string {
		return strings.Repeat("\n", strings.Count(c, "\n"))
	})
	raw := strings.Split(body, "\n")
	var out []line
	for i := frontMatterEnd(raw); i < len(raw); i++ {
		if strings.TrimSpace(raw[i]) != "" {
			out = append(out, line{i + 1, strings.TrimRight(raw[i], " \t\r")})
		}
	}
	return out
}

// frontMatterEnd returns the index just past a leading --- block, or 0.
func frontMatterEnd(raw []string) int {
	if len(raw) == 0 || strings.TrimSpace(raw[0]) != "---" {
		return 0
	}
	for i := 1; i < len(raw); i++ {
		if strings.TrimSpace(raw[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

// splitSections separates the lines before the first ## heading from each
// section.
func splitSections(lines []line) ([]line, []section) {
	var head []line
	var sections []section
	for _, l := range lines {
		if name, ok := strings.CutPrefix(l.text, "## "); ok {
			sections = append(sections, section{name: strings.TrimSpace(name), n: l.n})
			continue
		}
		if len(sections) == 0 {
			head = append(head, l)
			continue
		}
		sections[len(sections)-1].lines = append(sections[len(sections)-1].lines, l)
	}
	return head, sections
}

// checkNameAndTitle requires NNNN-kebab-title.md with one "# NNNN Title"
// line carrying the same ID, and returns that ID and the title's line.
func checkNameAndTitle(p *problems, file string, head []line) (string, int) {
	m := fileNamePattern.FindStringSubmatch(path.Base(file))
	n, title := titleLine(head)
	t := titlePattern.FindStringSubmatch(title)
	if m == nil || t == nil || t[1] != m[1] {
		p.add(n, "name the file NNNN-kebab-title.md and title it `# NNNN Title` with the same ID")
	}
	if m == nil {
		return "", n
	}
	return m[1], n
}

// titleLine returns the one line before the sections, or none.
func titleLine(head []line) (int, string) {
	if len(head) != 1 {
		return 1, ""
	}
	return head[0].n, head[0].text
}

// checkSectionNames requires exactly Why, Measurement, and Requirements, in
// that order, each with content.
func checkSectionNames(p *problems, sections []section) {
	var names []string
	empty := false
	for _, s := range sections {
		names = append(names, s.name)
		empty = empty || len(s.lines) == 0
	}
	if empty || strings.Join(names, ",") != strings.Join(sectionNames, ",") {
		p.add(1, "sections must be exactly ## Why, ## Measurement, ## Requirements, in that order, each with content")
	}
}
