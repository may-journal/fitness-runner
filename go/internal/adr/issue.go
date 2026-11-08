package adr

import "strings"

// MarkerPrefix opens the hidden line naming the file an `ADR` issue came
// from. A second migration finds the issue by it instead of opening another.
const MarkerPrefix = "<!-- fitness:adr-migrate:"

// Marker is the hidden line naming rel as an issue's source file.
func Marker(rel string) string {
	return MarkerPrefix + rel + " -->"
}

// SourceOf is the file an issue body's marker names, or "".
func SourceOf(body string) string {
	_, rest, ok := strings.Cut(body, MarkerPrefix)
	if !ok {
		return ""
	}
	rel, _, _ := strings.Cut(rest, " -->")
	return strings.TrimSpace(rel)
}

// IssueTitle is the `[ADR] Short decision` title of r's issue.
func (r Record) IssueTitle() string {
	return "[ADR] " + r.Title
}

// notRecorded fills a decision section the file left out.
const notRecorded = "Not recorded in the file this ADR came from."

// IssueBody is the `ADR` issue body for r, from the file at rel. A status
// note that links a newer ADR issue makes a Superseded status name it.
func (r Record) IssueBody(rel string) string {
	pitch := r.Summary
	if pitch == "" {
		pitch = r.Title
	}
	var b strings.Builder
	b.WriteString("> " + pitch + "\n\n## Status\n\n" + r.statusLine() + "\n")
	for _, s := range DecisionSections {
		b.WriteString("\n## " + s + "\n\n" + r.sectionText(s) + "\n")
	}
	b.WriteString("\n" + Marker(rel) + "\n")
	return b.String()
}

// statusLine is r's one-word Status, or `Superseded by` and the issue URL
// its note links.
func (r Record) statusLine() string {
	if url := FirstIssueURL(r.StatusNote); r.Status == "Superseded" && url != "" {
		return "Superseded by " + url
	}
	return r.Status
}

// sectionText is r's text for section s, with the status note closing the
// Context, since an issue's Status holds one word.
func (r Record) sectionText(s string) string {
	text := r.Sections[s]
	if text == "" {
		text = notRecorded
	}
	if s == "Context" && r.StatusNote != "" {
		text += "\n\nStatus note from the file: " + r.StatusNote
	}
	return text
}

// Pointer is the README a runner ADR keeps once its decision is an issue. It
// holds the title, summary, issue URL and `## Check` section.
func (r Record) Pointer(title, url string) string {
	parts := []string{"# " + title}
	if r.Summary != "" {
		parts = append(parts, r.Summary)
	}
	parts = append(parts, url)
	if r.Check != "" {
		parts = append(parts, "## Check", r.Check)
	}
	return strings.Join(parts, "\n\n") + "\n"
}
