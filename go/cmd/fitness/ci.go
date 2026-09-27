package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/render"
)

// maxAnnotations is GitHub's per-step display cap for error annotations. Past
// it GitHub silently drops them, so we cap deliberately and record the
// remainder in the job summary, which stays the complete record.
const maxAnnotations = 10

// finish emits the CI report when running under GitHub Actions, then maps the
// failure count to the process exit code.
func finish(rows []render.Row, failure int) int {
	if isGitHubActions() {
		emitCIReport(rows)
	}
	if failure > 0 {
		return 1
	}
	return 0
}

// isGitHubActions reports whether the runner is executing inside a GitHub
// Actions job.
func isGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// emitCIReport writes the job summary and prints error annotations when
// running under GitHub Actions. It never changes the process result — the
// exit code still comes from the check outcomes — so it is additive to the
// terminal table a local run prints.
func emitCIReport(rows []render.Row) {
	writeStepSummary(ciSummaryMarkdown(rows))
	for _, line := range annotations(rows, maxAnnotations) {
		fmt.Fprintln(os.Stdout, line)
	}
}

// writeStepSummary appends md to the GITHUB_STEP_SUMMARY file when it is set.
func writeStepSummary(md string) {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(md)
}

// ciSummaryMarkdown renders the results as a markdown job summary: a
// status table, then a section per failing check listing its errors.
func ciSummaryMarkdown(rows []render.Row) string {
	var b strings.Builder
	b.WriteString("## Fitness checks\n\n")
	b.WriteString("| Check | Status | Files | Time |\n| --- | --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %dms |\n", r.Name, statusMark(r.Ok), filesCell(r.FilesChecked), r.Ms)
	}
	for _, r := range rows {
		b.WriteString(rowDetail(r))
	}
	return b.String()
}

// statusMark renders a row's ok flag as a marked word.
func statusMark(ok bool) string {
	if ok {
		return "✅ pass"
	}
	return "❌ fail"
}

// filesCell renders the files count; a negative count (a crash or timeout)
// renders as a dash, matching the terminal table.
func filesCell(n int) string {
	if n < 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

// rowDetail renders one failing check's errors as a bulleted section, or ""
// when the check passed or carried no errors.
func rowDetail(r render.Row) string {
	if r.Ok || len(r.Errors) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n### ❌ %s\n\n", r.Name)
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "- %s\n", e)
	}
	return b.String()
}

// annotations returns the workflow-command annotation lines for the failing
// checks, capped at limit. When more errors exist than the cap, a trailing
// warning names how many were not annotated, so nothing is dropped silently.
func annotations(rows []render.Row, limit int) []string {
	all := allAnnotationLines(rows)
	if len(all) <= limit {
		return all
	}
	note := fmt.Sprintf("::warning::%d more issue(s) not annotated; see the job summary", len(all)-limit)
	return append(all[:limit], note)
}

// allAnnotationLines builds one annotation per error across every failing row.
func allAnnotationLines(rows []render.Row) []string {
	var out []string
	for _, r := range rows {
		if r.Ok {
			continue
		}
		for _, e := range r.Errors {
			out = append(out, annotationLine(r.Name, e))
		}
	}
	return out
}

var (
	// fileLineRe pulls a leading "path:line" out of an error string (cspell,
	// go-complexity, and the like format their errors this way).
	fileLineRe = regexp.MustCompile(`^([^\s:]+\.[A-Za-z0-9]+):(\d+)`)
	// fileOnlyRe pulls a leading "path:" when there is no line (prose-budget
	// reports a file-level violation without a line).
	fileOnlyRe = regexp.MustCompile(`^([^\s:]+\.[A-Za-z0-9]+):`)
)

// annotationLine formats one error as an ::error workflow command, linking it
// to a file and line when the error string carries them.
func annotationLine(check, err string) string {
	msg := escapeData(check + ": " + err)
	if m := fileLineRe.FindStringSubmatch(err); m != nil {
		return fmt.Sprintf("::error file=%s,line=%s::%s", escapeProp(m[1]), m[2], msg)
	}
	if m := fileOnlyRe.FindStringSubmatch(err); m != nil {
		return fmt.Sprintf("::error file=%s::%s", escapeProp(m[1]), msg)
	}
	return "::error::" + msg
}

// escapeData percent-encodes a workflow-command message per GitHub's spec.
func escapeData(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return strings.ReplaceAll(s, "\n", "%0A")
}

// escapeProp percent-encodes a workflow-command property value, which also
// forbids a bare comma and colon.
func escapeProp(s string) string {
	s = escapeData(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	return strings.ReplaceAll(s, ",", "%2C")
}
