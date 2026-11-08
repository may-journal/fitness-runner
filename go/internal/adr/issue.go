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

// IssueBody is the `ADR` issue body for r, from the file at rel. The issue's
// state carries its status, so the body holds no Status section.
func (r Record) IssueBody(rel string) string {
	pitch := r.Summary
	if pitch == "" {
		pitch = r.Title
	}
	var b strings.Builder
	b.WriteString("> " + pitch + "\n")
	for i, s := range DecisionSections {
		b.WriteString("\n## " + IssueSections[i] + "\n\n" + r.sectionText(s) + "\n")
	}
	b.WriteString("\n" + Marker(rel) + "\n")
	return b.String()
}

// Superseded reports whether r's file retired its decision, which closes its
// issue as not planned.
func (r Record) Superseded() bool {
	return r.Status == "Superseded"
}

// CloseComment is the comment that closes a superseded ADR's issue, naming
// the issue its status note links.
func (r Record) CloseComment() string {
	if url := r.supersededBy(); url != "" {
		return "Superseded by " + url
	}
	return "Superseded, as the file it came from said."
}

// supersededBy is the issue URL a superseded ADR's `by` note links, or "".
func (r Record) supersededBy() string {
	if !r.Superseded() || !strings.HasPrefix(r.StatusNote, "by ") {
		return ""
	}
	return FirstIssueURL(r.StatusNote)
}

// statusLine is r's Status as a check README keeps it, with `by` and the
// issue URL its note links.
func (r Record) statusLine() string {
	if url := r.supersededBy(); url != "" {
		return "Superseded by " + url
	}
	return r.Status
}

// sectionText is r's text for file section s, with a status note the
// issue's state cannot hold closing the Context.
func (r Record) sectionText(s string) string {
	text := r.Sections[s]
	if text == "" {
		text = notRecorded
	}
	if s == DecisionSections[0] && r.StatusNote != "" && r.supersededBy() == "" {
		text += "\n\nStatus note from the file: " + r.StatusNote
	}
	return text
}

// Pointer is the README a runner ADR keeps once its decision is an issue. It
// holds the title, summary, issue URL, a superseded status and `## Check`.
func (r Record) Pointer(title, url string) string {
	parts := []string{"# " + pointerTitle(title, r.Title)}
	if r.Summary != "" {
		parts = append(parts, r.Summary)
	}
	parts = append(parts, url)
	if r.Superseded() {
		parts = append(parts, "Status: "+r.statusLine())
	}
	if r.Check != "" {
		parts = append(parts, "## Check", r.Check)
	}
	return strings.Join(parts, "\n\n") + "\n"
}

// pointerTitle is a check README's title, dropping a leading ADR word so
// the trimmed README is no ADR by its title.
func pointerTitle(title, decision string) string {
	if titleADR.MatchString(title) {
		return decision
	}
	return title
}
