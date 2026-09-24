// Package mdtemplate validates a markdown document — a GitHub Issue or PR
// body — against a fixed template shape: a one-line blockquote pitch, an exact
// set of level-2 sections, and a per-section content rule. It is shared by the
// plan-structure and pr-structure checks so the pitch, required-section, and
// "no other sections" rules live in one place.
package mdtemplate

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Content is an extra requirement on a section's body.
type Content int

const (
	// Prose requires nothing beyond the heading itself.
	Prose Content = iota
	// Checklist requires at least one "- [ ]" / "1. [ ]" item.
	Checklist
	// Bullet requires at least one "-" / "*" / "+" list item.
	Bullet
)

// Section is one required level-2 heading and the content it must carry.
type Section struct {
	Heading  string
	Requires Content
}

// Spec is a template shape: the noun naming the document in messages, the
// pitch placeholder, the required sections in order, and whether an "Open
// questions" section earns a pointed message instead of the generic one.
type Spec struct {
	Noun             string
	Placeholder      string
	Sections         []Section
	BanOpenQuestions bool
}

// Error messages are constants so both checks' tests can pin the exact wording.
const (
	errNoPitch     = "Missing pitch: add a one-line blockquote (> …) before the first ## section"
	errPlaceholder = "Pitch is still the template placeholder; write the real one-line value"
	errOpenQ       = "Remove the `## Open questions` section — turn unknowns into checklist steps"
)

var (
	checkboxRe      = regexp.MustCompile(`^\s*(?:[-*+]|\d+\.)\s+\[[ xX]\]`)
	bulletRe        = regexp.MustCompile(`^\s*[-*+]\s+\S`)
	openQuestionsRe = regexp.MustCompile(`(?i)open\s+questions`)
)

// contentRule is the line matcher and the "empty section" message tail for a
// Content that requires a list item.
type contentRule struct {
	re  *regexp.Regexp
	msg string
}

// contentRules maps each list-bearing Content to its rule; Prose has no entry.
var contentRules = map[Content]contentRule{
	Checklist: {checkboxRe, "has no checklist item (- [ ] …)"},
	Bullet:    {bulletRe, "has no list item (- …)"},
}

// Validate returns one message per violation of spec, in order: the pitch,
// then each required section (missing, then its content rule), then every
// unexpected level-2 heading in document order. An empty result means the body
// matches the template.
func Validate(body string, spec Spec) []string {
	headings := mdx.Headings(body)
	var errs []string
	errs = append(errs, pitchErrors(body, headings, spec)...)
	errs = append(errs, sectionErrors(body, headings, spec)...)
	errs = append(errs, extraSectionErrors(headings, spec)...)
	return errs
}

// pitchErrors checks the leading one-line blockquote pitch.
func pitchErrors(body string, headings []mdx.Heading, spec Spec) []string {
	pitch, ok := pitchLine(body, headings)
	if !ok {
		return []string{errNoPitch}
	}
	if pitch == "" || (spec.Placeholder != "" && strings.Contains(pitch, spec.Placeholder)) {
		return []string{errPlaceholder}
	}
	return nil
}

// sectionErrors checks each required section is present and carries the
// content its rule demands.
func sectionErrors(body string, headings []mdx.Heading, spec Spec) []string {
	var errs []string
	for _, s := range spec.Sections {
		if msg := sectionError(body, headings, s); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}

// sectionError returns one section's violation message, or "" when it is
// present and satisfies its content rule.
func sectionError(body string, headings []mdx.Heading, s Section) string {
	if !hasH2(headings, s.Heading) {
		return "Missing `## " + s.Heading + "` section"
	}
	if rule, ok := contentRules[s.Requires]; ok && !sectionHasMatch(body, headings, s.Heading, rule.re) {
		return "`## " + s.Heading + "` " + rule.msg
	}
	return ""
}

// extraSectionErrors flags every level-2 heading that is not one of the spec's
// sections, with a pointed message for an "Open questions" section when the
// spec bans it.
func extraSectionErrors(headings []mdx.Heading, spec Spec) []string {
	allowed := allowedSet(spec)
	var errs []string
	for _, h := range headings {
		if msg := extraSectionError(h, allowed, spec); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}

// extraSectionError returns the message for one heading when it is an
// unexpected level-2 section, or "" when it is allowed or not a level-2
// heading.
func extraSectionError(h mdx.Heading, allowed map[string]bool, spec Spec) string {
	if h.Level != 2 || allowed[h.Text] {
		return ""
	}
	if spec.BanOpenQuestions && openQuestionsRe.MatchString(h.Text) {
		return errOpenQ
	}
	return "Unexpected `## " + h.Text + "` section — a " + spec.Noun + " has only " + joinHeadings(spec)
}

// allowedSet is the lookup of the spec's required section headings.
func allowedSet(spec Spec) map[string]bool {
	set := make(map[string]bool, len(spec.Sections))
	for _, s := range spec.Sections {
		set[s.Heading] = true
	}
	return set
}

// joinHeadings renders the section headings as "A", "A and B", or
// "A, B and C" for the extra-section message.
func joinHeadings(spec Spec) string {
	names := make([]string, len(spec.Sections))
	for i, s := range spec.Sections {
		names[i] = s.Heading
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// pitchLine returns the text of the first blockquote line (without the leading
// ">") that appears before the first level-2 heading, and whether one exists.
func pitchLine(body string, headings []mdx.Heading) (string, bool) {
	firstH2 := firstH2Line(headings)
	for _, ln := range mdx.NonFencedLines(body) {
		if firstH2 > 0 && ln.Num >= firstH2 {
			break
		}
		if after, ok := strings.CutPrefix(strings.TrimLeft(ln.Text, " \t"), ">"); ok {
			return strings.TrimSpace(after), true
		}
	}
	return "", false
}

// sectionHasMatch reports whether the section headed by heading (from its
// heading to the next level-2 heading or end of document) has a line matching
// re, ignoring fenced code.
func sectionHasMatch(body string, headings []mdx.Heading, heading string, re *regexp.Regexp) bool {
	start, end := sectionBounds(headings, heading)
	if start == 0 {
		return false
	}
	for _, ln := range mdx.NonFencedLines(body) {
		if inSection(ln.Num, start, end) && re.MatchString(ln.Text) {
			return true
		}
	}
	return false
}

func inSection(n, start, end int) bool {
	return n > start && (end == 0 || n < end)
}

// sectionBounds returns the 1-based line of the given level-2 heading and the
// line of the next level-2 heading after it (0 when it runs to end of doc).
func sectionBounds(headings []mdx.Heading, text string) (start, end int) {
	i := indexOfH2(headings, text)
	if i < 0 {
		return 0, 0
	}
	start = headings[i].Line
	for _, n := range headings[i+1:] {
		if n.Level == 2 {
			return start, n.Line
		}
	}
	return start, 0
}

func indexOfH2(headings []mdx.Heading, text string) int {
	for i, h := range headings {
		if h.Level == 2 && h.Text == text {
			return i
		}
	}
	return -1
}

func hasH2(headings []mdx.Heading, text string) bool {
	return indexOfH2(headings, text) >= 0
}

func firstH2Line(headings []mdx.Heading) int {
	for _, h := range headings {
		if h.Level == 2 {
			return h.Line
		}
	}
	return 0
}
