package architecture

import (
	"os"
	"path/filepath"
	"strings"
)

func (m *Model) validate(root string, files []string) {
	ids := m.index()
	for _, d := range m.Declarations {
		m.references(d, ids)
		m.nodeFields(d)
		m.evidence(root, d, files)
	}
	m.aliases()
	m.requirementLinks()
	m.inventory(root, files)
	m.manifestTargets(root, files)
}

func (m *Model) index() map[string]Declaration {
	ids := map[string]Declaration{}
	for _, d := range m.Declarations {
		if d.ID == "" {
			continue
		}
		if first, exists := ids[d.ID]; exists {
			m.fail(d.At, "duplicate ID %s; first declared at %s", d.ID, first.At)
		}
		ids[d.ID] = d
	}
	return ids
}

func (m *Model) references(d Declaration, ids map[string]Declaration) {
	for _, ref := range []string{d.Ref, d.Parent, d.From, d.To} {
		if ref == "" {
			continue
		}
		if _, exists := ids[ref]; !exists {
			m.fail(d.At, "unresolved ID %s (diagram %s)", ref, d.Diagram)
		}
	}
	m.parentCycle(d, ids)
}

func (m *Model) parentCycle(d Declaration, ids map[string]Declaration) {
	seen := map[string]bool{d.ID: true}
	for parent := d.Parent; parent != ""; parent = ids[parent].Parent {
		if seen[parent] {
			m.fail(d.At, "parent cycle through %s at %s", parent, ids[parent].At)
			return
		}
		seen[parent] = true
	}
}

func (m *Model) nodeFields(d Declaration) {
	if d.Ref != "" {
		return
	}
	if !contains(strings.Fields("actor system external runtime component symbol behavior relationship"), d.Type) {
		m.fail(d.At, "unsupported node type %q", d.Type)
	}
	m.requiredFields(d)
}

func (m *Model) requiredFields(d Declaration) {
	if contains([]string{"runtime", "component", "symbol"}, d.Type) && d.Parent == "" {
		m.fail(d.At, "%s needs a parent ID", d.Type)
	}
	if d.Type == "actor" && d.Name == "" {
		m.fail(d.At, "actor needs a name")
	}
	m.symbolFields(d)
	m.relationshipFields(d)
}

func (m *Model) symbolFields(d Declaration) {
	if d.Type == "symbol" && (d.Path == "" || d.Symbol == "") {
		m.fail(d.At, "symbol needs path and symbol fields")
	}
}

func (m *Model) relationshipFields(d Declaration) {
	if d.Type != "relationship" {
		return
	}
	if d.From == "" || d.To == "" {
		m.fail(d.At, "relationship needs directed from and to IDs")
	}
	if !contains(strings.Fields("uses calls imports inherits implements contains reads writes sends navigates"), d.Kind) {
		m.fail(d.At, "unsupported relationship kind %q", d.Kind)
	}
	m.relationshipProof(d)
}

func (m *Model) relationshipProof(d Declaration) {
	if !contains([]string{"allowed", "required"}, d.Mode) {
		m.fail(d.At, "relationship mode must be allowed or required")
	}
	if d.Evidence == "" {
		m.fail(d.At, "relationship needs evidence; use declared for an unproved constraint")
	}
}

func (m *Model) aliases() {
	seen := map[string]Declaration{}
	for _, d := range m.Declarations {
		if d.Type == "actor" {
			m.actorNames(d, seen)
		}
	}
}

func (m *Model) actorNames(d Declaration, seen map[string]Declaration) {
	for _, name := range append([]string{d.Name}, d.Aliases...) {
		if first, exists := seen[name]; exists {
			m.fail(d.At, "actor name or alias %q is ambiguous; first at %s", name, first.At)
		}
		seen[name] = d
	}
}

func (m *Model) requirementLinks() {
	ids := map[string]Location{}
	for _, doc := range m.Requirements {
		for _, id := range doc.IDs() {
			ids[id.ID] = Location{doc.Path, id.Line}
		}
	}
	for _, d := range m.Declarations {
		m.requirementRefs(d, ids)
	}
}

func (m *Model) requirementRefs(d Declaration, ids map[string]Location) {
	for _, id := range d.Requirements {
		if _, exists := ids[id]; !exists {
			m.fail(d.At, "unresolved requirement %s in docs/requirements (diagram %s)", id, d.Diagram)
		}
	}
}

func (m *Model) evidence(root string, d Declaration, files []string) {
	m.sourcePath(d, files)
	if d.Evidence == "" || d.Evidence == "declared" {
		return
	}
	kind, location, _ := strings.Cut(d.Evidence, ":")
	if kind != "static" {
		m.fail(d.At, "unsupported evidence %s (kind %q); runtime proof needs a resolver", d.Evidence, kind)
		return
	}
	m.staticEvidence(root, d, location, files)
}

func (m *Model) sourcePath(d Declaration, files []string) {
	if d.Path != "" && !contains(files, d.Path) {
		m.fail(d.At, "source path %s is not a tracked file (diagram %s)", d.Path, d.Diagram)
	}
}

func (m *Model) staticEvidence(root string, d Declaration, location string, files []string) {
	file, number, ok := strings.Cut(location, ":")
	line := positiveNumber(number)
	if !ok || line == 0 || !contains(files, file) {
		m.fail(d.At, "unresolved static evidence %s", location)
		return
	}
	if !sourceLine(root, file, line) {
		m.fail(d.At, "unresolved static evidence %s", location)
	}
}

func sourceLine(root, file string, line int) bool {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	return err == nil && line <= len(strings.Split(string(body), "\n"))
}
