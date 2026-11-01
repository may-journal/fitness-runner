package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Rule IDs of the two rules only code comments meet.
const (
	ruleFragment     = "comment-fragment"
	ruleOneStatement = "comment-one-statement"
)

// maxCommentSentences caps the sentences of every comment but a package clause's.
const maxCommentSentences = 2

// minSentenceWords is the fewest words a comment sentence may hold.
const minSentenceWords = 4

// connectors open a sentence that only carries on the one before it.
var connectors = map[string]bool{
	"so": true, "and": true, "but": true, "then": true, "also": true,
	"thus": true, "hence": true, "because": true, "which": true, "plus": true,
}

// docHeading opens a Go doc heading or note, such as `Deprecated:`.
var docHeading = regexp.MustCompile(`^(Deprecated|BUG\(\w+\)):`)

// identifier is a dotted, camelCase, PascalCase or snake_case name.
var identifier = regexp.MustCompile(`^(\w+(\.\w+)+|\w*[a-z][A-Z]\w*|\w+_\w+)$`)

// quoted is a one-line double-quoted literal, whose stops end no sentence.
var quoted = regexp.MustCompile(`"[^"\n]*"`)

// codeName is a code span holding one name, such as `New`.
var codeName = regexp.MustCompile("`([\\w.]+)`")

// statements splits a comment paragraph into sentences, unmasking code names and dropping literals and lone stops.
func statements(text string) []string {
	text = quoted.ReplaceAllString(codeName.ReplaceAllString(text, "$1"), "quote")
	var words []string
	for _, w := range strings.Fields(text) {
		if strings.Trim(w, ".!?") != "" {
			words = append(words, w)
		}
	}
	return mdx.Sentences(mdx.MaskInline(strings.Join(words, " ")))
}

// commentErrors judges a comment by the paragraph and list limits, then the fragment and one-statement rules.
func commentErrors(b comments.Block, name, markdown string, lim limits) []string {
	blocks := parse(markdown)
	errs := append(paragraphErrors(name, blocks, lim), listErrors(name, blocks, lim)...)
	paragraphs, items := split(blocks)
	errs = append(errs, fragmentErrors(name, b, paragraphs)...)
	errs = append(errs, statementErrors(name, b, paragraphs, items)...)
	return append(errs, slopErrors(name, b, blocks)...)
}

// split returns each paragraph's sentences and the count of list items.
func split(blocks []block) (paragraphs [][]string, items int) {
	for _, bl := range blocks {
		if bl.isList {
			items += len(bl.items)
		} else {
			paragraphs = append(paragraphs, statements(bl.text))
		}
	}
	return paragraphs, items
}

// fragmentErrors reports each paragraph sentence that is a fragment.
func fragmentErrors(name string, b comments.Block, paragraphs [][]string) []string {
	var errs []string
	for _, sentences := range paragraphs {
		for i, s := range sentences {
			if why := fragment(b, s, i == len(sentences)-1); why != "" {
				errs = append(errs, fmt.Sprintf("%s: %s: a sentence %s; join it to the sentence before it or drop it: %q",
					name, ruleFragment, why, snippet(s)))
			}
		}
	}
	return errs
}

// fragment says why sentence s is a fragment, or returns "" for a whole sentence.
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

// shortWhole reports a doc heading, an identifier-led doc line, or a paragraph's closing colon lead-in.
func shortWhole(b comments.Block, s, first string, last bool) bool {
	return docHeading.MatchString(s) || last && strings.HasSuffix(s, ":") || identifierLed(b, first)
}

// identifierLed reports a doc sentence led by an identifier or a name its declaration holds.
func identifierLed(b comments.Block, first string) bool {
	if b.InBody {
		return false
	}
	return identifier.MatchString(first) || holdsWord(b.Next, first)
}

// holdsWord reports word as a whole word of a code line.
func holdsWord(code, word string) bool {
	for _, w := range strings.FieldsFunc(code, func(r rune) bool { return !mdx.IsAlnum(r) && r != '_' }) {
		if w == word {
			return true
		}
	}
	return false
}

// packageClause reports a Go package comment, which keeps the paragraph limits alone.
func packageClause(b comments.Block) bool {
	return b.Lang == "go" && strings.HasPrefix(b.Next, "package ")
}

// statementErrors reports a comment holding more than two sentences, counting each list item as one.
func statementErrors(name string, b comments.Block, paragraphs [][]string, items int) []string {
	if packageClause(b) {
		return nil
	}
	n := items
	for _, sentences := range paragraphs {
		n += len(sentences)
	}
	if n <= maxCommentSentences {
		return nil
	}
	return []string{fmt.Sprintf("%s: %s: a comment has %d sentences (max %d); condense it to one statement: %q",
		name, ruleOneStatement, n, maxCommentSentences, snippet(b.Text))}
}
