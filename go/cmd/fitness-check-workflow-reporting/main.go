package check

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "workflow-reporting", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// workflowDir holds the workflow files the check reads.
const workflowDir = ".github/workflows/"

// summaryFile is the variable naming the file a step appends the job
// summary to.
const summaryFile = "GITHUB_STEP_SUMMARY"

// Rules the findings name.
const (
	ruleReport = "job-report"
	ruleYAML   = "workflow-yaml"
)

func run(root string, _ []string) (checkkit.Result, error) {
	files := workflowFiles(root)
	var findings []checkkit.Finding
	var unread []string
	for _, f := range files {
		found, err := judgeFile(root, f)
		if err != nil {
			unread = append(unread, walkfs.Unreadable(f, err))
		}
		findings = append(findings, found...)
	}
	result := checkkit.Found(len(files)-len(unread), render, findings...)
	result.Info = unread
	return result, nil
}

// render is a finding's error line: where, what is wrong, and the fix.
func render(f checkkit.Finding) string {
	return fmt.Sprintf("%s:%d: %s; %s", f.File, f.Line, f.Message, f.Fix)
}

// workflowFiles is the in-scope workflow files directly under
// .github/workflows.
func workflowFiles(root string) []string {
	var out []string
	for _, f := range walkfs.InScope(walkfs.FilesByExt(root, ".yml", ".yaml")) {
		if rest, ok := strings.CutPrefix(f, workflowDir); ok && !strings.Contains(rest, "/") {
			out = append(out, f)
		}
	}
	return out
}

// judgeFile reads one workflow and judges each of its jobs. A file YAML
// cannot read is one finding at the line where reading stopped. An error
// means the file could not be read at all.
func judgeFile(root, file string) ([]checkkit.Finding, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, err
	}
	doc, err := parseYAML(string(content))
	var bad *syntaxError
	if errors.As(err, &bad) {
		return []checkkit.Finding{{File: file, Line: bad.line, Rule: ruleYAML,
			Message: "this workflow is not valid YAML: " + bad.msg, Fix: "fix the YAML on this line"}}, nil
	}
	var out []checkkit.Finding
	for _, job := range doc.get("jobs").mapEntries() {
		out = appendFinding(out, judgeJob(file, job))
	}
	return out, nil
}

// mapEntries is a mapping's entries, or none for any other node.
func (n *node) mapEntries() []entry {
	if n == nil || n.kind != mapNode {
		return nil
	}
	return n.entries
}

// appendFinding appends f to list unless it is nil.
func appendFinding(list []checkkit.Finding, f *checkkit.Finding) []checkkit.Finding {
	if f == nil {
		return list
	}
	return append(list, *f)
}

// judgeJob judges one job. A job without steps, such as one that calls a
// reusable workflow, has nothing to report from.
// A matrix job is one definition, so it is judged once.
func judgeJob(file string, job entry) *checkkit.Finding {
	steps := job.val.get("steps")
	if steps == nil || hasReporter(steps) {
		return nil
	}
	f := &checkkit.Finding{File: file, Line: job.line, Rule: ruleReport,
		Message: fmt.Sprintf("job %s has no step that runs with if: always() and writes $%s, so a failed run shows only a badge", job.key, summaryFile),
		Fix:     fmt.Sprintf("add a last step with `if: ${{ always() }}` whose run writes the job's status to \"$%s\"", summaryFile)}
	if line := summaryLine(steps); line > 0 {
		f.Message = fmt.Sprintf("job %s writes $%s only when earlier steps pass, at line %d", job.key, summaryFile, line)
		f.Fix = fmt.Sprintf("add `if: ${{ always() }}` to the step at line %d", line)
	}
	return f
}

// hasReporter reports whether any step runs with always() and writes the
// job summary.
func hasReporter(steps *node) bool {
	for _, step := range steps.items {
		if writesSummary(step) && runsAlways(step.get("if")) {
			return true
		}
	}
	return false
}

// summaryLine is the line of the first step that writes the job summary,
// or 0 when none does.
func summaryLine(steps *node) int {
	for _, step := range steps.items {
		if writesSummary(step) {
			return step.line
		}
	}
	return 0
}

// writesSummary reports whether a step's run script names the summary
// file.
func writesSummary(step *node) bool {
	run := step.get("run")
	return run != nil && strings.Contains(run.text, summaryFile)
}

// runsAlways reports whether an `if:` condition is always(), written bare
// or inside ${{ }}.
func runsAlways(cond *node) bool {
	if cond == nil {
		return false
	}
	text := strings.TrimSpace(cond.text)
	if inner, ok := strings.CutPrefix(text, "${{"); ok {
		text = strings.TrimSpace(strings.TrimSuffix(inner, "}}"))
	}
	return text == "always()"
}
