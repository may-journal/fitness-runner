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
// Technology and Description are the C4 strings after the label; only
// containers and components carry a technology.
type Element struct {
	ID, Kind, Name, Technology, Description, Parent, View string
	Callout                                               int
	At                                                    Location
	// Members are a class body's lines; Annotations its <<name>> lines.
	Members, Annotations []Member
}

// Member is one line a class diagram states about a class.
type Member struct {
	Text string
	At   Location
}

// Relationship is a directed diagram edge between two element IDs.
type Relationship struct {
	// Arrow is the Mermaid arrow or C4 macro that drew the edge.
	From, To, Arrow, Label, View string
	Callout                      int
	At                           Location
}

// Claim is a repository path named by an element's diagram text or callout
// row. Owner is the element's ID.
type Claim struct {
	Owner, Path string
	At          Location
}

// Mention is a word an element's text names, tracked or not, such as
// the build output dist/esm or a package name. Named marks a code span or
// label; a technology or description word is only prose.
type Mention struct {
	Owner, Text string
	Named       bool
	At          Location
}

// Evidence is a callout row link to a tracked source file, with an optional
// line range. Subject is the element ID, or "from->to" for a relationship.
type Evidence struct {
	Subject, Path string
	Line, End     int
	At            Location
}

// String is the link as the row wrote it: the path and its line anchor.
func (e Evidence) String() string {
	switch {
	case e.Line == 0:
		return e.Path
	case e.End > e.Line:
		return fmt.Sprintf("%s#L%d-L%d", e.Path, e.Line, e.End)
	}
	return fmt.Sprintf("%s#L%d", e.Path, e.Line)
}

// holds reports whether line is in the linked range; no anchor holds
// every line.
func (e Evidence) holds(line int) bool {
	return e.Line == 0 || (line >= e.Line && line <= e.End)
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
	Mentions      []Mention
	Evidence      []Evidence
	Sources       []Source
	Requirements  []requirement.Document
	Files         []string
	Errors        []string
	repo          repository
	// journeyAt is the first journey diagram's fence line, or the journey
	// view's first line when it has no diagram.
	journeyAt Location
	// contextAt is the first context diagram's fence line.
	contextAt Location
}

// contextLocation is the first context diagram, or the first view when no
// context view exists, for errors about undeclared actors.
func (m *Model) contextLocation() Location {
	if m.contextAt.Path == "" && len(m.Files) > 0 {
		return Location{m.Files[0], 1}
	}
	return m.contextAt
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
