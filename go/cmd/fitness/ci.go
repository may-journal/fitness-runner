package main

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/render"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// maxAnnotations is GitHub's per-step display cap for error annotations. Past
// it GitHub silently drops them, so we cap deliberately and record the
// remainder in the job summary, which stays the complete record.
const maxAnnotations = 10

// finish emits the CI report when running under GitHub Actions, then maps the
// failure count to the process exit code.
func finish(rows []render.Row, success, failure, files int, elapsedMs int64) int {
	if isGitHubActions() {
		emitCIReport(rows, success, failure, files, elapsedMs)
	}
	if failure > 0 {
		return 1
	}
	return 0
}

// isGitHubActions reports whether the runner is executing inside a GitHub
// Actions job.
func isGitHubActions() bool {
	return report.Enabled()
}

// emitCIReport writes the job summary and prints error annotations when
// running under GitHub Actions. It never changes the process result — the
// exit code still comes from the check outcomes — so it is additive to the
// terminal table a local run prints.
func emitCIReport(rows []render.Row, success, failure, files int, elapsedMs int64) {
	writeStepSummary(ciSummaryMarkdown(rows, success, failure, files, elapsedMs))
	if failure > 0 {
		report.MarkFailure()
	}
	for _, line := range annotations(rows, maxAnnotations) {
		fmt.Println(line)
	}
}

// writeStepSummary appends md to the GITHUB_STEP_SUMMARY file when it is set.
func writeStepSummary(md string) {
	report.WriteSummary(md)
}

// ciSummaryMarkdown renders the results as a markdown job summary: a
// status table, then a section per failing check listing its errors.
func ciSummaryMarkdown(rows []render.Row, success, failure, files int, elapsedMs int64) string {
	var b strings.Builder
	b.WriteString("## Fitness checks\n\n")
	b.WriteString(headline(success, failure, files, elapsedMs) + "\n\n")
	b.WriteString("| Check | Status | Files | Time |\n| --- | --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %dms |\n", report.EscapeMarkdown(r.Name), statusMark(r.Ok), filesCell(r.FilesChecked), r.Ms)
	}
	for _, r := range rows {
		b.WriteString(rowDetail(r))
	}
	return b.String()
}

// headline is the one-line aggregate atop the summary: a positive line when
// every check passed, otherwise a count of what passed and what failed. Both
// forms name the files scanned and total time.
func headline(success, failure, files int, elapsedMs int64) string {
	total := success + failure
	if failure == 0 {
		return fmt.Sprintf("✅ **All %d checks passed** — %d files scanned in %dms", total, files, elapsedMs)
	}
	return fmt.Sprintf("❌ **%d of %d checks passed**, %d failed — %d files scanned in %dms",
		success, total, failure, files, elapsedMs)
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
	fmt.Fprintf(&b, "\n### ❌ %s\n\n", report.EscapeMarkdown(r.Name))
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "- %s\n", report.EscapeMarkdown(e))
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
		for _, e := range rowErrors(r) {
			out = append(out, annotationLine(r.Name, e))
		}
	}
	return out
}

// annotationLine delegates location parsing and escaping to the shared reporter.
func annotationLine(check, err string) string {
	return report.Annotation(check, err, true)
}

// rowErrors ensures a failing check without diagnostics still has an annotation.
func rowErrors(r render.Row) []string {
	if len(r.Errors) == 0 {
		return []string{"check failed without a diagnostic; see the check log"}
	}
	return r.Errors
}
