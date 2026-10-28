// Command adr-0006 keeps ADR 0006: only the Biome package runs Biome's
// GitHub reporter.
package main

import (
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// flag is the reporter flag, as a Go string literal passes it.
const flag = `"--reporter=github"`

// owner is the one package that may pass it.
const owner = "go/internal/lang/node/biome/"

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	files := sources(ctx)
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		if strings.Contains(text, flag) && !strings.HasPrefix(f, owner) {
			found = append(found, check.Finding{File: f, Rule: "one-biome-reader", Message: "runs Biome's reporter itself (ADR 0006, decision 3)",
				Fix: "read Biome through " + strings.TrimSuffix(owner, "/")})
		}
	}
	return check.Found(len(files), found...), nil
}

// sources lists the Go sources outside tests, doc comments and the
// generated check docs.
func sources(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files(".go") {
		if source(f) {
			out = append(out, f)
		}
	}
	return out
}

// source reports whether f is fitness code, not a test, doc comment or the
// generated check docs.
func source(f string) bool {
	skip := strings.HasSuffix(f, "_test.go") || path.Base(f) == "doc.go" || f == "go/cmd/fitness/checks.go"
	return strings.HasPrefix(f, "go/") && !skip
}
