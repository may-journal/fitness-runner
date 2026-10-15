package architecture

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	codeSpan     = regexp.MustCompile("`([^`]+)`")
	markdownLink = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	acceptanceID = regexp.MustCompile(`^\d{4}\.\d+$`)
	lineAnchor   = regexp.MustCompile(`^L(\d+)(?:-L(\d+))?$`)
	docNumber    = regexp.MustCompile(`^(\d{4})-`)
)

// rowContent reads what a callout row already says: code spans that name
// repository paths or acceptance IDs, and links to sources or requirements.
func (d *diagram) rowContent(s subject, text string, at Location) {
	for _, match := range codeSpan.FindAllStringSubmatch(text, -1) {
		d.span(s, match[1], at)
	}
	for _, match := range markdownLink.FindAllStringSubmatch(text, -1) {
		d.link(s, match[1], at)
	}
}

func (d *diagram) span(s subject, text string, at Location) {
	if acceptanceID.MatchString(text) {
		d.m.requirementRef(text, at)
		return
	}
	d.claim(s.owner(), text, true, at)
}

// claim records a path an element names. A named path must exist; an edge
// row's path is checked but owns nothing. Only a code span or label may name
// a root-level source by its bare name.
func (d *diagram) claim(owner, text string, bare bool, at Location) {
	d.mention(owner, text, bare, at)
	p, ok := d.m.repo.candidate(text, bare)
	if !ok {
		return
	}
	if !d.m.repo.exists(p) {
		d.m.fail(at, "unresolved source path %s", p)
		return
	}
	if owner != "" {
		d.claims = append(d.claims, Claim{Owner: owner, Path: p, At: at})
	}
}

// mention records a word an element names for the containers check,
// without the punctuation that ends a sentence, as in "builds dist/esm.".
func (d *diagram) mention(owner, text string, named bool, at Location) {
	text = strings.TrimLeft(strings.TrimRight(text, ".,;:)]}'\""), "([{'\"")
	if owner != "" && text != "" {
		d.mentions = append(d.mentions, Mention{Owner: owner, Text: path.Clean(text), Named: named, At: at})
	}
}

func (d *diagram) link(s subject, target string, at Location) {
	if strings.Contains(target, ":") || strings.HasPrefix(target, "#") {
		return
	}
	file, anchor, _ := strings.Cut(target, "#")
	p := path.Clean(path.Join(path.Dir(d.file), file))
	if id := requirementDoc(p); id != "" {
		d.m.requirementRef(id, at)
		return
	}
	if sourceFile(p) {
		d.evidence(s, p, anchor, at)
	}
}

func requirementDoc(p string) string {
	if match := docNumber.FindStringSubmatch(path.Base(p)); match != nil && path.Dir(p) == "docs/requirements" {
		return match[1]
	}
	return ""
}

// evidence records a static proof location. Runtime proof has no form in
// the diagrams, so nothing here proves runtime behavior.
func (d *diagram) evidence(s subject, p, anchor string, at Location) {
	first, last, ok := d.anchorLines(p, anchor)
	if !ok || !d.m.repo.files[p] {
		d.m.fail(at, "unresolved evidence %s", strings.TrimSuffix(p+"#"+anchor, "#"))
		return
	}
	d.m.Evidence = append(d.m.Evidence, Evidence{Subject: s.name(), Path: p, Line: first, End: last, At: at})
}

func (d *diagram) anchorLines(p, anchor string) (int, int, bool) {
	if anchor == "" {
		return 0, 0, true
	}
	match := lineAnchor.FindStringSubmatch(anchor)
	if match == nil {
		return 0, 0, false
	}
	first, _ := strconv.Atoi(match[1])
	last := first
	if match[2] != "" {
		last, _ = strconv.Atoi(match[2])
	}
	return first, last, inRange(first, last, d.m.repo.lines(p))
}

func inRange(first, last, size int) bool {
	return first >= 1 && first <= last && last <= size
}

func (m *Model) requirementRef(id string, at Location) {
	for _, doc := range m.Requirements {
		for _, known := range doc.IDs() {
			if known.ID == id {
				return
			}
		}
	}
	m.fail(at, "unresolved requirement %s in docs/requirements", id)
}
