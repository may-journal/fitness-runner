package report

import (
	"fmt"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// FindingAnnotation returns the error annotation for one finding of check.
// A finding that names its file points there, at its line and column, with
// the check and rule as its title, and says how to fix it and where to read
// why. A finding with no file, such as an error line a check reported
// without findings, is located from its message text instead.
func FindingAnnotation(check string, f checkkit.Finding) string {
	if f.File == "" {
		return Annotation(check, f.Message, true)
	}
	return "::error " + findingProperties(check, f) + "::" + escapeData(findingText(f))
}

// findingProperties are the file, line, column and title of a finding's
// annotation.
func findingProperties(check string, f checkkit.Finding) string {
	properties := "file=" + escapeProperty(f.File)
	if f.Line > 0 {
		properties += ",line=" + strconv.Itoa(f.Line)
	}
	if f.Column > 0 {
		properties += ",col=" + strconv.Itoa(f.Column)
	}
	return properties + ",title=" + escapeProperty(check+" ("+f.Rule+")")
}

// FindingMarkdown returns one finding as a job summary list item: where it
// is, what is wrong, then how to fix it and where to read why, nested
// under it.
func FindingMarkdown(f checkkit.Finding) string {
	if f.File == "" {
		return "- " + EscapeMarkdown(f.Message) + "\n"
	}
	return "- " + EscapeMarkdown(where(f)+" - "+f.Message) + "\n" + nested(f)
}

// nested lists a finding's fix and explain command as nested list items.
func nested(f checkkit.Finding) string {
	out := ""
	for _, n := range notes(f) {
		out += "  - " + EscapeMarkdown(n) + "\n"
	}
	return out
}

// findingText is a finding's message, then its fix and explain command,
// each on its own line.
func findingText(f checkkit.Finding) string {
	text := f.Message
	for _, n := range notes(f) {
		text += "\n" + n
	}
	return text
}

// notes are the lines that follow a finding's message: how to fix it, and
// the command that says why its check exists, each when the finding has it.
func notes(f checkkit.Finding) []string {
	var out []string
	if f.Fix != "" {
		out = append(out, "Fix: "+f.Fix)
	}
	if f.Explain != "" {
		out = append(out, "Why: "+f.Explain)
	}
	return out
}

// where is a finding's place as file, file:line or file:line:column.
func where(f checkkit.Finding) string {
	switch {
	case f.Line == 0:
		return f.File
	case f.Column == 0:
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Column)
}
