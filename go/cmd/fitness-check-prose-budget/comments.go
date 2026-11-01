package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// The rule IDs of the two rules that only code comments meet.
const (
	ruleFragment     = "comment-fragment"
	ruleOneStatement = "comment-one-statement"
)

// maxBodySentences is the most sentences a comment inside a function body
// or after code may hold.
const maxBodySentences = 2

// minSentenceWords is the fewest words a sentence of a comment may hold.
const minSentenceWords = 4

// connectors open a sentence that only carries on the sentence before it.
var connectors = map[string]bool{
	"so": true, "and": true, "but": true, "then": true, "also": true,
	"thus": true, "hence": true, "because": true, "which": true, "plus": true,
}

// docHeading opens a sentence that Go doc reads as a heading or a note,
// such as `Deprecated:` or `BUG(name):`.
var docHeading = regexp.MustCompile(`^(Deprecated|BUG\(\w+\)):`)

// identifier is a word written as code: a dotted name, a camelCase or
// PascalCase name with a hump, or a snake_case name.
var identifier = regexp.MustCompile(`^(\w+(\.\w+)+|\w*[a-z][A-Z]\w*|\w+_\w+)$`)

// commentErrors judges one comment block by the paragraph and list limits.
// A comment has no headings, so it skips the section limits. It then meets
// the fragment and one-statement rules, which Markdown files never meet.
func commentErrors(b comments.Block, name, markdown string, lim limits) []string {
	blocks := parse(markdown)
	errs := append(paragraphErrors(name, blocks, lim), listErrors(name, blocks, lim)...)
	errs = append(errs, fragmentErrors(name, b, blocks)...)
	return append(errs, statementErrors(name, b, blocks)...)
}

// fragmentErrors reports each sentence of a comment's paragraphs that is a
// fragment, which belongs in the sentence before it.
func fragmentErrors(name string, b comments.Block, blocks []block) []string {
	var errs []string
	for _, bl := range blocks {
		if bl.isList {
			continue
		}
		sentences := mdx.Sentences(mdx.MaskInline(bl.text))
		for i, s := range sentences {
			if why := fragment(b, s, i == len(sentences)-1); why != "" {
				errs = append(errs, fmt.Sprintf("%s: %s: a sentence %s; join it to the sentence before it or drop it: %q",
					name, ruleFragment, why, snippet(s)))
			}
		}
	}
	return errs
}

// fragment says why sentence s of comment b is a fragment, or "" when it is
// a whole sentence. last marks the last sentence of its paragraph.
func fragment(b comments.Block, s string, last bool) string {
	first := strings.Trim(strings.Fields(s)[0], ",;:")
	if connectors[strings.ToLower(first)] {
		return fmt.Sprintf("starts with %q", first)
	}
	if n := mdx.WordCount(s); n < minSentenceWords && !shortWhole(b, s, first, last) {
		return fmt.Sprintf("has %d words (min %d)", n, minSentenceWords)
	}
	return ""
}

// shortWhole reports a short sentence that still stands whole: a Go doc
// heading, or a doc line led by an identifier. A paragraph's last line that
// ends in a colon stands whole too, since it leads into code or a list.
func shortWhole(b comments.Block, s, first string, last bool) bool {
	return docHeading.MatchString(s) || last && strings.HasSuffix(s, ":") || identifierLed(b, first)
}

// identifierLed reports a doc comment's sentence whose first word is code:
// an identifier, or a name the declaration after the comment holds.
func identifierLed(b comments.Block, first string) bool {
	if b.InBody {
		return false
	}
	return identifier.MatchString(first) || holdsWord(b.Next, first)
}

// holdsWord reports word as one whole word of a line of code.
func holdsWord(code, word string) bool {
	for _, w := range strings.FieldsFunc(code, func(r rune) bool { return !mdx.IsAlnum(r) && r != '_' }) {
		if w == word {
			return true
		}
	}
	return false
}

// statementErrors reports a comment inside a function body or after code
// that holds more sentences than one statement needs. Each list item counts
// as one sentence.
func statementErrors(name string, b comments.Block, blocks []block) []string {
	if !b.InBody {
		return nil
	}
	n := 0
	for _, bl := range blocks {
		if bl.isList {
			n += len(bl.items)
		} else {
			n += len(mdx.Sentences(mdx.MaskInline(bl.text)))
		}
	}
	if n <= maxBodySentences {
		return nil
	}
	return []string{fmt.Sprintf("%s: %s: a comment in a function body or after code has %d sentences (max %d); condense it to one statement: %q",
		name, ruleOneStatement, n, maxBodySentences, snippet(b.Text))}
}
