// Package golang is the Go language module's home: how a Go repo proves a
// runtime, names its requirement tests and sets the Go checks' options.
// The toolchain package beside it finds modules and runs the go command
// for the Go checks, and the coverage package measures entry coverage.
package golang

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Technology is the name a Go product's container must give its language.
const Technology = "Go"

// Ext is the Go source file ending.
const Ext = ".go"

// Manifest is the file that marks a Go module.
const Manifest = "go.mod"

// Sum is the checksum file the go command writes beside a module's
// manifest.
const Sum = "go.sum"

var (
	packageMain = regexp.MustCompile(`^package\s+main\b`)
	buildIgnore = regexp.MustCompile(`^//\s*(go:build|\+build)\s+ignore\b`)
)

const goCommand = "Go command"

// Runtime reads Go product sources for commands: a product .go file
// declaring package main without a go:build ignore constraint proves a Go
// command, one per folder, whose container names Go.
var Runtime = runtimes.Reader{Sources: map[string]func(*runtimes.Finder, string){Ext: command}}

// command records a Go folder as one command the first time a product
// file in it declares package main. Library packages are source layers,
// and a file built only by hand, under a "go:build ignore" constraint,
// is a generator.
func command(f *runtimes.Finder, file string) {
	dir := path.Dir(file)
	if f.Known(goCommand, dir) {
		return
	}
	lines, ok := f.Lines(file)
	if !ok {
		return
	}
	if line := mainLine(lines); line > 0 {
		f.Products = append(f.Products, runtimes.Product{Kind: goCommand, Name: commandName(f, dir), Technology: Technology, Paths: []string{dir}, At: runtimes.Location{Path: file, Line: line}})
	}
}

// commandName is the binary go build makes from dir: the folder's name, or
// for the repository root the last element of the go.mod module path,
// else the root folder's name.
func commandName(f *runtimes.Finder, dir string) string {
	if dir != "." {
		return path.Base(dir)
	}
	if module := module(f); module != "" {
		return path.Base(module)
	}
	return filepath.Base(f.Root())
}

// module is the root go.mod's module path, or "" without one.
func module(f *runtimes.Finder) string {
	if !f.Exists(Manifest) {
		return ""
	}
	lines, _ := f.Lines(Manifest)
	for _, line := range lines {
		if module, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.Trim(strings.TrimSpace(module), `"`)
		}
	}
	return ""
}

// mainLine is the line declaring package main, or 0 when the file is not
// a command or carries an ignore constraint before its package clause.
func mainLine(lines []string) int {
	for i, line := range lines {
		if buildIgnore.MatchString(line) {
			return 0
		}
		if packageMain.MatchString(line) {
			return i + 1
		}
	}
	return 0
}
