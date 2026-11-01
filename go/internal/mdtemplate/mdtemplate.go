// Package mdtemplate validates an Issue or PR body against a template shape,
// shared by plan-structure and pr-structure.
package mdtemplate

import (
	"os"
	"path/filepath"
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

// Spec is a template shape: the document noun, pitch placeholder, ordered
// required sections, and whether "Open questions" earns a pointed message.
type Spec struct {
	Noun             string
	Placeholder      string
	Sections         []Section
	BanOpenQuestions bool
	// NoPitch skips the leading-blockquote pitch rule, for a document that
	// opens straight into sections — a file like an ADR, not an Issue body.
	NoPitch bool
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

// commentRe matches the opening of an HTML comment, the guidance a template
// section carries to tell a person or bot what belongs there.
var commentRe = regexp.MustCompile(`<!--`)

// SpecFromTemplate derives a Spec from a template's level-2 headings, requiring
// a pitch only when the template opens with a blockquote.
func SpecFromTemplate(body, noun string) Spec {
	headings := mdx.Headings(body)
	var sections []Section
	for _, h := range headings {
		if h.Level == 2 {
			sections = append(sections, Section{Heading: h.Text, Requires: Prose})
		}
	}
	_, hasPitch := pitchLine(body, headings)
	return Spec{Noun: noun, Sections: sections, NoPitch: !hasPitch}
}

// TemplateLabels returns the labels from a GitHub issue template's `labels`
// front matter, as one name, comma list, flow list or block list.
func TemplateLabels(body string) []string {
	f := parseFront(body)
	if labels := f.lists["labels"]; len(labels) > 0 {
		return labels
	}
	return scalarLabels(f.scalars["labels"])
}

// IssueTemplateDir holds a repo's GitHub issue templates; each one names an
// issue kind by its first label.
const IssueTemplateDir = ".github/ISSUE_TEMPLATE"

// IssueTemplates reads root's issue templates, keyed by each one's first
// label. A template with no label names no kind.
func IssueTemplates(root string) map[string]string {
	out := map[string]string{}
	paths, _ := filepath.Glob(filepath.Join(root, IssueTemplateDir, "*.md"))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if labels := TemplateLabels(string(raw)); err == nil && len(labels) > 0 {
			out[labels[0]] = string(raw)
		}
	}
	return out
}

// KindSpec is SpecFromTemplate with a REPLACE-ME placeholder, no Open questions
// section, and a checklist wherever the template shows a task-list item.
func KindSpec(template, noun string) Spec {
	spec := SpecFromTemplate(template, noun)
	spec.Placeholder, spec.BanOpenQuestions = "REPLACE-ME", true
	headings := mdx.Headings(template)
	for i, s := range spec.Sections {
		if sectionHasMatch(template, headings, s.Heading, checkboxRe) {
			spec.Sections[i].Requires = Checklist
		}
	}
	return spec
}

// SpecFor is the shape from root's issue template labeled label, else
// fallback, keeping the template the one source.
func SpecFor(root, label string, fallback Spec) Spec {
	if template, ok := IssueTemplates(root)[label]; ok {
		return KindSpec(template, fallback.Noun)
	}
	return fallback
}

// scalarLabels splits a `labels` scalar, one name or a comma-separated list.
func scalarLabels(value string) []string {
	var labels []string
	for _, part := range strings.Split(value, ",") {
		if l := unquote(part); l != "" {
			labels = append(labels, l)
		}
	}
	return labels
}

// MissingCommentSections returns the level-2 headings whose section carries no
// HTML comment, so every section is guided.
func MissingCommentSections(body string) []string {
	headings := mdx.Headings(body)
	var out []string
	for _, h := range headings {
		if h.Level == 2 && !sectionHasMatch(body, headings, h.Text, commentRe) {
			out = append(out, h.Text)
		}
	}
	return out
}

// Validate returns one message per violation of spec: the pitch, each required
// section, then unexpected level-2 headings; empty means the body matches.
func Validate(body string, spec Spec) []string {
	headings := mdx.Headings(body)
	var errs []string
	if !spec.NoPitch {
		errs = append(errs, pitchErrors(body, headings, spec)...)
	}
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

// extraSectionErrors flags each level-2 heading outside the spec's sections,
// pointedly for a banned "Open questions" section.
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

// extraSectionError returns the message for an unexpected level-2 heading, or ""
// when it is allowed or not level 2.
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

// sectionHasMatch reports whether the section under heading has a line
// matching re, ignoring fenced code.
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

// sectionBounds returns the 1-based line of a level-2 heading and of the next
// one, which is 0 at the document's end.
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
