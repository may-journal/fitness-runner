package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/render"
)

// formatFlag picks how a run prints its results: `--format=json` prints
// every check's result and findings to stdout as one JSON object, and moves
// the table people read to stderr.
const formatFlag = "--format"

// formatJSON is set when the run prints its results as JSON. run sets it
// once, before any check runs.
var formatJSON bool

// takeFormat pulls `--format=<format>` out of argv, so neither check
// selection nor a single check's passthrough sees it. Only json is a format;
// table, the default, is accepted too.
func takeFormat(argv []string) (json bool, rest []string, err error) {
	format, rest, err := oneValueFlag(argv, formatFlag, "json")
	if err != nil || (format != "" && !slices.Contains([]string{"json", "table"}, format)) {
		return false, nil, errors.New("use --format=json or --format=table once")
	}
	return format == "json", rest, nil
}

// ready fails a run when fitness has no checks, and otherwise takes the
// output format out of argv, returning the rest.
func ready(argv []string) (rest []string, err error) {
	if installErr != nil {
		return nil, installErr
	}
	formatJSON, rest, err = takeFormat(argv)
	return rest, err
}

// runReport is what `fitness --format=json` prints.
type runReport struct {
	// Ok is false when any check failed.
	Ok bool `json:"ok"`
	// Checks holds one entry per check that ran, in table order.
	Checks []checkReport `json:"checks"`
}

// checkReport is one check's entry in the JSON output.
type checkReport struct {
	// Name is the check's name, which `fitness <name>` runs alone.
	Name string `json:"name"`
	// Status is passed, failed, or info, as the table shows it.
	Status string `json:"status"`
	// FilesChecked counts the files judged; -1 when the check never judged.
	FilesChecked int `json:"filesChecked"`
	// Ms is the check's run time in milliseconds.
	Ms int64 `json:"ms"`
	// Errors are the check's error lines, as the table shows them.
	Errors []string `json:"errors"`
	// Info are notes that never fail the run.
	Info []string `json:"info,omitempty"`
	// Findings are the errors in a shape an agent acts on. A check that
	// reports none gets one finding per error line, with the check's name
	// as its rule.
	Findings []checkkit.Finding `json:"findings"`
}

// printJSON prints the run's results as one JSON object on stdout, when
// the run asked for JSON. found holds each row's findings, as allFindings
// gives them.
func printJSON(rows []render.Row, found [][]checkkit.Finding, totals render.Totals) {
	if !formatJSON {
		return
	}
	report := runReport{Ok: totals.Failure == 0, Checks: make([]checkReport, len(rows))}
	for i, row := range rows {
		report.Checks[i] = checkReport{
			Name: row.Name, Status: status(row), FilesChecked: row.FilesChecked, Ms: row.Ms,
			Errors: nonNil(row.Errors), Info: row.Info, Findings: found[i],
		}
	}
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness --format=json: "+err.Error())
		return
	}
	fmt.Println(string(out))
}

// status names a row's result as the table does.
func status(row render.Row) string {
	switch {
	case !row.Ok:
		return "failed"
	case row.IsInfo():
		return "info"
	}
	return "passed"
}

// allFindings returns each row's findings, made once for every output that
// shows them: the JSON, the job summary and the CI annotations.
func allFindings(checks []resolved, outcomes []outcome, rows []render.Row) [][]checkkit.Finding {
	found := make([][]checkkit.Finding, len(rows))
	for i, row := range rows {
		found[i] = findingsFor(checks[i].name, outcomes[i], row)
	}
	return found
}

// findingsFor returns a check's findings for the JSON output, as
// ownFindings picks them, such as one per error line for a timeout. Every
// finding names the command that explains its check.
func findingsFor(name string, o outcome, row render.Row) []checkkit.Finding {
	found := ownFindings(name, o, row)
	for i := range found {
		if found[i].Explain == "" {
			found[i].Explain = checkkit.ExplainCommand(name)
		}
	}
	return nonNil(found)
}

// ownFindings returns a copy of the check's own findings when its row shows
// the errors it printed, and otherwise one finding per error line, ruled by
// the check's name.
func ownFindings(name string, o outcome, row render.Row) []checkkit.Finding {
	if len(o.res.Findings) > 0 && slices.Equal(o.res.Errors, row.Errors) {
		return slices.Clone(o.res.Findings)
	}
	found := make([]checkkit.Finding, len(row.Errors))
	for i, e := range row.Errors {
		found[i] = checkkit.Finding{Rule: name, Message: e}
	}
	return found
}

// nonNil turns a nil slice into an empty one, so JSON shows [] not null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
