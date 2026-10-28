// Command adr-0001 keeps ADR 0001: each budget a built-in check declares
// names the external cost that needs it, and the repo raises no budget for
// all checks.
package main

import (
	"path"
	"regexp"
	"sort"
	"strconv"

	"github.com/may-journal/fitness-runner/go/check"
)

// budget is one declared budget and the external cost behind it.
type budget struct {
	Ms   int    `json:"ms"`
	Cost string `json:"cost"`
}

// declared matches the budget a check declares in its describe metadata.
var declared = regexp.MustCompile(`TimeoutMs: (\d+)`)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var cfg struct {
		Budgets map[string]budget `json:"budgets"`
	}
	if _, err := ctx.Config(&cfg); err != nil {
		return check.Result{}, err
	}
	var settings struct {
		TimeoutMs *int `json:"timeoutMs"`
	}
	if _, err := ctx.Settings(&settings); err != nil {
		return check.Result{}, err
	}
	found, files, err := budgets(ctx, cfg.Budgets)
	if settings.TimeoutMs != nil {
		found = append(found, check.Finding{File: ".fitnessrc.json", Rule: "no-raised-default",
			Message: "timeoutMs raises the budget of every check (ADR 0001, decision 1)", Fix: "cut the slow work instead"})
	}
	return check.Found(files+1, found...), err
}

// budgets judges each check's declared budget against the recorded ones,
// and each recorded one against its check.
func budgets(ctx *check.Context, recorded map[string]budget) ([]check.Finding, int, error) {
	var found []check.Finding
	seen := map[string]bool{}
	files := globMain(ctx)
	for _, f := range files {
		more, err := judgeBudget(ctx, f, recorded, seen)
		if err != nil {
			return nil, 0, err
		}
		found = append(found, more...)
	}
	return append(found, stale(recorded, seen)...), len(files), nil
}

// judgeBudget fails the budget f declares when config.json records
// another, marking its check seen.
func judgeBudget(ctx *check.Context, f string, recorded map[string]budget, seen map[string]bool) ([]check.Finding, error) {
	name, ms, err := declaredBudget(ctx, f)
	if err != nil || ms == 0 {
		return nil, err
	}
	seen[name] = true
	if recorded[name].Ms == ms {
		return nil, nil
	}
	return []check.Finding{{File: f, Rule: "budget-recorded",
		Message: name + " declares a " + strconv.Itoa(ms) + "ms budget that fitness/adr-0001/config.json does not record (ADR 0001, decision 2)",
		Fix:     "cut the work, or record the external cost that needs it"}}, nil
}

// globMain lists every built-in check's main.go.
func globMain(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files(".go") {
		if ok, _ := path.Match("go/cmd/fitness-check-*/main.go", f); ok {
			out = append(out, f)
		}
	}
	return out
}

// declaredBudget reads a check's name and the budget its main.go declares,
// 0 for none.
func declaredBudget(ctx *check.Context, f string) (string, int, error) {
	text, err := ctx.Read(f)
	if err != nil {
		return "", 0, err
	}
	name := path.Base(path.Dir(f))[len("fitness-check-"):]
	m := declared.FindStringSubmatch(text)
	if m == nil {
		return name, 0, nil
	}
	ms, err := strconv.Atoi(m[1])
	return name, ms, err
}

// stale fails a recorded budget whose check no longer declares one.
func stale(recorded map[string]budget, seen map[string]bool) []check.Finding {
	var found []check.Finding
	for name := range recorded {
		if !seen[name] {
			found = append(found, check.Finding{File: "fitness/adr-0001/config.json", Rule: "budget-recorded",
				Message: name + " no longer declares a budget", Fix: "delete or update its entry"})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Message < found[j].Message })
	return found
}
