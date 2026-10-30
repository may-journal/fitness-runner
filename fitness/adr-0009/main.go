// Command adr-0009 keeps ADR 0009 in fitness itself: org policy, no check
// turned off, and no local check that replaces a bundled one.
package main

import (
	"path"

	"github.com/may-journal/fitness-runner/go/check"
)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var settings struct {
		Policy         string   `json:"policy"`
		DisabledChecks []string `json:"disabledChecks"`
	}
	if _, err := ctx.Settings(&settings); err != nil {
		return check.Result{}, err
	}
	var found []check.Finding
	if settings.Policy == "external" || len(settings.DisabledChecks) > 0 {
		found = append(found, check.Finding{File: ".fitnessrc.json", Rule: "org-bar", Message: "sets external policy or disabledChecks (ADR 0009, decisions 5 and 8)",
			Fix: "keep org policy and fix the findings instead"})
	}
	readmes := folderReadmes(ctx)
	return check.Found(len(readmes)+1, append(found, replacements(ctx, readmes)...)...), nil
}

// replacements fails each check in fitness/ named like a bundled check.
func replacements(ctx *check.Context, readmes []string) []check.Finding {
	var found []check.Finding
	for _, f := range readmes {
		if name := path.Base(path.Dir(f)); ctx.Exists("go/cmd/fitness-check-" + name) {
			found = append(found, check.Finding{File: f, Rule: "no-replacement", Message: name + " shares a bundled check's name (ADR 0009, decision 5)",
				Fix: "give the check a name of its own"})
		}
	}
	return found
}

// folderReadmes lists the README of each check in fitness/.
func folderReadmes(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files(".md") {
		if ok, _ := path.Match("fitness/*/README.md", f); ok {
			out = append(out, f)
		}
	}
	return out
}
