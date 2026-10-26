package architecture

import "strings"

// addDiagram checks a finished diagram and adds it to the model.
func (m *Model) addDiagram(d *diagram) {
	d.annotated()
	for _, r := range d.edges {
		d.endpoint(r.From, r.At)
		d.endpoint(r.To, r.At)
		m.Relationships = append(m.Relationships, *r)
	}
	for _, e := range d.order {
		d.enclosed(e)
		m.Elements = append(m.Elements, *e)
	}
	d.overlaps()
	m.Claims = append(m.Claims, d.claims...)
	m.Mentions = append(m.Mentions, d.mentions...)
}

func (d *diagram) endpoint(id string, at Location) {
	if d.elements[id] == nil {
		d.m.fail(at, "unresolved ID %s; declare it in this diagram", id)
	}
}

// enclosed requires a boundary around each container of a container view
// and each component of a component view.
func (d *diagram) enclosed(e *Element) {
	level := map[string]string{"C4Container": "Container", "C4Component": "Component"}[d.form]
	if e.Parent == "" && level != "" && internal(e.Kind, level) {
		d.m.fail(e.At, "%s %s needs an enclosing boundary", strings.ToLower(level), e.ID)
	}
}

// internal reports a view's own unit: not external and not a boundary.
func internal(kind, level string) bool {
	return strings.HasPrefix(kind, level) && !strings.HasSuffix(kind, "_Ext") && !strings.HasSuffix(kind, "Boundary")
}

// overlaps fails two elements that claim the same source when neither
// encloses the other.
func (d *diagram) overlaps() {
	for i, a := range d.claims {
		for _, b := range d.claims[i+1:] {
			d.overlap(a, b)
		}
	}
}

func (d *diagram) overlap(a, b Claim) {
	if d.nested(a.Owner, b.Owner) {
		return
	}
	if covers(a.Path, b.Path) || covers(b.Path, a.Path) {
		d.m.fail(b.At, "overlapping source claims: %s claims %s and %s claims %s at %s", b.Owner, b.Path, a.Owner, a.Path, a.At)
	}
}

func (d *diagram) nested(a, b string) bool {
	return a == b || d.encloses(a, b) || d.encloses(b, a)
}

func (d *diagram) encloses(outer, inner string) bool {
	for e := d.elements[inner]; e != nil && e.Parent != ""; e = d.elements[e.Parent] {
		if e.Parent == outer {
			return true
		}
	}
	return false
}

// actors keeps each person's ID and name one-to-one across all views, so
// later checks can match requirement Given text to one actor.
func (m *Model) actors() {
	names, ids := map[string]Element{}, map[string]Element{}
	for _, e := range m.Elements {
		if person(e.Kind) {
			m.actor(e, names, ids)
		}
	}
}

func (m *Model) actor(e Element, names, ids map[string]Element) {
	if first, seen := ids[e.ID]; seen && first.Name != e.Name {
		m.fail(e.At, "actor %s is named %q here and %q at %s", e.ID, e.Name, first.Name, first.At)
	}
	if first, seen := names[e.Name]; seen && first.ID != e.ID {
		m.fail(e.At, "actor name %q is ambiguous: %s here and %s at %s", e.Name, e.ID, first.ID, first.At)
	}
	keepFirst(ids, e.ID, e)
	keepFirst(names, e.Name, e)
}

func keepFirst(seen map[string]Element, key string, e Element) {
	if _, ok := seen[key]; !ok {
		seen[key] = e
	}
}
