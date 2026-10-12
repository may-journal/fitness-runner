// Package architecture reads one model from the diagrams and callout tables
// already in each architecture view. Mermaid IDs are the stable identities;
// callout numbers stay display references. Nothing here is a proof: an edge
// counts as proved only when later checks resolve its evidence.
package architecture

import (
	"fmt"
	"sort"

	"github.com/may-journal/fitness-runner/go/internal/requirement"
)

// Location is a one-based line in tracked repository content.
type Location struct {
	Path string
	Line int
}

func (l Location) String() string { return fmt.Sprintf("%s:%d", l.Path, l.Line) }

// Element is a node or boundary. ID is its Mermaid ID, Kind its C4 macro,
// flowchart class, or "class", and Parent the enclosing boundary's ID.
type Element struct {
	ID, Kind, Name, Parent, View string
	Callout                      int
	At                           Location
}

// Relationship is a directed diagram edge between two element IDs.
type Relationship struct {
	From, To, Label, View string
	Callout               int
	At                    Location
}

// Claim is a repository path named by an element's diagram text or callout
// row. Owner is the element's ID.
type Claim struct {
	Owner, Path string
	At          Location
}

// Evidence is a callout row link to a tracked source file, with an optional
// line. Subject is the element ID, or "from->to" for a relationship.
type Evidence struct {
	Subject, Path string
	Line          int
	At            Location
}

// Source is one tracked source file, its category, and the elements whose
// claims cover it. Categories never remove a file from the inventory.
type Source struct {
	Path, Category string
	Owners         []string
}

// Model is the shared input to architecture checks.
type Model struct {
	Elements      []Element
	Relationships []Relationship
	Claims        []Claim
	Evidence      []Evidence
	Sources       []Source
	Mappings      []Mapping
	Requirements  []requirement.Document
	Files         []string
	Errors        []string
	repo          repository
	// journeyAt is the first requirement table's header line, or the
	// journey view's first line when it has none.
	journeyAt Location
}

func (m *Model) fail(at Location, format string, args ...any) {
	m.Errors = append(m.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
