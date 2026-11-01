package architecture

import (
	"fmt"
	"regexp"
	"strings"
)

// maxLabelIDs is the most requirement IDs one journey label may name, so a
// step that fulfils more is split.
const maxLabelIDs = 8

// labelID is a requirement number, or an acceptance ID, written in a
// journey label after its callout number.
var labelID = regexp.MustCompile(`\b\d{4}(?:\.\d+)?\b`)

// entityCode is a Mermaid character code such as #9829; in a label, which
// names no requirement.
var entityCode = regexp.MustCompile(`#\w+;`)

const journeyView = "journey"

// journeyStart records where the journey begins: its first diagram's fence
// line, or the view's first line when it has no diagram.
func (m *Model) journeyStart(at Location) {
	if m.journeyAt == (Location{}) {
		m.journeyAt = at
	}
}

// JourneyFiles counts the files the journey check judges: journey views and
// requirement docs.
func (m *Model) JourneyFiles() int {
	count := len(m.Requirements)
	for _, file := range m.Files {
		if viewFile(file) == journeyView {
			count++
		}
	}
	return count
}

// JourneyErrors compares requirement IDs named in journey labels with
// requirement docs, both ways, and requires each behavior node to name one.
func (m *Model) JourneyErrors() []string {
	if m.journeyAt == (Location{}) {
		return nil
	}
	j := &journey{m: m, named: map[string]bool{}, nodes: map[string]bool{}}
	j.labels()
	j.docs()
	j.behavior()
	return j.errs
}

// labels reads every journey label: elements first, then edges.
func (j *journey) labels() {
	for _, e := range j.m.Elements {
		if e.View == journeyView {
			j.label(e.ID, e.Name, e.At)
		}
	}
	for _, r := range j.m.Relationships {
		if r.View == journeyView {
			j.label("", r.Label, r.At)
		}
	}
}

// journey holds one comparison: the docs some label names and the elements
// whose own label names one.
type journey struct {
	m     *Model
	named map[string]bool
	nodes map[string]bool
	errs  []string
}

func (j *journey) fail(at Location, format string, args ...any) {
	j.errs = append(j.errs, at.String()+": "+fmt.Sprintf(format, args...))
}

// label reads the requirement IDs one label names. owner is the element the
// label belongs to; an edge label has none. A label that names any ID
// covers its node, so a mistyped ID fails once, as unknown.
func (j *journey) label(owner, text string, at Location) {
	ids := uniqueIDs(labelID.FindAllString(entityCode.ReplaceAllString(text, ""), -1))
	if len(ids) > maxLabelIDs {
		j.fail(at, "label names %d requirements; split the step so each names at most %d", len(ids), maxLabelIDs)
	}
	for _, id := range ids {
		j.named[id] = j.valid(id, at)
	}
	if owner != "" && len(ids) > 0 {
		j.nodes[owner] = true
	}
}

func uniqueIDs(ids []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// valid requires a label ID to name a requirement doc that exists. An
// acceptance ID names a test's promise, not a doc.
func (j *journey) valid(id string, at Location) bool {
	switch {
	case acceptanceID.MatchString(id):
		j.fail(at, "%s is an acceptance ID; name requirement doc %s", id, id[:4])
	case !j.m.hasDoc(id):
		j.fail(at, "no requirement doc %s in docs/requirements", id)
	default:
		return true
	}
	return false
}

func (m *Model) hasDoc(id string) bool {
	for _, doc := range m.Requirements {
		if doc.ID == id {
			return true
		}
	}
	return false
}

// docs requires every requirement doc in some journey label. A doc without
// a valid ID belongs to the requirements check.
func (j *journey) docs() {
	for _, doc := range j.m.Requirements {
		if doc.ID != "" && !j.named[doc.ID] {
			j.fail(j.m.journeyAt, "requirement %s is missing from the journey diagram; name it in the label of the step that fulfils it", doc.ID)
		}
	}
}

// behavior requires every node that promises behavior to name a
// requirement in its own label or an enclosing subgraph's. Personas and
// layout subgraphs promise none.
func (j *journey) behavior() {
	for _, e := range j.m.Elements {
		if e.View == journeyView && promises(e.Kind) && !j.covered(e) {
			j.fail(e.At, "journey node %s names no requirement; add the IDs it fulfils to its label or type it as a persona", e.ID)
		}
	}
}

// covered reports a requirement ID in the element's label or in the label
// of a subgraph around it.
func (j *journey) covered(e Element) bool {
	for id := e.ID; id != ""; id = j.parent(id) {
		if j.nodes[id] {
			return true
		}
	}
	return false
}

func (j *journey) parent(id string) string {
	for _, e := range j.m.Elements {
		if e.View == journeyView && e.ID == id {
			return e.Parent
		}
	}
	return ""
}

func promises(kind string) bool {
	return kind != "subgraph" && !strings.HasPrefix(strings.ToLower(kind), "person")
}
