// cspell:ignore Jaccard

// Package dupe scores how alike two issues read: the share of words their
// titles and bodies have in common, once markdown, code, links and filler
// words are gone. `fitness dupe-check` comments when a new issue scores high
// against an older one.
package dupe

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Words is the set of words an issue's title and body share with others.
type Words map[string]bool

var (
	// fenceRe is a fenced code block, which holds code, not prose.
	fenceRe = regexp.MustCompile("(?s)```.*?(```|$)")
	// codeRe is inline code.
	codeRe = regexp.MustCompile("`[^`\n]*`")
	// commentRe is an HTML comment, such as a template hint.
	commentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	// headingRe is a heading line; issue templates give every issue of a
	// kind the same ones.
	headingRe = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s.*$`)
	// linkRe keeps a markdown link's text and drops its target.
	linkRe = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	// urlRe is a bare link.
	urlRe = regexp.MustCompile(`https?://\S+`)
)

// minLength is the fewest letters a word that counts has.
const minLength = 3

// filler are common words that say nothing about what an issue is about.
var filler = setOf(`the and for with that this from into onto when then than
		what which who whom whose why how where while are was were been being have has had
		does did done doing not but its our ours your yours their theirs they them there here
		these those each every all any some one two can could should would will shall may might
		must also only just more most less very much many such own same other others out off
		about above below after before again once over under both few nor too via per yet
		new add adds added use uses used make makes made get gets got need needs needed
		like now issue issues plan plans work happen happens`)

// setOf is the set of the words in list.
func setOf(list string) Words {
	set := Words{}
	for _, w := range strings.Fields(list) {
		set[w] = true
	}
	return set
}

// Of is the word set of an issue with title and body.
func Of(title, body string) Words {
	text := title + "\n" + plain(body)
	words := Words{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), notWordRune) {
		if counts(w) {
			words[w] = true
		}
	}
	return words
}

// plain is body with code, comments, headings and link targets dropped.
func plain(body string) string {
	for _, re := range []*regexp.Regexp{fenceRe, codeRe, commentRe, headingRe, urlRe} {
		body = re.ReplaceAllString(body, " ")
	}
	return linkRe.ReplaceAllString(body, "$1")
}

// notWordRune splits text on anything but letters and digits.
func notWordRune(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// counts reports whether w is long enough, not filler, and not a bare
// number such as an issue reference.
func counts(w string) bool {
	return utf8.RuneCountInString(w) >= minLength && !filler[w] && strings.TrimFunc(w, unicode.IsDigit) != ""
}

// Score is the Jaccard index of a and b: the words they share over the words
// either has, from 0 for none to 1 for the same set.
func Score(a, b Words) float64 {
	shared := 0
	for w := range a {
		if b[w] {
			shared++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}
