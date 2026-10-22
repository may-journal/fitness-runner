package main

import (
	"os"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/release"
)

// generatedCommand runs the generated check, which this repo's local
// fitness-check-generated program starts.
const generatedCommand = "generated"

// generatedDescribe is the check's describe handshake. The local program
// answers --describe with the same line itself, since compiling this tool
// can outlast the handshake's budget.
var generatedDescribe = checkkit.Describe{
	Name:      "generated",
	Summary:   "Fails when a file `go generate` writes from doc comments is stale, or a check package has no doc comment.",
	Scope:     checkkit.ScopeFiles,
	TimeoutMs: 120000,
}

// checkGenerated runs the generated check with args, as the check protocol
// starts a check, and exits with its verdict.
func checkGenerated(args []string) {
	os.Args = append([]string{checkkit.BinPrefix + generatedCommand}, args...)
	checkkit.Main(checkkit.Check{Describe: generatedDescribe, Run: judgeGenerated})
}

// judgeGenerated fails each generated file whose content differs from what
// `go generate` writes now. Code the generator cannot read, such as a check
// package without a doc comment, fails the check with the reason.
func judgeGenerated(root string, _ []string) (checkkit.Result, error) {
	config, err := release.New(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	stale, total, err := config.Stale()
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	errs := make([]string, len(stale))
	for i, path := range stale {
		errs[i] = path + ": stale or edited by hand; run `go -C go generate ./cmd/fitness` and commit the result"
	}
	if len(errs) > 0 {
		return checkkit.Fail(total, errs...), nil
	}
	return checkkit.Pass(total), nil
}
