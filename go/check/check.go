// Package check is the SDK for a repo's own fitness checks, written in Go.
// A check lives in its repo's `fitness/<name>/` folder, with a README.md
// that states its rule and a main.go that calls Main. `fitness check
// <name>` builds it with `fitness/go.mod` into the user cache, and the
// suite, the git hooks and CI run it beside the built-in checks.
//
//	package main
//
//	import "github.com/may-journal/fitness-runner/go/check"
//
//	var Rule = check.Check{TimeoutMs: 10000, Run: func(ctx *check.Context) (check.Result, error) {
//		files := ctx.Files(".swift")
//		var found []check.Finding
//		for _, f := range files {
//			text, err := ctx.Read(f)
//			if err != nil {
//				return check.Result{}, err
//			}
//			if strings.Contains(text, "TODO") {
//				found = append(found, check.Finding{File: f, Rule: "no-todo", Message: "holds a TODO", Fix: "do it or file an issue"})
//			}
//		}
//		return check.Found(len(files), found...), nil
//	}}
//
//	func main() { check.Main(Rule) }
//
// Main speaks the check protocol (docs/check-protocol.md): --root,
// --describe, the JSON result, the exit code, and a panic or an error,
// which report info and never fail a run. Judge runs a check in a unit
// test, on a repo Repo writes, and Test runs it on the repo the test runs
// in. docs/local-checks.md is the guide.
package check

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Check is one check: what --describe says about it, and the function
// that judges a repo.
type Check struct {
	// Name is the check name. Empty takes the program's name, which is the
	// check's folder name when fitness builds it.
	Name string
	// Summary says in one line what the check enforces, for `fitness list`.
	// Empty takes the README's first sentence.
	Summary string
	// TimeoutMs is the check's budget; 0 means the runner default. A check
	// past its budget is stopped and fails.
	TimeoutMs int
	// MinFitness is the oldest fitness release, as major.minor.patch, that
	// runs the check as written. An older fitness stops the run and names
	// the upgrade.
	MinFitness string
	// Run judges the repo the context names. An error means the check could
	// not judge, which reports info and never fails a run.
	Run func(ctx *Context) (Result, error)
}

// Context is what a run gives a check.
type Context struct {
	// Root is the repo root, where the check runs.
	Root string
	// Args are the arguments after the check's name, as in `fitness
	// <name> --fix`.
	Args []string
	// Changed lists the files the run is scoped to, relative to Root: the
	// staged files at commit time, a pull request's changes in CI, or a
	// `--dir` folder's files. Nil means the run judges every file.
	Changed []string
	// Dir is the check's own folder relative to Root, such as
	// fitness/no-todo, or "" when the check runs outside fitness.
	Dir string
}

// DirEnv names the variable through which fitness hands a folder check
// its own folder, relative to the repo root.
const DirEnv = "FITNESS_CHECK_DIR"

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
		Run: func(root string, args []string) (checkkit.Result, error) {
			return c.Run(&Context{Root: root, Args: args, Changed: checkkit.ChangedFiles(), Dir: os.Getenv(DirEnv)})
		},
	}
}

// Files lists the repo's tracked files whose base names end in any of
// exts, such as .swift or _test.go, as slash-separated paths relative to
// Root, sorted. No exts lists every
// tracked file. In a fitness run the list is shared, so a check runs no git
// command of its own.
func (ctx *Context) Files(exts ...string) []string {
	if len(exts) == 0 {
		exts = []string{""}
	}
	return walkfs.FilesByExt(ctx.Root, exts...)
}

// InScope narrows files to the ones the run is scoped to, keeping all of
// them when it judges every file.
func (ctx *Context) InScope(files []string) []string {
	if ctx.Changed == nil {
		return files
	}
	scoped := map[string]bool{}
	for _, f := range ctx.Changed {
		scoped[f] = true
	}
	var out []string
	for _, f := range files {
		if scoped[f] {
			out = append(out, f)
		}
	}
	return out
}

// Read returns the text of a file, given relative to Root.
func (ctx *Context) Read(rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(ctx.Root, filepath.FromSlash(rel)))
	return string(data), err
}

// Exists reports whether a file or folder, given relative to Root, exists.
func (ctx *Context) Exists(rel string) bool {
	_, err := os.Stat(filepath.Join(ctx.Root, filepath.FromSlash(rel)))
	return err == nil
}

// Config decodes the check's own config.json, in its folder, into v. A
// check without one leaves v as it is and returns false.
func (ctx *Context) Config(v any) (bool, error) {
	if ctx.Dir == "" {
		return false, nil
	}
	return decode(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Dir), "config.json"), v)
}

// Settings decodes the repo's .fitnessrc.json, such as its policy, into
// v. A repo without one leaves v as it is and returns false.
func (ctx *Context) Settings(v any) (bool, error) {
	return ctx.ReadJSON(".fitnessrc.json", v)
}

// ReadJSON decodes a JSON file, given relative to Root, into v. A missing
// file leaves v as it is and returns false.
func (ctx *Context) ReadJSON(rel string, v any) (bool, error) {
	return decode(filepath.Join(ctx.Root, filepath.FromSlash(rel)), v)
}

// decode reads the JSON file at path into v, reporting whether it exists.
func decode(path string, v any) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err == nil {
		err = json.Unmarshal(data, v)
	}
	return err == nil, err
}
