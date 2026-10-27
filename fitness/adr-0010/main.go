// Command adr-0010 keeps ADR 0010: every built-in check is a package the
// one fitness binary runs, and its name list is generated.
package main

import (
	"path"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	exports, err := packages(ctx)
	if err != nil {
		return check.Result{}, err
	}
	found := missingMain(exports)
	registry, err := ctx.Read("go/cmd/fitness/checks.go")
	if err != nil {
		return check.Result{}, err
	}
	if !strings.HasPrefix(registry, "// Code generated") {
		found = append(found, check.Finding{File: "go/cmd/fitness/checks.go", Rule: "generated-registry", Message: "is no longer generated (ADR 0010, decision 3)",
			Fix: "run go -C go generate ./cmd/fitness"})
	}
	return check.Found(len(exports)+1, found...), nil
}

// packages maps each built-in check package to whether it exports Main.
func packages(ctx *check.Context) (map[string]bool, error) {
	exports := map[string]bool{}
	for _, f := range ctx.Files(".go") {
		if !strings.HasPrefix(f, "go/cmd/fitness-check-") {
			continue
		}
		text, err := ctx.Read(f)
		if err != nil {
			return nil, err
		}
		exports[path.Dir(f)] = exports[path.Dir(f)] || strings.Contains(text, "\nfunc Main()")
	}
	return exports, nil
}

// missingMain fails each check package that exports no Main.
func missingMain(exports map[string]bool) []check.Finding {
	var found []check.Finding
	for dir, ok := range exports {
		if !ok {
			found = append(found, check.Finding{File: dir, Rule: "exports-main", Message: "exports no func Main() (ADR 0010, decision 1)",
				Fix: "export Main, so the one fitness binary runs the check"})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].File < found[j].File })
	return found
}
