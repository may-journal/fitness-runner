// Command adr-0012 keeps ADR 0012: each fitness check lives in a fitness/<name>/
// folder, with no flat check program anywhere else.
package main

import (
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var settings struct {
		LocalChecks string `json:"localChecks"`
	}
	if _, err := ctx.Settings(&settings); err != nil {
		return check.Result{}, err
	}
	var found []check.Finding
	if settings.LocalChecks != "" {
		found = append(found, check.Finding{File: ".fitnessrc.json", Rule: "no-local-checks-folder",
			Message: "names a localChecks folder (ADR 0012, decision 1)", Fix: "move its checks into fitness/ and drop the key"})
	}
	files := ctx.Files()
	for _, f := range files {
		found = append(found, placed(f)...)
	}
	return check.Found(len(files)+1, found...), nil
}

// placed fails a flat check program outside go/cmd, which belongs in a
// fitness/<name>/ folder with its README.
func placed(f string) []check.Finding {
	if !strings.HasPrefix(path.Base(f), "fitness-check-") || strings.HasPrefix(f, "go/cmd/") {
		return nil
	}
	return []check.Finding{{File: f, Rule: "check-in-folder", Message: "is a flat check program (ADR 0012, decisions 1 and 2)",
		Fix: "move it into fitness/<name>/ with a README.md"}}
}
