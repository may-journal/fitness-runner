package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	issueSource = regexp.MustCompile(`^(?:[\w.-]+/[\w.-]+)?#\d+$`)
	codeSource  = regexp.MustCompile(`^([^\s:]+):([1-9]\d*)$`)
	acceptID    = regexp.MustCompile(`^(\d{4})\.[1-9]\d*$`)
)

// chainKeywords open the three nested lines under each acceptance ID.
var chainKeywords = []string{"Given", "When", "Then"}

// checkMeasurement requires a numerator, a "-" line, a denominator, and a
// Source line citing one line of code or one issue.
func checkMeasurement(p *problems, root string, s section) {
	if len(s.lines) == 0 {
		return
	}
	if len(s.lines) != 4 || strings.TrimSpace(s.lines[1].text) != "-" {
		p.add(s.n, "Measurement needs four lines: numerator, `-`, denominator, `Source:`")
		return
	}
	checkSourceLine(p, root, s.lines[3])
}

// checkSourceLine requires "Source:" and a valid citation after it.
func checkSourceLine(p *problems, root string, l line) {
	src, ok := strings.CutPrefix(strings.TrimSpace(l.text), "Source:")
	if !ok {
		p.add(l.n, "the last Measurement line must start with `Source:`")
		return
	}
	if msg := checkSource(root, strings.Trim(strings.TrimSpace(src), "`")); msg != "" {
		p.add(l.n, msg)
	}
}

// checkSource accepts one issue reference or one existing path:line, and
// returns the problem otherwise.
func checkSource(root, src string) string {
	if issueSource.MatchString(src) {
		return ""
	}
	m := codeSource.FindStringSubmatch(src)
	if m == nil {
		return "Source must cite either one line of code (`path:line`) or one issue (`#123`)"
	}
	return checkCodeLine(root, m[1], m[2])
}

// checkCodeLine requires a repo file with at least n lines.
func checkCodeLine(root, file, n string) string {
	clean := path.Clean(file)
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
	if strings.HasPrefix(clean, "../") || err != nil {
		return "Source cites " + file + ", which is not a file in this repo"
	}
	want, _ := strconv.Atoi(n)
	if strings.Count(string(b), "\n")+1 < want {
		return "Source cites line " + n + " of " + file + ", which is past its end"
	}
	return ""
}

// item is one list line: its indent width and text after "- ".
type item struct {
	line
	indent int
}

// parseRequirements groups list items under top-level acceptance IDs and
// checks each one's Given, When, Then chain.
func parseRequirements(p *problems, docID string, s section) []idLine {
	if len(s.lines) == 0 {
		return nil
	}
	var out []idLine
	seen := map[string]bool{}
	for _, g := range groupItems(p, s.lines) {
		a := checkAcceptance(p, docID, g)
		if seen[a.id] {
			p.add(a.line, a.id+" appears twice; each acceptance needs its own ID")
		}
		seen[a.id] = true
		out = append(out, a)
	}
	return out
}

// groupItems splits list items into groups, each a top-level item and the
// items nested under it.
func groupItems(p *problems, lines []line) [][]item {
	var groups [][]item
	for _, l := range lines {
		it, ok := toItem(p, l)
		if !ok {
			continue
		}
		if it.indent == 0 || len(groups) == 0 {
			groups = append(groups, nil)
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], it)
	}
	return groups
}

// toItem parses a "- text" line, reporting any other line.
func toItem(p *problems, l line) (item, bool) {
	trimmed := strings.TrimLeft(l.text, " \t")
	text, ok := strings.CutPrefix(trimmed, "- ")
	if !ok {
		p.add(l.n, "Requirements holds only list items")
		return item{}, false
	}
	indent := len(strings.ReplaceAll(l.text[:len(l.text)-len(trimmed)], "\t", "    "))
	return item{line{l.n, strings.TrimSpace(text)}, indent}, true
}

// checkAcceptance requires a top-level NNNN.N ID with the doc's prefix,
// nesting exactly one Given, one When, and one Then.
func checkAcceptance(p *problems, docID string, g []item) idLine {
	top := g[0]
	checkAcceptanceID(p, docID, top)
	if len(g) != len(chainKeywords)+1 {
		p.add(top.n, top.text+" needs exactly one Given, one When, and one Then; a branch is a new ID")
		return idLine{top.text, top.n}
	}
	for i, kw := range chainKeywords {
		checkChainLine(p, kw, g[i], g[i+1])
	}
	return idLine{top.text, top.n}
}

// checkChainLine requires child to start with kw and sit deeper than parent.
func checkChainLine(p *problems, kw string, parent, child item) {
	if !strings.HasPrefix(child.text, kw+" ") {
		p.add(child.n, "this line must start with `"+kw+" `")
	}
	if child.indent <= parent.indent {
		p.add(child.n, "`"+kw+"` must be nested one level under the line above")
	}
}

// checkAcceptanceID requires a top-level `NNNN.N` with the doc's ID.
func checkAcceptanceID(p *problems, docID string, top item) {
	m := acceptID.FindStringSubmatch(top.text)
	if top.indent != 0 || m == nil || m[1] != docID {
		p.add(top.n, "top-level items must be acceptance IDs `"+docID+".N`")
	}
}
