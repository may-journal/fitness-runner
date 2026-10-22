package main

import (
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
	for _, a := range args {
		if a != "--json" {
			return false, fmt.Errorf("unknown argument %q; use --json or nothing", a)
		}
		asJSON = true
	}
	return asJSON, nil
}

// listChecks describes every check in the catalog and marks the ones the
// repo's config selects.
func listChecks() ([]listed, error) {
	_, cfg, suite, err := blobConfig()
	if err != nil {
		return nil, err
	}
	checks, err := resolveList(checkCatalog())
	if err != nil {
		return nil, err
	}
	checks = withBudgets(cfg, checks)
	out := make([]listed, len(checks))
	for i, c := range checks {
		out[i] = listed{Name: c.name, Summary: c.desc.Summary, Scope: c.desc.Scope,
			BudgetMs: timeoutFor(c).Milliseconds(), Enabled: slices.Contains(suite, c.name)}
	}
	return out, nil
}

// printList writes one aligned row per check: name, scope, budget, whether
// it is enabled, and its summary.
func printList(w io.Writer, checks []listed) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tSCOPE\tBUDGET\tENABLED\tSUMMARY")
	for _, c := range checks {
		fmt.Fprintf(tw, "%s\t%s\t%ss\t%s\t%s\n", c.Name, orDash(c.Scope),
			trimFloat(float64(c.BudgetMs)/1000), yesNo(c.Enabled), orDash(c.Summary))
	}
	_ = tw.Flush()
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
