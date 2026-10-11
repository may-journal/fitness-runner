package main

import (
	"fmt"
	"strings"
	"unicode"
)

// adr0004 states the rule that one user-facing area has one requirement doc.
const adr0004 = "https://github.com/may-journal/fitness-runner/blob/main/docs/architecture/adr/0004-one-requirement-doc-per-area.md"

// minOverlap is the share of words two Givens must have in common, under the
// same When, for their docs to cover one area. It is loose on purpose: a
// reworded Given still counts.
const minOverlap = 0.3

// scenario is one acceptance's Given and When, the part that names its area.
type scenario struct {
	idLine
	given string
	when  string
}

// overlapErrors flags each pair of docs holding acceptances with the same
// When and similar Givens, since one area belongs in one doc.
func overlapErrors(docs []reqDoc) []string {
	var errs []string
	for i, a := range docs {
		for _, b := range docs[i+1:] {
			if x, y, ok := sharedScenario(a, b); ok {
				errs = append(errs, at(b.path, y.line, fmt.Sprintf("%s and %s both cover %q (%s, %s); extend one doc, see %s", a.path, b.path, "When "+y.when, x.id, y.id, adr0004)))
			}
		}
	}
	return errs
}

// sharedScenario returns the first pair of a's and b's scenarios with the
// same When and similar Givens.
func sharedScenario(a, b reqDoc) (scenario, scenario, bool) {
	for _, x := range a.scenarios {
		for _, y := range b.scenarios {
			if strings.EqualFold(x.when, y.when) && similar(x.given, y.given) {
				return x, y, true
			}
		}
	}
	return scenario{}, scenario{}, false
}

// similar reports whether the two texts share at least minOverlap of their
// distinct words.
func similar(a, b string) bool {
	wa, wb := words(a), words(b)
	shared := 0
	for w := range wa {
		if wb[w] {
			shared++
		}
	}
	union := len(wa) + len(wb) - shared
	return union > 0 && float64(shared)/float64(union) >= minOverlap
}

// fillers are words too common to say what a Given is about.
var fillers = map[string]bool{"a": true, "an": true, "the": true, "and": true, "or": true, "of": true, "in": true, "on": true, "to": true, "with": true, "that": true, "is": true, "not": true, "no": true, "its": true, "it": true, "whose": true}

// words is the set of lowercase words in s, fillers left out.
func words(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !fillers[w] {
			out[w] = true
		}
	}
	return out
}
