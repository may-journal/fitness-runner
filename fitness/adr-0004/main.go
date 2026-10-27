// Command adr-0004 keeps ADR 0004: each requirement doc names one area, and
// no two docs name the same one.
package main

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// doc matches a requirement doc's name and takes its area.
var doc = regexp.MustCompile(`^docs/requirements/\d{4}-(.+)\.md$`)

// kinds are the area prefixes this repo names its docs with.
var kinds = []string{"checks-", "engine-", "hooks-", "github-"}

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	owners := map[string]string{}
	var found []check.Finding
	for _, f := range ctx.Files(".md") {
		m := doc.FindStringSubmatch(f)
		if m == nil {
			continue
		}
		found = append(found, area(f, m[1], owners)...)
	}
	return check.Found(len(owners), found...), nil
}

// area judges one doc's area against its kind and the docs before it.
func area(f, name string, owners map[string]string) []check.Finding {
	var found []check.Finding
	if !slices.ContainsFunc(kinds, func(k string) bool { return strings.HasPrefix(name, k) }) {
		found = append(found, check.Finding{File: f, Rule: "area-kind", Message: "names no kind of area (ADR 0004)",
			Fix: "start its area with checks-, engine-, hooks- or github-"})
	}
	if prior, taken := owners[name]; taken {
		found = append(found, check.Finding{File: f, Rule: "one-doc-per-area", Message: "covers the area of " + path.Base(prior) + " (ADR 0004, decision 5)",
			Fix: "fold the newer doc into the older one"})
	}
	owners[name] = f
	return found
}
