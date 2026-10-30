package repocheck

import (
	"os"
	"strings"
)

// Doc returns a folder check's README as `fitness explain` prints it: the
// text after its title line, or "" for a flat program or a README that
// cannot be read.
func Doc(c Check) string {
	if c.Readme == "" {
		return ""
	}
	data, err := os.ReadFile(c.Readme)
	if err != nil {
		return ""
	}
	text := strings.TrimLeft(string(data), "\n")
	if strings.HasPrefix(text, "# ") {
		_, text, _ = strings.Cut(text, "\n")
	}
	return strings.TrimSpace(text) + "\n"
}

// Summary is the first sentence of the README's first paragraph, which
// `fitness list` shows for a folder check whose --describe gives none.
func Summary(c Check) string {
	text := firstParagraph(Doc(c))
	if end := strings.Index(text, ". "); end >= 0 {
		return text[:end+1]
	}
	return text
}

// firstParagraph is the first paragraph of prose in doc, its lines joined,
// skipping headings and comments before it.
func firstParagraph(doc string) string {
	var para []string
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		if line == "" && len(para) > 0 {
			break
		}
		if prose(line) {
			para = append(para, line)
		}
	}
	return strings.Join(para, " ")
}

// prose reports whether a README line is prose: not blank, a heading or a
// comment.
func prose(line string) bool {
	return line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "<!--")
}
