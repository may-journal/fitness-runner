// Package architecture reads the versioned contract beside architecture diagrams.
// It resolves declarations and evidence locations, without claiming that a
// declared relationship has been proved by source or runtime analysis.
package architecture

import (
	"fmt"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/requirement"
)

// Location is a one-based location in tracked repository content.
type Location struct {
	Path string
	Line int
}

func (l Location) String() string { return fmt.Sprintf("%s:%d", l.Path, l.Line) }

// Declaration preserves semantic fields independently of display callouts.
// Ref identifies a declaration in another view. Fields retain typed evidence
// for later checks; a diagram alone never proves a relationship.
type Declaration struct {
	ID, Ref, Type, Parent, Path, Symbol, Name string
	Aliases, Owns, Requirements               []string
	From, To, Kind, Mode, Evidence, Target    string
	View                                      string
	At, Diagram                               Location
}

// Source is an inventory entry. Category never removes a file from ownership.
type Source struct {
	Path, Category, Owner string
	At                    Location
}

// Target is a build target identity, separate from ordinary source paths.
type Target struct {
	ID, Category, Owner string
	At                  Location
}

// Model is the shared input to architecture checks. Evidence is deliberately
// not a boolean proof; child checks must resolve its meaning themselves.
type Model struct {
	Declarations []Declaration
	Sources      []Source
	Targets      []Target
	Requirements []requirement.Document
	Files        []string
	Errors       []string
	diagrams     []diagramGraph
}

func (m *Model) fail(at Location, format string, args ...any) {
	m.Errors = append(m.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

func list(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
