package architecture

import (
	"path"
	"regexp"
	"strings"
)

var word = regexp.MustCompile(`[a-z0-9][a-z0-9.+-]*`)

// prove checks each evidence link an edge row of the components view
// makes: the file belongs to the edge's source, and it names the target.
// An edge row without a link allows the edge; a linked one requires it.
func (c *components) prove() {
	for _, ev := range c.m.Evidence {
		if from, to, ok := c.edgeSubject(ev.Subject); ok {
			c.proveEdge(ev, from, to)
		}
	}
}

// edgeSubject resolves an evidence subject to an edge the components
// view draws.
func (c *components) edgeSubject(subject string) (string, string, bool) {
	from, to, ok := strings.Cut(subject, "->")
	return from, to, ok && c.draws(from, to)
}

// draws reports an edge the components view draws exactly, from from to to.
func (c *components) draws(from, to string) bool {
	for _, r := range c.m.Relationships {
		if r.View == componentsView && r.From == from && r.To == to {
			return true
		}
	}
	return false
}

func (c *components) proveEdge(ev Evidence, from, to string) {
	owner := c.owners[ev.Path]
	switch {
	case !supportedEvidence(ev.Path):
		c.fail(ev.At, "unsupported evidence %s; Go and Swift sources only", ev)
	case !c.within(owner, from):
		c.fail(ev.At, "evidence %s is owned by %s, not %s", ev, ownerName(owner), from)
	case external(c.elements[to].Kind):
		c.proveExternal(ev, c.elements[to])
	case !c.names(ev, to):
		c.fail(ev.At, "evidence %s names nothing %s owns", ev, to)
	}
}

func supportedEvidence(p string) bool {
	return contains([]string{".go", ".swift"}, path.Ext(p))
}

func ownerName(owner string) string {
	if owner == "" {
		return "no component"
	}
	return owner
}

// names reports a reference in the linked lines that reaches a source to
// owns or encloses.
func (c *components) names(ev Evidence, to string) bool {
	for _, r := range c.refs {
		if r.At.Path == ev.Path && ev.holds(r.At.Line) && c.reaches(r, to) {
			return true
		}
	}
	return false
}

func (c *components) reaches(r reference, to string) bool {
	for _, owner := range c.pathOwners(r.To) {
		if c.within(owner, to) {
			return true
		}
	}
	return false
}

// proveExternal requires the linked lines to name the external system,
// by a word of its name, technology, or description, outside comments.
func (c *components) proveExternal(ev Evidence, e Element) {
	words := longWords(e.Name + " " + e.Technology + " " + e.Description)
	sc := &scanner{}
	for i, line := range c.m.repo.read(ev.Path) {
		code := sc.line(line)
		if ev.holds(i+1) && namesAny(code, words) {
			return
		}
	}
	c.fail(ev.At, "evidence %s does not name %s outside comments", ev, e.Name)
}

// longWords lists the words of three or more characters in text, so an
// article in a description names nothing.
func longWords(text string) []string {
	var words []string
	for _, token := range word.FindAllString(strings.ToLower(text), -1) {
		if len(token) >= 3 {
			words = append(words, token)
		}
	}
	return words
}

func namesAny(code string, words []string) bool {
	for _, token := range word.FindAllString(strings.ToLower(code), -1) {
		if contains(words, token) {
			return true
		}
	}
	return false
}
