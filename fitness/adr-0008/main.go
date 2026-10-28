// Command adr-0008 keeps ADR 0008: each language module is one lang.Module
// value in its own file.
package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var found []check.Finding
	files := modules(ctx)
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		if n := strings.Count(text, "= lang.Module{"); n != 1 {
			found = append(found, check.Finding{File: f, Rule: "one-module-per-file", Message: fmt.Sprintf("declares %d language modules (ADR 0008, decision 1)", n),
				Fix: "give each language its own file with its one lang.Module value"})
		}
	}
	return check.Found(len(files), found...), nil
}

// modules lists the files beside the registry in go/internal/lang/langs.
func modules(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files(".go") {
		if path.Dir(f) == "go/internal/lang/langs" && path.Base(f) != "langs.go" && !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	return out
}
