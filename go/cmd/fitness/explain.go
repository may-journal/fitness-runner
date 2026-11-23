package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/pkg/conf"
	"github.com/may-journal/fitness-runner/go/pkg/repocheck"
)

// explained is one check as `fitness explain --json` gives it.
type explained struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Scope   string `json:"scope"`
	// Stage is the stage whose total times the check: fast or tests.
	Stage string `json:"stage"`
	docFields
	// Note says why the check's describe handshake gave no metadata.
	Note string `json:"note,omitempty"`
	// Body is the check's documentation as Markdown.
	Body string `json:"body"`
	// Config lists the .fitnessrc.json keys the check declares it reads.
	Config []conf.Key `json:"config,omitempty"`
}

// configSection lists the keys a check declares as Markdown, one line per
// leaf key, or "" for none.
func configSection(keys []conf.Key) string {
	lines := configLines(keys)
	if len(lines) == 0 {
		return ""
	}
	return "\n## Config keys\n\n" + strings.Join(lines, "\n") + "\n"
}

// configLines is one list item per leaf key under keys, in order.
func configLines(keys []conf.Key) []string {
	var lines []string
	for _, k := range keys {
		if k.Keys != nil {
			lines = append(lines, configLines(k.Keys)...)
			continue
		}
		line := fmt.Sprintf("- `%s` (%s", k.Path, k.Type)
		if k.Default != nil {
			value, _ := json.Marshal(k.Default) // it came from JSON
			line += ", default " + string(value)
		}
		lines = append(lines, line+"): "+k.Description)
	}
	return lines
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
	if err := repoConfig(); err != nil {
		report.Error("fitness explain", err)
		return 1
	}
	if catalog := knownChecks(); !slices.Contains(catalog, name) {
		fmt.Fprintf(os.Stderr, "fitness explain: no check named %q; %s\n", name, suggest(name, catalog))
		return 2
	}
	e, err := explainCheck(name)
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
	fmt.Printf("# %s\n\nScope: %s · Stage: %s\n\n%s%s%s%s", e.Name, orDash(e.Scope), e.Stage, e.line(), noteLine(e.Note), e.Body, configSection(e.Config))
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
// explains itself too. A config or local check folder that cannot load is an
// error, as is an install with no checks.
func repoConfig() error {
	if installErr != nil {
		return installErr
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}
	return loadLocalChecks(cfg, root)
}

// knownChecks are the names explain answers for: the repo's catalog and every
// local-branch, even when a language ignore file skips it.
func knownChecks() []string {
	names := checkCatalog()
	if !slices.Contains(names, localBranch) {
		names = append(names, localBranch)
	}
	return names
}

// explainCheck describes the check from its README, its Markdown in go/docs, or
// handshake summary. A failed handshake or an unmet minFitness returns a note.
func explainCheck(name string) (explained, error) {
	checks, err := resolveList([]string{name})
	if err != nil {
		return explained{}, err
	}
	c := checks[0]
	body := localBranchDoc.Prose()
	if name != localBranch || replaced(name) {
		body = cmp.Or(repocheck.Doc(localBins[name]), orDash(c.desc.Summary)+"\n")
	}
	return explained{Name: name, Summary: c.desc.Summary, Scope: c.desc.Scope, Stage: stageOf(c), docFields: effectiveFields(c.desc),
		Note: cmp.Or(c.needs, c.describeErr), Body: body, Config: c.desc.Config}, nil
}

// replaced reports whether a local check or an installed program takes the
// place of the built-in check name.
func replaced(name string) bool {
	_, local := localBins[name]
	_, installed := installedPrograms[name]
	return local || installed
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
