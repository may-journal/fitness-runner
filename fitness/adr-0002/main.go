// Command adr-0002 keeps ADR 0002: no code, config or hook names a second
// lint path beside the eslint check.
package main

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// second is the retired lint command, split so this file never names it.
var second = "fitness-shared" + " lint"

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	files := ctx.Files(".go", ".json", ".yml", ".yaml", ".js", ".mjs", ".cjs", ".ts", ".sh")
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		if strings.Contains(text, second) {
			found = append(found, check.Finding{File: f, Rule: "one-lint-path", Message: "names a second lint path, " + second + " (ADR 0002)",
				Fix: "lint through the eslint check and its one config"})
		}
	}
	return check.Found(len(files), found...), nil
}
