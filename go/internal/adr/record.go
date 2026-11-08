package adr

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Record is one ADR file's parts, as an `ADR` issue holds them.
type Record struct {
	// Title is the decision, without the ADR word or number.
	Title string
	// Summary is the one-line summary under the title, or "".
	Summary string
	// Status is Proposed, Accepted or Superseded.
	Status string
	// StatusNote is what the file's status line says past its first word.
	StatusNote string
	// Sections holds the Context, Decision and Consequences text by name.
	Sections map[string]string
	// Check is a runner ADR's `## Check` section, which stays in its README.
	Check string
}

// numberPrefix matches what comes before the decision in an ADR title: the
// ADR word, its number and a separator.
var numberPrefix = regexp.MustCompile(`(?i)^(?:ADR\b[\s#:-]*)?(?:\d+(?:\s*[—–:.\-]+\s*|\s+))?`)

// statusLine matches a `Status: accepted` line and takes its text.
var statusLine = regexp.MustCompile(`(?i)^[\s>*_-]*status[\s*_]*:[\s*_]*(.*?)[\s*_]*$`)

// statusWords maps a status's first word, lowercased, to an ADR issue's
// Status; a word it lacks reads as Proposed.
var statusWords = map[string]string{
	"": "Accepted", "accepted": "Accepted", "proposed": "Proposed",
	"superseded": "Superseded", "deprecated": "Superseded", "retired": "Superseded",
}

// Parse reads an ADR file's parts.
func Parse(content string) Record {
	title := Title(content)
	r := Record{Title: decision(title), Sections: map[string]string{}}
	r.Summary, r.Status, r.StatusNote = preamble(content)
	for _, s := range DecisionSections {
		r.Sections[s], _ = mdtemplate.SectionText(content, s)
	}
	r.Check, _ = mdtemplate.SectionText(content, "Check")
	if status, ok := mdtemplate.SectionText(content, "Status"); ok {
		r.Status, r.StatusNote = statusOf(status)
	}
	return r
}

// decision is a title without its ADR word and number, or the whole title
// when nothing follows them.
func decision(title string) string {
	if rest := numberPrefix.ReplaceAllString(title, ""); rest != "" {
		return rest
	}
	return title
}

// preamble reads the lines between the title and the first section: the
// summary paragraph and the status line.
func preamble(content string) (summary, status, note string) {
	status = statusWords[""]
	for _, line := range before(content) {
		if m := statusLine.FindStringSubmatch(line); m != nil {
			status, note = statusOf(m[1])
		} else if summary == "" {
			summary = line
		}
	}
	return summary, status, note
}

// before is each non-blank line after the title and before the first `##`.
func before(content string) []string {
	var out []string
	for _, line := range strings.Split(mdx.StripComments(mdx.StripFrontMatter(content)), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## "):
			return out
		case line != "" && !strings.HasPrefix(line, "# "):
			out = append(out, line)
		}
	}
	return out
}

// statusOf maps a status text to a Status and the note past its first word.
func statusOf(text string) (string, string) {
	word, note, _ := strings.Cut(strings.TrimSpace(text), " ")
	word = strings.ToLower(strings.Trim(word, ".,;:*_"))
	status, ok := statusWords[word]
	if !ok {
		status, note = "Proposed", strings.TrimSpace(text)
	}
	return status, strings.TrimLeft(strings.TrimSpace(note), ";,:— ")
}
