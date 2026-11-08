// Command adr-0001 keeps ADR 0001: a check joins the test stage only for a
// test run. No setting raises a total.
package main

import (
	"path"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// testStage marks a check that declares the test stage in its describe
// metadata.
const testStage = "Stage: checkkit.StageTests"

// configFile records each test-stage check and the run it waits on.
const configFile = "fitness/adr-0001/config.json"

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var cfg struct {
		Tests map[string]string `json:"tests"`
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
	found, files, err := stages(ctx, cfg.Tests)
	if settings.TimeoutMs != nil {
		found = append(found, check.Finding{File: ".fitnessrc.json", Rule: "no-timeout-setting",
			Message: "timeoutMs sets a limit per check, which the run no longer has (ADR 0001, decision 1)", Fix: "delete it and cut the slow work instead"})
	}
	return check.Found(files+1, found...), err
}

// stages judges each check that joins the test stage against the recorded
// ones, and each recorded one against its check.
func stages(ctx *check.Context, recorded map[string]string) ([]check.Finding, int, error) {
	var found []check.Finding
	seen := map[string]bool{}
	files := globMain(ctx)
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return nil, 0, err
		}
		if !strings.Contains(text, testStage) {
			continue
		}
		name := strings.TrimPrefix(path.Base(path.Dir(f)), "fitness-check-")
		seen[name] = true
		if recorded[name] == "" {
			found = append(found, check.Finding{File: f, Rule: "test-stage-recorded",
				Message: name + " joins the test stage, which " + configFile + " does not record (ADR 0001, decision 2)",
				Fix:     "cut the work so it stays fast, or record the test run it waits on"})
		}
	}
	return append(found, stale(recorded, seen)...), len(files), nil
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

// stale fails a recorded check that no longer joins the test stage.
func stale(recorded map[string]string, seen map[string]bool) []check.Finding {
	var found []check.Finding
	for name := range recorded {
		if !seen[name] {
			found = append(found, check.Finding{File: configFile, Rule: "test-stage-recorded",
				Message: name + " no longer joins the test stage", Fix: "delete its entry"})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Message < found[j].Message })
	return found
}
