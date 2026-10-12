package architecture

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Mapping is one row of a journey view's requirement table: the requirement
// the row names, the journey node IDs it lists, and the row's line.
type Mapping struct {
	Doc   string
	Nodes []string
	At    Location
}

var (
	separatorRow = regexp.MustCompile(`^\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
	cellLink     = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)\)`)
	docID        = regexp.MustCompile(`^\d{4}$`)
)

const journeyView = "journey"

// readMappings reads each requirement table of a journey view: a table whose
// first header cell is Requirement. Each row names one requirement doc and
// the journey nodes that fulfil it.
func (m *Model) readMappings(file string, lines []string) {
	if m.journeyAt == (Location{}) {
		m.journeyAt = Location{file, 1}
	}
	for i := 0; i+1 < len(lines); i++ {
		if !requirementHeader(lines[i], lines[i+1]) {
			continue
		}
		if len(m.Mappings) == 0 {
			m.journeyAt = Location{file, i + 1}
		}
		i = m.mappingRows(file, lines, i+2)
	}
}

// requirementHeader reports a table header whose first cell is Requirement,
// followed by its separator row.
func requirementHeader(line, next string) bool {
	c := cells(line)
	return strings.Contains(line, "|") && strings.EqualFold(c[0], "Requirement") && separatorRow.MatchString(strings.TrimSpace(next))
}

// cells splits a table row into trimmed cells without its outer pipes.
func cells(line string) []string {
	inner := strings.TrimSpace(line)
	inner = strings.TrimSuffix(strings.TrimPrefix(inner, "|"), "|")
	parts := strings.Split(inner, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// mappingRows reads the rows from start and returns the last row's index.
func (m *Model) mappingRows(file string, lines []string, start int) int {
	i := start
	for ; i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != ""; i++ {
		row := append(cells(lines[i]), "", "")
		m.Mappings = append(m.Mappings, Mapping{Doc: docRef(file, row[0]), Nodes: ids(row[1]), At: Location{file, i + 1}})
	}
	return i - 1
}

// docRef reads the requirement cell. A link to a requirement doc names the
// doc by its file; any other text is read as written.
func docRef(file, cell string) string {
	match := cellLink.FindStringSubmatch(cell)
	if match == nil {
		return strings.Trim(cell, "` ")
	}
	target, _, _ := strings.Cut(match[2], "#")
	if id := requirementDoc(path.Clean(path.Join(path.Dir(file), target))); id != "" {
		return id
	}
	return strings.TrimSpace(match[1])
}

// ids reads the comma-separated node IDs of a journey cell.
func ids(cell string) []string {
	var out []string
	for _, part := range strings.Split(cell, ",") {
		if id := strings.Trim(part, "` "); id != "" {
			out = append(out, id)
		}
	}
	return out
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

// JourneyErrors compares the journey's requirement tables with every
// requirement doc and every behavior node, in both directions. Without a
// journey view there is no table to judge; the contract names the missing
// view.
func (m *Model) JourneyErrors() []string {
	if m.journeyAt == (Location{}) {
		return nil
	}
	j := &journey{m: m, rows: map[string]Mapping{}, mapped: map[string]bool{}}
	for _, row := range m.Mappings {
		j.row(row)
	}
	j.docs()
	j.nodes()
	return j.errs
}

// journey holds one comparison: each requirement's first row and the node
// IDs some row names.
type journey struct {
	m      *Model
	rows   map[string]Mapping
	mapped map[string]bool
	errs   []string
}

func (j *journey) fail(at Location, format string, args ...any) {
	j.errs = append(j.errs, at.String()+": "+fmt.Sprintf(format, args...))
}

func (j *journey) row(r Mapping) {
	if !j.validDoc(r) {
		return
	}
	if first, again := j.rows[r.Doc]; again {
		j.fail(r.At, "requirement %s is mapped again; first at %s", r.Doc, first.At)
		return
	}
	j.rows[r.Doc] = r
	j.rowNodes(r)
}

// validDoc requires the row to name a requirement doc that exists. An
// acceptance ID names a test's promise, not a doc.
func (j *journey) validDoc(r Mapping) bool {
	switch {
	case acceptanceID.MatchString(r.Doc):
		j.fail(r.At, "%s is an acceptance ID; map requirement doc %s", r.Doc, r.Doc[:4])
	case !docID.MatchString(r.Doc):
		j.fail(r.At, "%q is not a requirement doc ID such as 0001", r.Doc)
	case !j.m.hasDoc(r.Doc):
		j.fail(r.At, "no requirement doc %s in docs/requirements", r.Doc)
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

func (j *journey) rowNodes(r Mapping) {
	if len(r.Nodes) == 0 {
		j.fail(r.At, "requirement %s maps no journey node", r.Doc)
	}
	for _, id := range r.Nodes {
		if !j.m.journeyNode(id) {
			j.fail(r.At, "requirement %s maps unknown journey node %s", r.Doc, id)
			continue
		}
		j.mapped[id] = true
	}
}

func (m *Model) journeyNode(id string) bool {
	for _, e := range m.Elements {
		if e.View == journeyView && e.ID == id {
			return true
		}
	}
	return false
}

// docs requires a row for every requirement doc. A doc without a valid ID
// belongs to the requirements check.
func (j *journey) docs() {
	for _, doc := range j.m.Requirements {
		if _, ok := j.rows[doc.ID]; !ok && doc.ID != "" {
			j.fail(j.m.journeyAt, "requirement %s has no row in the journey's requirement table", doc.ID)
		}
	}
}

// nodes requires a row for every node that promises behavior. Personas and
// layout subgraphs promise none.
func (j *journey) nodes() {
	for _, e := range j.m.Elements {
		if e.View == journeyView && behavior(e.Kind) && !j.mapped[e.ID] {
			j.fail(e.At, "journey node %s maps to no requirement; list it in a requirement row or type it as a persona", e.ID)
		}
	}
}

func behavior(kind string) bool {
	return kind != "subgraph" && !strings.HasPrefix(strings.ToLower(kind), "person")
}
