// Package check writes a repo's own fitness check in Go. A check lives in
// its repo's `fitness/<name>/` folder, with a README.md that states its
// rule and a main.go that calls Main. Fitness builds it with the folder's
// `fitness/go.mod` into the user cache, runs it in the suite, the git hooks
// and CI, and lists and explains it beside the built-in checks.
//
//	package main
//
//	import "github.com/may-journal/fitness-runner/go/check"
//
//	func main() {
//		check.Main(check.Check{Summary: "No TODO in Swift sources.", Run: run})
//	}
//
// A check speaks the check protocol (docs/check-protocol.md), and Main
// handles all of it: --root, --describe, the JSON result, the exit code,
// and a panic or an error, which report info and never fail a run. Test runs
// a check as a Go test, so `go -C fitness test ./...` judges the repo the
// way `fitness <name>` does. docs/local-checks.md is the guide.
package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Result is a check's verdict: whether it passed, an error line per
// problem, how many files it judged, and info notes that never fail a run.
type Result = checkkit.Result

// Finding is one problem in a shape an agent acts on without parsing
// prose: file, line, rule, message, and fix.
type Finding = checkkit.Finding

// Check is one check: what --describe says about it, and the function that
// judges a repo.
type Check struct {
	// Name is the check name. Empty takes the program's name, which is the
	// check's folder name when fitness builds it.
	Name string
	// Summary says in one line what the check enforces, for `fitness list`.
	// Empty takes the README's first sentence.
	Summary string
	// TimeoutMs is the check's budget; 0 means the runner default.
	TimeoutMs int
	// MinFitness is the oldest fitness release, as major.minor.patch, that
	// runs the check as written. An older fitness stops the run and names
	// the upgrade.
	MinFitness string
	// Run judges the repo at root, given the arguments after the check's
	// name. An error means the check could not judge, which is info.
	Run func(root string, args []string) (Result, error)
}

// Main runs the check as its program, speaking the check protocol, and
// exits.
func Main(c Check) {
	checkkit.Main(c.kit())
}

// kit is the check as the shared protocol code runs it.
func (c Check) kit() checkkit.Check {
	name := c.Name
	if name == "" {
		name = strings.TrimPrefix(filepath.Base(os.Args[0]), checkkit.BinPrefix)
	}
	return checkkit.Check{
		Describe: checkkit.Describe{Name: name, Summary: c.Summary, TimeoutMs: c.TimeoutMs, MinFitness: c.MinFitness},
		Run:      c.Run,
	}
}

// Pass is a passing result that judged files files.
func Pass(files int) Result {
	return checkkit.Pass(files)
}

// Fail is a failing result that judged files files, with one error line per
// problem.
func Fail(files int, errors ...string) Result {
	return checkkit.Fail(files, errors...)
}

// Info is a passing result that carries notes, such as a tool the check
// could not find.
func Info(files int, notes ...string) Result {
	return checkkit.Info(files, notes...)
}

// Files lists the repo's tracked files under root whose names end in any of
// exts, as slash-separated paths relative to root, sorted. No exts lists
// every tracked file. In a fitness run the list is shared, so a check runs
// no git command of its own.
func Files(root string, exts ...string) []string {
	if len(exts) == 0 {
		exts = []string{""}
	}
	return walkfs.FilesByExt(root, exts...)
}

// ChangedFiles lists the files the run is scoped to: the staged files at
// commit time, a pull request's changes in CI, or a `--dir` folder's files.
// Nil means the run judges every file.
func ChangedFiles() []string {
	return checkkit.ChangedFiles()
}

// Test runs the check on the repo the test runs in, as a Go test: each
// error line fails the test, and each info note is logged. So `go test`
// in the fitness folder judges the repo as `fitness <name>` does.
func Test(t testing.TB, c Check) {
	t.Helper()
	root := repoRoot(t)
	result, err := c.Run(root, nil)
	if err != nil {
		t.Fatalf("the check could not judge %s: %v", root, err)
	}
	for _, note := range result.Info {
		t.Log(note)
	}
	report(t, result)
}

// report fails t with each of the result's error lines, or with a note when
// it failed without one.
func report(t testing.TB, result Result) {
	t.Helper()
	for _, line := range result.Errors {
		t.Error(line)
	}
	if !result.Ok && len(result.Errors) == 0 {
		t.Error("the check failed without an error line")
	}
}

// repoRoot is the root of the git work tree the test runs in.
func repoRoot(t testing.TB) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("a check's test runs inside the repo it judges: git rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}
