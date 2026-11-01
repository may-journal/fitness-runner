package architecture

import (
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/requirement"
)

// Actor is a person declared in the system context. Name is its label
// without the callout number; ID is its Mermaid ID, the one alias the
// diagram already declares. System and legend nodes are never actors.
type Actor struct {
	ID, Name string
	At       Location
}

// person reports a C4 Person macro, the one form a person takes. A
// flowchart node classed person is no person, since the system context is
// a C4Context diagram.
func person(kind string) bool {
	return strings.HasPrefix(kind, "Person")
}

// Actors lists the context view's C4 people in diagram order, once per ID
// across both roots.
func (m *Model) Actors() []Actor {
	var actors []Actor
	seen := map[string]bool{}
	for _, e := range m.Elements {
		if e.View == "context" && person(e.Kind) && !seen[e.ID] {
			seen[e.ID] = true
			actors = append(actors, Actor{ID: e.ID, Name: e.Name, At: e.At})
		}
	}
	return actors
}

// roster resolves the words a Given may start with: each actor's full name
// and its ID. Longer spellings are tried first, so "AI agent" wins over "AI".
type roster struct {
	byWord map[string][]Actor
	words  []string
	at     Location
	names  string
}

func (m *Model) roster() roster {
	r := roster{byWord: map[string][]Actor{}, at: m.contextLocation()}
	var names []string
	for _, a := range m.Actors() {
		r.byWord[a.Name] = append(r.byWord[a.Name], a)
		if a.ID != a.Name {
			r.byWord[a.ID] = append(r.byWord[a.ID], a)
		}
		names = append(names, a.Name)
	}
	r.words = sortedKeys(r.byWord)
	sort.SliceStable(r.words, func(i, j int) bool { return len(r.words[i]) > len(r.words[j]) })
	r.names = strings.Join(names, ", ")
	return r
}

// CheckGivens requires every acceptance's Given to start with declared actors
// joined by "and", naming the Given and context lines on each failure.
func (m *Model) CheckGivens() []string {
	r := m.roster()
	var errs []string
	for _, doc := range m.Requirements {
		for _, a := range doc.Acceptances {
			errs = append(errs, r.given(doc, a)...)
		}
	}
	return errs
}

// clause is one Given while its actors are read: the full text for
// messages, and the rest still to read.
type clause struct {
	full, rest string
	at         Location
}

// given checks one acceptance. An acceptance without a Given line is a doc
// shape error the requirements check owns.
func (r roster) given(doc requirement.Document, a requirement.IDLine) []string {
	if a.GivenLine == 0 {
		return nil
	}
	c := clause{full: a.Given, rest: a.Given, at: Location{doc.Path, a.GivenLine}}
	more := true
	for more {
		if errs := r.actor(&c); errs != nil {
			return errs
		}
		c.rest, more = cutWord(c.rest, "and")
	}
	return nil
}

// actor consumes one actor spelling at the start of the clause's rest.
func (r roster) actor(c *clause) []string {
	for _, word := range r.words {
		if rest, ok := cutWord(c.rest, word); ok {
			c.rest = rest
			return r.resolved(word, c.at)
		}
	}
	return []string{c.at.String() + ": Given " + quote(c.full) + c.where() + "; " + r.declared()}
}

// cutWord removes word from the start of text when it ends the text or
// precedes a space, returning the rest.
func cutWord(text, word string) (string, bool) {
	rest, ok := strings.CutPrefix(text, word)
	if !ok || (rest != "" && !strings.HasPrefix(rest, " ")) {
		return text, false
	}
	return strings.TrimPrefix(rest, " "), true
}

// where says which part of the Given names no actor.
func (c clause) where() string {
	if c.rest == c.full {
		return " starts with no actor"
	}
	return " names no actor at " + quote(c.rest)
}

func (r roster) declared() string {
	if r.names == "" {
		return "no actor is declared at " + r.at.String()
	}
	return "actors declared at " + r.at.String() + " are " + r.names
}

// resolved fails a spelling that names two actors, such as one actor's ID
// that is another's name.
func (r roster) resolved(word string, at Location) []string {
	actors := r.byWord[word]
	if len(actors) < 2 {
		return nil
	}
	var places []string
	for _, a := range actors {
		places = append(places, a.ID+" at "+a.At.String())
	}
	return []string{at.String() + ": Given actor " + quote(word) + " is ambiguous: " + strings.Join(places, " and ")}
}

func quote(text string) string {
	return `"` + text + `"`
}
