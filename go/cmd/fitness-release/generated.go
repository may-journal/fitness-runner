package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	Summary:   "Fails when a file `go generate` writes from doc comments is stale, or a check package lacks a doc comment or a comments treatment.",
	Scope:     checkkit.ScopeFiles,
	Comments:  checkkit.CommentsJudged,
	TimeoutMs: 120000,
}

// checkGenerated runs the generated check with args, as the check protocol
// starts a check, and exits with its verdict.
func checkGenerated(args []string) {
	os.Args = append([]string{checkkit.BinPrefix + generatedCommand}, args...)
	checkkit.Main(checkkit.Check{Describe: generatedDescribe, Run: judgeGenerated})
}

// judgeGenerated fails each generated file differing from what `go generate` writes now, and each program whose describe line drifts.
func judgeGenerated(root string, _ []string) (checkkit.Result, error) {
	if !touchesInputs(checkkit.ChangedFiles()) {
		return checkkit.Pass(0), nil
	}
	config, err := release.New(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	stale, total, err := config.Stale()
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	errs := append(staleErrors(stale), describeDrift(root)...)
	if len(errs) > 0 {
		return checkkit.Fail(total, errs...), nil
	}
	return checkkit.Pass(total), nil
}

// staleErrors names each stale file and how to fix it.
func staleErrors(stale []string) []string {
	errs := make([]string, len(stale))
	for i, path := range stale {
		errs[i] = path + ": stale or edited by hand; run `go -C go generate ./cmd/fitness` and commit the result"
	}
	return errs
}

// touchesInputs reports whether a run's changed files can change what the
// generator writes: any Go file, or any file it writes. Nil changed files
// mean an unscoped run, which judges everything.
func touchesInputs(changed []string) bool {
	if changed == nil {
		return true
	}
	return slices.ContainsFunc(changed, release.Input)
}

// localProgram is the local check that starts this one, relative to the
// repository root.
const localProgram = "fitness/generated/check"

// describeDrift fails the local program when the describe line it answers
// itself says something other than generatedDescribe.
func describeDrift(root string) []string {
	script, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(localProgram)))
	if err != nil {
		return []string{localProgram + ": " + err.Error()}
	}
	_, rest, _ := strings.Cut(string(script), "echo '")
	line, _, _ := strings.Cut(rest, "'")
	var got checkkit.Describe
	if json.Unmarshal([]byte(line), &got) != nil || got != generatedDescribe {
		return []string{localProgram + ": its --describe line differs from generatedDescribe in go/cmd/fitness-release/generated.go; make them match"}
	}
	return nil
}
