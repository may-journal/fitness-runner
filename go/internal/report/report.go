// Package report integrates command outcomes with GitHub Actions.
package report

import (
	"fmt"
	"os"
	"strings"
)

const annotationLimit = 10

// Enabled reports whether workflow commands are understood by the caller.
func Enabled() bool { return os.Getenv("GITHUB_ACTIONS") == "true" }

// Error preserves a terminal diagnostic and publishes a global CI failure.
func Error(operation string, err error) {
	if err == nil {
		return
	}
	console := operation + ": " + err.Error()
	if Enabled() {
		console = "Fitness: " + strings.ReplaceAll(strings.ReplaceAll(console, "\r", "\n"), "\n", "\nFitness: ")
	}
	fmt.Fprintln(os.Stderr, console)
	if !Enabled() {
		return
	}
	MarkFailure()
	fmt.Fprintln(os.Stdout, Annotation(operation, err.Error(), false))
	WriteSummary("## ❌ " + EscapeMarkdown(operation) + "\n\n" + EscapeMarkdown(err.Error()) + "\n\n")
}

// Outcome publishes a successful operation in the job summary.
func Outcome(operation, message string) {
	WriteSummary("## ✅ " + EscapeMarkdown(operation) + "\n\n" + EscapeMarkdown(message) + "\n\n")
}

// EmitAnnotations publishes at most ten errors per call, disclosing overflow.
func EmitAnnotations(check string, errors []string, locate bool) {
	if !Enabled() {
		return
	}
	if len(errors) > 0 {
		MarkFailure()
	}
	for _, message := range errors[:min(len(errors), annotationLimit)] {
		fmt.Fprintln(os.Stdout, Annotation(check, message, locate))
	}
	if len(errors) > annotationLimit {
		fmt.Fprintf(os.Stdout, "::warning::%d more issue(s) not annotated; see the complete report in the job summary or logs\n", len(errors)-annotationLimit)
	}
}

// Annotation returns a safely escaped workflow command, optionally locating it.
func Annotation(check, message string, locate bool) string {
	properties := ""
	if locate {
		properties = location(message)
	}
	return "::error" + properties + "::" + escapeData(check+": "+message)
}

func escapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

func escapeProperty(s string) string {
	return strings.NewReplacer(":", "%3A", ",", "%2C").Replace(escapeData(s))
}

// EscapeMarkdown keeps diagnostic text from injecting Markdown or HTML.
func EscapeMarkdown(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "|", "\\|", "#", "\\#", "\r", "", "\n", "<br>").Replace(s)
}

// MarkFailure tells later workflow steps that a specific diagnostic was emitted.
func MarkFailure() {
	if !Enabled() {
		return
	}
	path := os.Getenv("GITHUB_ENV")
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not mark reported failure: %v\n", err)
		return
	}
	defer file.Close()
	if _, err := fmt.Fprintln(file, "FITNESS_FAILURE_REPORTED=true"); err != nil {
		fmt.Fprintf(os.Stderr, "Could not mark reported failure: %v\n", err)
	}
}
