// Command adr-0009 keeps ADR 0009 in fitness itself: org policy, with no
// check turned off.
package main

import (
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
	return check.Found(1, found...), nil
}
