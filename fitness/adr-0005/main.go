// Command adr-0005 keeps ADR 0005: the Swift reference parser is plain Go,
// with no dependency, cgo or Swift toolchain.
package main

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// toolchain matches cgo or a Swift toolchain started from Go.
var toolchain = regexp.MustCompile(`import "C"|exec\.Command\("(swift[a-z-]*|xcrun)"`)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	mod, err := ctx.Read("go/go.mod")
	if err != nil {
		return check.Result{}, err
	}
	var found []check.Finding
	if strings.Contains(mod, "\nrequire") {
		found = append(found, check.Finding{File: "go/go.mod", Rule: "no-dependency", Message: "requires a module (ADR 0005, decision 1)",
			Fix: "build the Swift parser with the standard library alone"})
	}
	files := views(ctx)
	more, err := toolchains(ctx, files)
	return check.Found(len(files)+1, append(found, more...)...), err
}

// toolchains fails each view source that uses cgo or a Swift toolchain.
func toolchains(ctx *check.Context, files []string) ([]check.Finding, error) {
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return nil, err
		}
		if toolchain.MatchString(text) {
			found = append(found, check.Finding{File: f, Rule: "no-toolchain", Message: "uses cgo or a Swift toolchain (ADR 0005, decisions 1 and 5)",
				Fix: "parse Swift in Go, so the check runs offline everywhere"})
		}
	}
	return found, nil
}

// views lists the Swift view reader's Go sources.
func views(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files(".go") {
		if strings.HasPrefix(f, "go/internal/lang/swift/views/") {
			out = append(out, f)
		}
	}
	return out
}
