package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/requirement"
)

// adr0004 states the rule that one user-facing area has one requirement doc.
const adr0004 = "https://github.com/may-journal/fitness-runner/blob/main/docs/architecture/adr/0004-one-requirement-doc-per-area.md"

// minOverlap is the share of words two Givens must have in common, under the
// same When, for their docs to cover one area. It is loose on purpose, so a
// reworded Given still counts.
const minOverlap = 0.3

// minShared is the fewest words two similar Givens share. Short Givens meet
// minOverlap with one word, and one word, such as the rest of a multi-word
// actor, says too little about an area.
const minShared = 2

// overlapErrors flags each pair of docs holding acceptances with the same
// When and similar Givens, since one area belongs in one doc.
func overlapErrors(docs []requirement.Document) []string {
	var errs []string
	for i, a := range docs {
		for _, b := range docs[i+1:] {
			if x, y, ok := sharedArea(a, b); ok {
				errs = append(errs, requirement.At(b.Path, y.Line, fmt.Sprintf("%s and %s both cover %q (%s, %s); extend one doc, see %s", a.Path, b.Path, "When "+y.When, x.ID, y.ID, adr0004)))
			}
		}
	}
	return errs
}

// sharedArea returns the first pair of a's and b's acceptances with the same
// When and similar Givens.
func sharedArea(a, b requirement.Document) (requirement.IDLine, requirement.IDLine, bool) {
	for _, x := range a.Acceptances {
		for _, y := range b.Acceptances {
			if sameArea(x, y) {
				return x, y, true
			}
		}
	}
	return requirement.IDLine{}, requirement.IDLine{}, false
}

// sameArea reports whether two acceptances share a When and have similar
// Givens.
func sameArea(x, y requirement.IDLine) bool {
	return x.When != "" && strings.EqualFold(x.When, y.When) && similar(x.Given, y.Given)
}

// similar reports whether the two Givens share at least minShared words and
// minOverlap of their distinct words. The actor that opens every Given says
// nothing about the area, so its first word is left out.
func similar(a, b string) bool {
	wa, wb := words(withoutActor(a)), words(withoutActor(b))
	shared := 0
	for w := range wa {
		if wb[w] {
			shared++
		}
	}
	union := len(wa) + len(wb) - shared
	return shared >= minShared && float64(shared)/float64(union) >= minOverlap
}

// withoutActor drops a Given's first word, the actor it starts with.
func withoutActor(given string) string {
	_, rest, _ := strings.Cut(strings.TrimSpace(given), " ")
	return rest
}

// fillers are words too common to say what a Given is about.
var fillers = map[string]bool{"a": true, "an": true, "the": true, "and": true, "or": true, "of": true, "in": true, "on": true, "to": true, "with": true, "that": true, "is": true, "not": true, "no": true, "its": true, "it": true, "whose": true, "has": true, "have": true, "my": true}

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
