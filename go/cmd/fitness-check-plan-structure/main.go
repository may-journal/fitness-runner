// Command fitness-check-plan-structure validates that a plan — a GitHub Issue
// body under the `Plan` label — follows the Plan issue template: a one-line
// blockquote pitch, a `## Background` section, and a `## What needs to happen`
// section with at least one checklist item, and no other `##` sections
// (double-especially no "Open questions").
//
// Unlike the file-walking checks, this one reads its target from stdin (or a
// `--body-file` path, `-` meaning stdin), falling back to the runner's
// context-inline `--body` value (FITNESS_CTX_MESSAGE). With no input at all it
// passes with zero files checked, so it is inert if ever run by the file
// runner.
package main

import (
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "plan-structure", ContextInlineArg: "--body"},
		Run:      run,
	})
}

func run(_ string, args []string) (checkkit.Result, error) {
	body, ok, err := resolveBody(args)
	if err != nil {
		return checkkit.Result{}, err
	}
	if !ok {
		return checkkit.Pass(0), nil
	}
	return judge(body), nil
}

// resolveBody returns the plan text and whether any input was supplied, in
// priority order: --body-file (- means stdin), the context-inline body
// (FITNESS_CTX_MESSAGE), then piped stdin. A terminal stdin is treated as no
// input so the check stays non-blocking under the file runner and in tests.
func resolveBody(args []string) (string, bool, error) {
	if path, ok := bodyFileArg(args); ok {
		if path == "-" {
			return readAll(os.Stdin)
		}
		raw, err := os.ReadFile(path)
		return string(raw), true, err
	}
	if msg, ok := checkkit.CtxMessage(); ok {
		return msg, true, nil
	}
	if stdinPiped() {
		return readAll(os.Stdin)
	}
	return "", false, nil
}

func readAll(r io.Reader) (string, bool, error) {
	raw, err := io.ReadAll(r)
	return string(raw), true, err
}

// bodyFileArg extracts a --body-file value in either form.
func bodyFileArg(args []string) (string, bool) {
	for i, a := range args {
		if a == "--body-file" && i+1 < len(args) {
			return args[i+1], true
		}
		if v, ok := strings.CutPrefix(a, "--body-file="); ok {
			return v, true
		}
	}
	return "", false
}

// stdinPiped reports whether stdin is a pipe or file rather than a terminal.
func stdinPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

const (
	bgHeading   = "Background"
	wnhHeading  = "What needs to happen"
	placeholder = "REPLACE-ME"
)

// Error messages, kept as constants so tests pin the exact wording.
const (
	errNoPitch      = "Missing pitch: add a one-line blockquote (> …) before the first ## section"
	errPlaceholder  = "Pitch is still the template placeholder; write the real one-line value"
	errNoBackground = "Missing `## Background` section"
	errNoWNH        = "Missing `## What needs to happen` section"
	errNoChecklist  = "`## What needs to happen` has no checklist item (- [ ] …)"
	errOpenQ        = "Remove the `## Open questions` section — turn unknowns into checklist steps"
)

var checkboxRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+\.)\s+\[[ xX]\]`)

// judge validates one plan body against the Plan template shape and returns a
// Result whose Errors list one violation each; filesChecked is always 1.
func judge(body string) checkkit.Result {
	headings := mdx.Headings(body)
	var errs []string
	errs = append(errs, pitchErrors(body, headings)...)
	errs = append(errs, sectionErrors(body, headings)...)
	errs = append(errs, extraSectionErrors(headings)...)
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}

// pitchErrors checks the leading one-line blockquote pitch.
func pitchErrors(body string, headings []mdx.Heading) []string {
	pitch, ok := pitchLine(body, headings)
	if !ok {
		return []string{errNoPitch}
	}
	if pitch == "" || strings.Contains(pitch, placeholder) {
		return []string{errPlaceholder}
	}
	return nil
}

// sectionErrors checks the two required sections and the checklist under
// "What needs to happen".
func sectionErrors(body string, headings []mdx.Heading) []string {
	var errs []string
	if !hasH2(headings, bgHeading) {
		errs = append(errs, errNoBackground)
	}
	if !hasH2(headings, wnhHeading) {
		errs = append(errs, errNoWNH)
	} else if !sectionHasChecklist(body, headings) {
		errs = append(errs, errNoChecklist)
	}
	return errs
}

// extraSectionErrors flags every level-2 heading that is not one of the two
// allowed sections, with a pointed message for an "Open questions" section.
func extraSectionErrors(headings []mdx.Heading) []string {
	var errs []string
	for _, h := range headings {
		if !isExtraH2(h) {
			continue
		}
		if isOpenQuestions(h.Text) {
			errs = append(errs, errOpenQ)
			continue
		}
		errs = append(errs, "Unexpected `## "+h.Text+"` section — a plan has only Background and What needs to happen")
	}
	return errs
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

// sectionHasChecklist reports whether the "What needs to happen" section (from
// its heading to the next level-2 heading or end of document) has at least one
// checklist item, ignoring fenced code.
func sectionHasChecklist(body string, headings []mdx.Heading) bool {
	start, end := sectionBounds(headings, wnhHeading)
	if start == 0 {
		return false
	}
	for _, ln := range mdx.NonFencedLines(body) {
		if !inSection(ln.Num, start, end) {
			continue
		}
		if checkboxRe.MatchString(ln.Text) {
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

// isExtraH2 reports whether h is a level-2 heading other than the two allowed
// plan sections.
func isExtraH2(h mdx.Heading) bool {
	return h.Level == 2 && h.Text != bgHeading && h.Text != wnhHeading
}

func firstH2Line(headings []mdx.Heading) int {
	for _, h := range headings {
		if h.Level == 2 {
			return h.Line
		}
	}
	return 0
}

var openQuestionsRe = regexp.MustCompile(`(?i)open\s+questions`)

func isOpenQuestions(text string) bool {
	return openQuestionsRe.MatchString(text)
}
