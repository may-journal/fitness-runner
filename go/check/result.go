package check

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Result is a check's verdict: whether it passed, an error line per
// problem, how many files it judged, its findings, and info notes that
// never fail a run.
type Result = checkkit.Result

// Finding is one problem in a shape an agent acts on without parsing
// prose: File, Line, Column, Rule, Message and Fix. Fitness fills in the
// command that explains the check.
type Finding = checkkit.Finding

// Found is the verdict for findings: it passes with none, and fails with
// one error line per finding, as `file:line: message (rule); fix`.
func Found(files int, findings ...Finding) Result {
	return checkkit.Found(files, Line, findings...)
}

// Line renders a finding as the error line a person reads.
func Line(f Finding) string {
	where := f.File
	if f.Line > 0 {
		where = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	line := f.Message + " (" + f.Rule + ")"
	if where != "" {
		line = where + ": " + line
	}
	if f.Fix != "" {
		line += "; " + f.Fix
	}
	return line
}

// Pass is a passing result that judged files files.
func Pass(files int) Result {
	return checkkit.Pass(files)
}

// Fail is a failing result that judged files files, with one error line per
// problem, for a check that reports no findings.
func Fail(files int, errors ...string) Result {
	return checkkit.Fail(files, errors...)
}

// Info is a passing result that carries notes, such as a tool the check
// could not find.
func Info(files int, notes ...string) Result {
	return checkkit.Info(files, notes...)
}
