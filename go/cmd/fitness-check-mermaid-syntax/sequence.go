// cspell:ignore Descr autonumber

package check

import (
	"fmt"
	"regexp"
	"strings"
)

// blockOpeners are the sequence statements that open a block closed by end.
var blockOpeners = map[string]bool{
	"loop": true, "alt": true, "opt": true, "par": true, "critical": true,
	"break": true, "rect": true, "box": true, "par_over": true,
}

// blockBranches are the statements that split a block, by the block each
// belongs in.
var blockBranches = map[string]string{"else": "alt", "and": "par", "option": "critical"}

// blockAlias is the block a branch sees an opener as; par_over takes and
// like par.
var blockAlias = map[string]string{"par_over": "par"}

// sequenceArgs are the sequence statements that take operands, and the
// fields each needs.
var sequenceArgs = map[string]int{
	"participant": 2, "actor": 2, "create": 3, "destroy": 2, "activate": 2, "deactivate": 2,
	"autonumber": 1, "title": 1, "accTitle:": 1, "accDescr:": 1, "link": 2, "links": 2,
	"properties": 2, "details": 2,
}

// note is a note statement: placement, actors, colon, text.
var note = regexp.MustCompile(`(?i)^note\s+(?:left of|right of|over)\s+[^:]+:`)

// arrow is a sequence message arrow.
var arrow = regexp.MustCompile(`<<--?>>|--?>>|--?>|--?[x)]`)

// sequence parses a sequenceDiagram's statements.
func sequence(_ srcLine, lines []srcLine) []problem {
	s := &seq{}
	for _, l := range lines {
		if msg := s.statement(l.n, l.text); msg != "" {
			s.problems = append(s.problems, problem{l.n, msg})
		}
	}
	for _, b := range s.open {
		s.problems = append(s.problems, problem{b.line, b.kind + " is never closed with end"})
	}
	return s.problems
}

// block is an open sequence block.
type block struct {
	kind string
	line int
}

// seq is a sequence diagram parse in progress.
type seq struct {
	open     []block
	problems []problem
}

// statement parses one line and returns its error, or "".
func (s *seq) statement(n int, text string) string {
	word := strings.Fields(text)[0]
	switch {
	case blockOpeners[word]:
		s.open = append(s.open, block{word, n})
		return ""
	case word == "end":
		return s.end()
	case blockBranches[word] != "":
		return s.branch(word)
	}
	return s.simple(text, word)
}

// end closes the innermost block.
func (s *seq) end() string {
	if len(s.open) == 0 {
		return "end closes no block"
	}
	s.open = s.open[:len(s.open)-1]
	return ""
}

// branch checks that else, and, or option sits in its own block.
func (s *seq) branch(word string) string {
	want := blockBranches[word]
	if len(s.open) == 0 || seen(s.open[len(s.open)-1].kind) != want {
		return fmt.Sprintf("%s belongs inside %s", word, want)
	}
	return ""
}

// seen is the block kind a branch statement sees for an opener.
func seen(kind string) string {
	if alias, ok := blockAlias[kind]; ok {
		return alias
	}
	return kind
}

// simple parses a keyword statement, a note, or a message.
func (s *seq) simple(text, word string) string {
	if need, ok := sequenceArgs[word]; ok {
		return needFields(text, word, need)
	}
	if strings.EqualFold(word, "note") {
		return matchOr(note, text, "a note needs left of, right of, or over, an actor, and a colon")
	}
	return message(text)
}

// needFields fails a keyword statement with too few parts.
func needFields(text, word string, need int) string {
	if len(strings.Fields(text)) < need {
		return fmt.Sprintf("%s needs %d parts", word, need)
	}
	return ""
}

// matchOr is msg when re does not match text.
func matchOr(re *regexp.Regexp, text, msg string) string {
	if re.MatchString(text) {
		return ""
	}
	return msg
}

// message parses `From->>To: text`, with an optional + or - activation.
func message(text string) string {
	loc := arrow.FindStringIndex(text)
	if loc == nil || strings.TrimSpace(text[:loc[0]]) == "" {
		return fmt.Sprintf("%q is not a sequence statement or a message such as A->>B: text", text)
	}
	to, _, colon := strings.Cut(strings.TrimLeft(text[loc[1]:], "+- "), ":")
	if !colon || strings.TrimSpace(to) == "" {
		return fmt.Sprintf("message %q needs a target and a colon, as in A->>B: text", text)
	}
	return ""
}
