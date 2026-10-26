package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/render"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// maxAnnotations is GitHub's per-step display cap for error annotations. Past
// it GitHub silently drops them, so we cap deliberately and record the
// remainder in the job summary, which stays the complete record.
const maxAnnotations = 10

// finish emits the CI report when running under GitHub Actions, then maps the
// failure count to the process exit code. Info rows never change it.
func finish(rows []render.Row, found [][]checkkit.Finding, perf perfReport, t render.Totals) int {
	if isGitHubActions() {
		emitCIReport(rows, found, perf, t)
	}
	if t.Failure > 0 {
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
func emitCIReport(rows []render.Row, found [][]checkkit.Finding, perf perfReport, t render.Totals) {
	writeStepSummary(ciSummaryMarkdown(rows, found, t) + timingMarkdown(perf))
	if t.Failure > 0 {
		report.MarkFailure()
	}
	for _, line := range annotations(rows, found, maxAnnotations) {
		fmt.Println(line)
	}
	for _, line := range notices(rows, maxAnnotations) {
		fmt.Println(line)
	}
}

// writeStepSummary appends md to the GITHUB_STEP_SUMMARY file when it is set.
func writeStepSummary(md string) {
	report.WriteSummary(md)
}

// ciSummaryMarkdown renders the results as a markdown job summary: a
// status table, then a section per failing check listing its errors and an
// Info section for checks that could not judge.
func ciSummaryMarkdown(rows []render.Row, found [][]checkkit.Finding, t render.Totals) string {
	var b strings.Builder
	b.WriteString("## Fitness checks\n\n")
	b.WriteString(headline(t) + "\n\n")
	b.WriteString("| Check | Status | Files | Time |\n| --- | --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %dms |\n", report.EscapeMarkdown(r.Name), statusMark(r), filesCell(r.FilesChecked), r.Ms)
	}
	for i, r := range rows {
		b.WriteString(rowDetail(r, found[i]))
	}
	b.WriteString(infoSection(rows))
	return b.String()
}

// timingMarkdown renders each check's time against its budget, slowest
// first, so the check nearest its limit tops the table.
func timingMarkdown(perf perfReport) string {
	checks := slices.Clone(perf.Checks)
	slices.SortStableFunc(checks, func(a, b perfCheck) int { return cmp.Compare(b.Ms, a.Ms) })
	var b strings.Builder
	b.WriteString("\n### Check timing\n\n| Check | Time | Budget | Used |\n| --- | --- | --- | --- |\n")
	for _, c := range checks {
		fmt.Fprintf(&b, "| %s | %dms | %dms | %d%% |\n", report.EscapeMarkdown(c.Name), c.Ms, c.TimeoutMs, budgetShare(c))
	}
	return b.String()
}

// budgetShare is the percent of its budget a check spent.
func budgetShare(c perfCheck) int64 {
	if c.TimeoutMs <= 0 {
		return 0
	}
	return c.Ms * 100 / c.TimeoutMs
}

// headline is the one-line aggregate atop the summary: a positive line when
// every check passed, otherwise a count of what passed with the failures and
// info rows named. Both forms name the files scanned and total time.
func headline(t render.Totals) string {
	total := t.Success + t.Failure + t.Info
	if t.Failure == 0 && t.Info == 0 {
		return fmt.Sprintf("✅ **All %d checks passed** — %d files scanned in %dms", total, t.Files, t.Ms)
	}
	mark := "✅"
	if t.Failure > 0 {
		mark = "❌"
	}
	return fmt.Sprintf("%s **%d of %d checks passed**%s — %d files scanned in %dms",
		mark, t.Success, total, render.Counts(t), t.Files, t.Ms)
}

// statusMark renders a row's status as a marked word.
func statusMark(r render.Row) string {
	if !r.Ok {
		return "❌ fail"
	}
	if r.IsInfo() {
		return "ℹ️ info"
	}
	return "✅ pass"
}

// infoSection lists every info note under one heading, or "" when there is
// none. Info never fails the run.
func infoSection(rows []render.Row) string {
	var b strings.Builder
	for _, r := range rows {
		for _, note := range r.Info {
			fmt.Fprintf(&b, "- %s: %s\n", report.EscapeMarkdown(r.Name), report.EscapeMarkdown(note))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n### ℹ️ Info\n\n" + b.String()
}

// notices returns one notice per info note, capped at limit, with a final
// notice naming how many more the job summary lists.
func notices(rows []render.Row, limit int) []string {
	var all []string
	for _, r := range rows {
		for _, note := range r.Info {
			all = append(all, report.Notice(r.Name, note))
		}
	}
	if len(all) <= limit {
		return all
	}
	more := fmt.Sprintf("::notice::%d more info note(s); see the job summary", len(all)-limit)
	return append(all[:limit], more)
}

// filesCell renders the files count; a negative count (a crash or timeout)
// renders as a dash, matching the terminal table.
func filesCell(n int) string {
	if n < 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

// rowDetail renders one failing check's findings as a bulleted section, or
// "" when the check passed or carried no errors. The findings are the ones
// `--format=json` prints, so the summary and the JSON never disagree.
func rowDetail(r render.Row, found []checkkit.Finding) string {
	if r.Ok || len(r.Errors) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n### ❌ %s\n\n", report.EscapeMarkdown(r.Name))
	for _, f := range found {
		b.WriteString(report.FindingMarkdown(f))
	}
	return b.String()
}

// annotations returns the workflow-command annotation lines for the failing
// checks, capped at limit. When more findings exist than the cap, a trailing
// warning names how many were not annotated, so nothing is dropped silently.
func annotations(rows []render.Row, found [][]checkkit.Finding, limit int) []string {
	all := allAnnotationLines(rows, found)
	if len(all) <= limit {
		return all
	}
	note := fmt.Sprintf("::warning::%d more issue(s) not annotated; see the job summary", len(all)-limit)
	return append(all[:limit], note)
}

// allAnnotationLines builds one annotation per finding across every failing
// row, through the shared reporter.
func allAnnotationLines(rows []render.Row, found [][]checkkit.Finding) []string {
	var out []string
	for i, r := range rows {
		if r.Ok {
			continue
		}
		for _, f := range orUndiagnosed(r.Name, found[i]) {
			out = append(out, report.FindingAnnotation(r.Name, f))
		}
	}
	return out
}

// orUndiagnosed ensures a failing check without diagnostics still has an
// annotation.
func orUndiagnosed(check string, found []checkkit.Finding) []checkkit.Finding {
	if len(found) == 0 {
		return []checkkit.Finding{{Rule: check, Message: "check failed without a diagnostic; see the check log"}}
	}
	return found
}
