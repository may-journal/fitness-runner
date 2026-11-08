package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/may-journal/fitness-runner/go/internal/boxes"
	"github.com/may-journal/fitness-runner/go/internal/render"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// jsonFlag names the file that receives the run's performance report: each
// check's time, stage, files, binary size, and error lines, as JSON.
//
//fitness:usage --json=<file>
const jsonFlag = "--json"

// perfReport is the `--json` record of one run. The box diagram reads the
// same type, so the writer and the reader share one schema.
type perfReport = boxes.Report

// perfCheck is one check's entry in that record.
type perfCheck = boxes.Check

// jsonPath pulls `--json=<file>` out of argv, so neither check selection nor
// a single check's passthrough sees it.
func jsonPath(argv []string) (path string, rest []string, err error) {
	return oneValueFlag(argv, jsonFlag, "<file>")
}

// oneValueFlag pulls every `flag` and `flag=<value>` out of argv, accepting none
// or one valued flag; anything else is an error naming the form.
func oneValueFlag(argv []string, flag, form string) (value string, rest []string, err error) {
	flags, rest := splitFlag(argv, flag)
	if len(flags) == 0 {
		return "", rest, nil
	}
	value = strings.TrimPrefix(flags[0], flag+"=")
	if len(flags) > 1 || value == "" || value == flag {
		return "", nil, errors.New("use " + flag + "=" + form + " once")
	}
	return value, rest, nil
}

// splitFlag separates every `flag` and `flag=<value>` argument from the rest
// of argv.
func splitFlag(argv []string, flag string) (flags, rest []string) {
	for _, a := range argv {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			flags = append(flags, a)
			continue
		}
		rest = append(rest, a)
	}
	return flags, rest
}

// wantsPerf reports whether a run's report has a reader: a `--json` file or
// the CI job summary.
func wantsPerf(path string) bool {
	return path != "" || isGitHubActions()
}

// perfFor builds the report only when `--json` or the CI summary reads it, so a
// plain local run never stats the binaries.
func perfFor(wanted bool, checks []resolved, outcomes []outcome, rows []render.Row, elapsedMs int64) perfReport {
	if !wanted {
		return perfReport{}
	}
	return buildPerfReport(checks, outcomes, rows, elapsedMs)
}

// buildPerfReport pairs each check with its row, in dispatch order.
func buildPerfReport(checks []resolved, outcomes []outcome, rows []render.Row, elapsedMs int64) perfReport {
	runner, _ := os.Executable()
	r := perfReport{ElapsedMs: elapsedMs, RunnerBytes: fileBytes(runner), Checks: make([]perfCheck, len(checks))}
	for i, c := range checks {
		r.Checks[i] = perfCheck{
			Name:         c.name,
			Ok:           rows[i].Ok,
			Ms:           rows[i].Ms,
			Stage:        stageOf(c),
			TimedOut:     outcomes[i].timedOut,
			FilesChecked: rows[i].FilesChecked,
			BinaryBytes:  fileBytes(c.prog.Path),
			ErrorLines:   len(rows[i].Errors),
		}
	}
	return r
}

// writePerfReport writes the report as indented JSON, creating its folder.
func writePerfReport(path string, r perfReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// fileBytes returns a file's size, or 0 when it cannot be read.
func fileBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// runPerf prints each check's duration from a retained `--json` record,
// slowest first, with the stage whose total times it.
func runPerf(args []string) int {
	if len(args) != 1 {
		report.Error("fitness perf", errors.New("use fitness perf <report.json>"))
		return 2
	}
	r, err := readReport(args[0])
	if err != nil {
		report.Error("fitness perf", err)
		return 1
	}
	writeDurations(os.Stdout, r)
	return 0
}

// writeDurations prints one row per check, slowest first, then the run's
// wall time.
func writeDurations(w io.Writer, r perfReport) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tSTAGE\tTIME\tRESULT")
	for _, c := range slowestFirst(r.Checks) {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.Name, orDash(c.Stage), took(c.Ms), c.Result())
	}
	_ = tw.Flush()
	fmt.Fprintf(w, "The run took %s.\n", took(r.ElapsedMs))
}
