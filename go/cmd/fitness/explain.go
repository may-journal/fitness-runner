package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// explained is one check as `fitness explain --json` gives it.
type explained struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Scope   string `json:"scope"`
	// BudgetMs is the time the check may run before it is stopped.
	BudgetMs int64 `json:"budgetMs"`
	// Note says why the check's describe handshake gave no metadata.
	Note string `json:"note,omitempty"`
	// Body is the check's documentation as Markdown.
	Body string `json:"body"`
}

// runExplain handles `fitness explain <check> [--json]`: the check's
// documentation, generated from its Go doc comments into this binary, with
// what its describe handshake says. An unknown name exits 2 and suggests the
// closest names.
func runExplain(args []string) int {
	name, asJSON, err := explainArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness explain:", err)
		return 2
	}
	cfg, err := repoConfig()
	if err != nil {
		report.Error("fitness explain", err)
		return 1
	}
	if catalog := knownChecks(); !slices.Contains(catalog, name) {
		fmt.Fprintf(os.Stderr, "fitness explain: no check named %q; %s\n", name, suggest(name, catalog))
		return 2
	}
	e, err := explainCheck(cfg, name)
	if err != nil {
		report.Error("fitness explain", err)
		return 1
	}
	return printExplained(e, asJSON)
}

// printExplained writes the explanation as Markdown, or as JSON.
func printExplained(e explained, asJSON bool) int {
	if asJSON {
		return writeJSON(os.Stdout, e)
	}
	fmt.Printf("# %s\n\nScope: %s · Budget: %ss\n\n%s%s", e.Name, orDash(e.Scope), seconds(e.BudgetMs), noteLine(e.Note), e.Body)
	return 0
}

// noteLine is the paragraph saying why a check gave no metadata, or "".
func noteLine(note string) string {
	if note == "" {
		return ""
	}
	return "Note: " + note + "\n\n"
}

// explainArgs reads one check name and an optional --json flag.
func explainArgs(args []string) (name string, asJSON bool, err error) {
	asJSON, rest := takeJSONFlag(args)
	if len(rest) != 1 || strings.HasPrefix(rest[0], "-") {
		return "", false, fmt.Errorf("name one check, as `fitness explain <check> [--json]`; `fitness list` names them all")
	}
	return rest[0], asJSON, nil
}

// repoConfig loads this repo's config and local checks, so a local check
// explains itself too, with the repo's budgets. A config or local check
// folder that cannot load is an error, as it is for `fitness list`, and so
// is an install that found no checks.
func repoConfig() (*conf.Config, error) {
	if installErr != nil {
		return nil, installErr
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}
	return cfg, loadLocalChecks(cfg, root)
}

// knownChecks are the names explain answers for: the repo's catalog, and
// every built-in check, even one a language ignore file leaves out of runs.
func knownChecks() []string {
	names := checkCatalog()
	for name := range builtins {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// explainCheck describes the check and reads its documentation: the
// generated docs of a built-in check, else the summary its handshake gives.
// A failed handshake comes back as the note saying why.
func explainCheck(cfg *conf.Config, name string) (explained, error) {
	checks, err := resolveList([]string{name})
	if err != nil {
		return explained{}, err
	}
	c := withBudgets(cfg, checks)[0]
	body, ok := builtinDocs[name]
	if !ok || replaced(name) {
		body = orDash(c.desc.Summary) + "\n"
	}
	return explained{Name: name, Summary: c.desc.Summary, Scope: c.desc.Scope, BudgetMs: timeoutFor(c).Milliseconds(),
		Note: c.describeErr, Body: body}, nil
}

// replaced reports whether a local check or a program beside the runner
// takes the place of the built-in check name.
func replaced(name string) bool {
	_, local := localBins[name]
	_, beside := besidePrograms[name]
	return local || beside
}

// suggest names up to three checks closest to name, or points at `fitness
// list` when none is close.
func suggest(name string, catalog []string) string {
	limit := max(2, len(name)/3)
	var near []string
	for _, candidate := range catalog {
		if editDistance(name, candidate) <= limit || strings.Contains(candidate, name) {
			near = append(near, candidate)
		}
	}
	slices.SortStableFunc(near, func(a, b string) int {
		return cmp.Compare(editDistance(name, a), editDistance(name, b))
	})
	if len(near) == 0 {
		return "`fitness list` names them all"
	}
	return "did you mean " + strings.Join(near[:min(3, len(near))], ", ") + "?"
}

// editDistance counts the single-letter inserts, deletes, and changes that
// turn a into b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+change(a[i-1], b[j-1]))
		}
		prev = cur
	}
	return prev[len(b)]
}

// change is the cost of turning letter x into letter y.
func change(x, y byte) int {
	if x == y {
		return 0
	}
	return 1
}
