// Package questions finds each heading, bold line or label line that opens a
// list of open questions. The open-questions and plan-structure checks share
// this one definition.
package questions

import (
	"cmp"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Hit is one line that opens a list of open questions, with how it does.
type Hit struct {
	Line int
	Kind string
	Text string
}

// The kinds of line a Hit names.
const (
	Heading = "heading"
	Bold    = "bold line"
	Label   = "label line"
)

// name is the phrase, in any case: open questions, unresolved questions or
// questions for the owner.
const name = `(?:open|unresolved)\s+questions|questions\s+for\s+the\s+owner`

var (
	nameRe = regexp.MustCompile(`(?i)\b(?:` + name + `)\b`)
	// labelRe is a line that starts with the name, then a colon or nothing.
	// A sentence that starts with the name goes on without a colon.
	labelRe = regexp.MustCompile(`(?i)^(?:` + name + `)\s*(?::|$)`)
	atxRe   = regexp.MustCompile(`^ {0,3}#{1,6}\s+(.*)$`)
	boldRe  = regexp.MustCompile(`^(?:\*\*|__)(.+)(?:\*\*|__)\s*:?$`)
	// htmlHeadingRe is an <h1> to <h6> element, its text in group 1.
	htmlHeadingRe = regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]\s*>`)
	tagRe         = regexp.MustCompile(`<[^>]*>`)
	spanRe        = regexp.MustCompile("`[^`\n]+`")
	// htmlCodeRes are the HTML a reader never sees as prose: comments, and
	// code, script and style elements.
	htmlCodeRes = []*regexp.Regexp{
		regexp.MustCompile(`(?s)<!--.*?-->`),
		regexp.MustCompile(`(?is)<pre\b.*?</pre\s*>`),
		regexp.MustCompile(`(?is)<code\b.*?</code\s*>`),
		regexp.MustCompile(`(?is)<script\b.*?</script\s*>`),
		regexp.MustCompile(`(?is)<style\b.*?</style\s*>`),
	}
	emphasis = strings.NewReplacer("**", "", "__", "")
)

// lineRule is one way a line opens a list: its kind and its test.
type lineRule struct {
	kind  string
	match func(line string) bool
}

// labelRule is a label line, after its tags and emphasis are dropped.
var labelRule = lineRule{Label, func(line string) bool {
	return labelRe.MatchString(strings.TrimLeft(emphasis.Replace(plain(line)), "*_ "))
}}

// markdownRules are the line rules of Markdown, in the order they are tried.
var markdownRules = []lineRule{
	{Heading, func(line string) bool {
		m := atxRe.FindStringSubmatch(line)
		return m != nil && Named(m[1])
	}},
	{Bold, func(line string) bool {
		m := boldRe.FindStringSubmatch(strings.TrimSpace(line))
		return m != nil && Named(m[1])
	}},
	labelRule,
}

// Named reports whether a heading's text names a list of open questions.
func Named(text string) bool {
	return nameRe.MatchString(text)
}

// Find returns every line of content that opens a list of open questions,
// in order, judging only what a reader sees as prose.
func Find(content string, html bool) []Hit {
	content = prose(content, html)
	hits := append(htmlHeadings(content), lineHits(blankMatches(content, htmlHeadingRe), html)...)
	slices.SortStableFunc(hits, func(a, b Hit) int { return cmp.Compare(a.Line, b.Line) })
	return hits
}

// prose blanks the code in content, keeping every line. Markdown loses its
// code spans and fenced blocks; HTML keeps its backticks as prose.
func prose(content string, html bool) string {
	if !html {
		content = maskMarkdownCode(content)
	}
	for _, re := range htmlCodeRes {
		content = blankMatches(content, re)
	}
	return content
}

// htmlHeadings are the <h1> to <h6> elements in content that name the list.
func htmlHeadings(content string) []Hit {
	var hits []Hit
	for _, loc := range htmlHeadingRe.FindAllStringSubmatchIndex(content, -1) {
		if text := plain(content[loc[2]:loc[3]]); Named(text) {
			hits = append(hits, Hit{Line: lineAt(content, loc[0]), Kind: Heading, Text: text})
		}
	}
	return hits
}

// lineHits are the lines of content that open the list by a line rule.
func lineHits(content string, html bool) []Hit {
	var hits []Hit
	for i, line := range strings.Split(content, "\n") {
		if kind := lineKind(line, html); kind != "" {
			hits = append(hits, Hit{Line: i + 1, Kind: kind, Text: plain(line)})
		}
	}
	return hits
}

// lineKind is the kind of the first rule line meets, or "". An HTML page
// has label lines alone, since its headings are tags.
func lineKind(line string, html bool) string {
	rules := markdownRules
	if html {
		rules = []lineRule{labelRule}
	}
	for _, r := range rules {
		if r.match(line) {
			return r.kind
		}
	}
	return ""
}

// maskMarkdownCode blanks fenced blocks and code spans, keeping every line.
func maskMarkdownCode(content string) string {
	lines := mdx.BlankFenced(strings.Split(content, "\n"))
	for i, line := range lines {
		lines[i] = blankMatches(line, spanRe)
	}
	return strings.Join(lines, "\n")
}

// blankMatches replaces each match of re with spaces, keeping its newlines
// so line numbers hold.
func blankMatches(s string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(s, func(m string) string {
		return strings.Map(func(r rune) rune {
			if r == '\n' {
				return r
			}
			return ' '
		}, m)
	})
}

// plain is s without its tags, its spaces collapsed.
func plain(s string) string {
	return strings.Join(strings.Fields(tagRe.ReplaceAllString(s, " ")), " ")
}

// lineAt is the 1-based line of byte offset i in s.
func lineAt(s string, i int) int {
	return strings.Count(s[:i], "\n") + 1
}
