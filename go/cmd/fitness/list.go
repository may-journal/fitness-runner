package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// listed is one check as `fitness list` shows it.
type listed struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Scope   string `json:"scope"`
	// BudgetMs is the time the check may run before it is stopped.
	BudgetMs int64 `json:"budgetMs"`
	// Enabled says whether this repo's config runs the check.
	Enabled bool `json:"enabled"`
	// Blob says whether `fitness --blob` runs the check on a text.
	Blob bool `json:"blob"`
	// Note says why the row holds no metadata: the check's program is not
	// installed, or its describe handshake failed.
	Note string `json:"note,omitempty"`
}

// runList handles `fitness list [--json]`: every check this repo can run,
// built-in, beside the runner or local, in catalog order, with what its
// describe handshake says and whether the repo's config enables it.
func runList(args []string) int {
	asJSON, err := listFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness list:", err)
		return 2
	}
	if installErr != nil {
		report.Error("fitness list", installErr)
		return 1
	}
	checks, err := listChecks()
	if err != nil {
		report.Error("fitness list", err)
		return 1
	}
	if asJSON {
		return writeJSON(os.Stdout, checks)
	}
	printList(os.Stdout, checks)
	return 0
}

// listFlags reads the one flag `fitness list` takes.
func listFlags(args []string) (asJSON bool, err error) {
	asJSON, rest := takeJSONFlag(args)
	if len(rest) > 0 {
		return false, fmt.Errorf("unknown argument %q; use --json or nothing", rest[0])
	}
	return asJSON, nil
}

// takeJSONFlag takes --json out of args, saying whether it was there.
func takeJSONFlag(args []string) (asJSON bool, rest []string) {
	for _, a := range args {
		if a == "--json" {
			asJSON = true
			continue
		}
		rest = append(rest, a)
	}
	return asJSON, rest
}

// seconds writes a budget in milliseconds as seconds, as both commands
// show it.
func seconds(ms int64) string {
	return trimFloat(float64(ms) / 1000)
}

// listChecks describes every check in the catalog and marks the ones the
// repo's config selects. A check whose program is missing, or whose
// handshake failed, still gets its row, with a note saying why, so one broken
// check never hides the rest.
func listChecks() ([]listed, error) {
	_, cfg, suite, err := blobConfig()
	if err != nil {
		return nil, err
	}
	prebuild(checkCatalog())
	found, missing := installedChecks(checkCatalog())
	checks, err := resolveList(found)
	if err != nil {
		return nil, err
	}
	byName := map[string]resolved{}
	for _, c := range withBudgets(cfg, checks) {
		byName[c.name] = c
	}
	var out []listed
	for _, name := range checkCatalog() {
		out = append(out, listRowFor(name, byName, missing, suite))
	}
	return out, nil
}

// installedChecks splits names into those with a check program and those
// without, keeping their order. Each missing one maps to the note saying
// why.
func installedChecks(names []string) (found []string, missing map[string]string) {
	missing = map[string]string{}
	for _, name := range names {
		if _, err := findCheckBinary(name); err != nil {
			missing[name] = notInstalled(name, err)
			continue
		}
		found = append(found, name)
	}
	return found, missing
}

// notInstalled says why check name has no program: a local check that
// cannot build says why, and any other has none installed.
func notInstalled(name string, err error) string {
	if _, local := localBins[name]; local {
		return err.Error()
	}
	return "not installed: no fitness-check-" + name + " program"
}

// listRowFor is the row for check name: its metadata, or a note saying the
// program is not installed.
func listRowFor(name string, byName map[string]resolved, missing map[string]string, suite []string) listed {
	enabled := slices.Contains(suite, name)
	if note, ok := missing[name]; ok {
		return listed{Name: name, Enabled: enabled, Note: note}
	}
	c := byName[name]
	return listed{Name: name, Summary: c.desc.Summary, Scope: c.desc.Scope, BudgetMs: timeoutFor(c).Milliseconds(),
		Enabled: enabled, Blob: c.desc.Blob, Note: cmp.Or(c.needs, c.describeErr)}
}

// printList writes one aligned row per check: name, scope, budget, whether
// it is enabled, and its summary, or the note saying why it has none.
func printList(w io.Writer, checks []listed) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tSCOPE\tBUDGET\tENABLED\tSUMMARY")
	for _, c := range checks {
		fmt.Fprintf(tw, "%s\t%s\t%ss\t%s\t%s\n", c.Name, orDash(c.Scope),
			seconds(c.BudgetMs), yesNo(c.Enabled), summaryOrNote(c))
	}
	_ = tw.Flush()
}

// summaryOrNote is a row's summary, followed by its note in brackets when it
// has one.
func summaryOrNote(c listed) string {
	if c.Note == "" {
		return orDash(c.Summary)
	}
	return orDash(c.Summary) + " (" + c.Note + ")"
}

// writeJSON writes v as indented JSON and returns the exit code.
func writeJSON(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "fitness:", err)
		return 1
	}
	return 0
}

// orDash shows a value a check left empty as a dash.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// yesNo spells a flag as a reader says it.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
