package runtimes

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	packageMain = regexp.MustCompile(`^package\s+main\b`)
	buildIgnore = regexp.MustCompile(`^//\s*(go:build|\+build)\s+ignore\b`)
)

const goCommand = "Go command"

// goCommand records a Go folder as one command the first time a product
// file in it declares package main. Library packages are source layers,
// and a file built only by hand, under a "go:build ignore" constraint,
// is a generator.
func (f *finder) goCommand(file string) {
	dir := path.Dir(file)
	if f.known(goCommand, dir) {
		return
	}
	lines, ok := f.lines(file)
	if !ok {
		return
	}
	if line := mainLine(lines); line > 0 {
		f.Products = append(f.Products, Product{Kind: goCommand, Name: f.commandName(dir), Technology: "Go", Paths: []string{dir}, At: Location{file, line}})
	}
}

// commandName is the binary go build makes from dir: the folder's name, or
// for the repository root the last element of the go.mod module path,
// else the root folder's name.
func (f *finder) commandName(dir string) string {
	if dir != "." {
		return path.Base(dir)
	}
	if module := f.module(); module != "" {
		return path.Base(module)
	}
	return filepath.Base(f.root)
}

// module is the root go.mod's module path, or "" without one.
func (f *finder) module() string {
	if !f.exists["go.mod"] {
		return ""
	}
	lines, _ := f.lines("go.mod")
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

// known reports whether a product of kind is already built from dir.
func (f *finder) known(kind, dir string) bool {
	for _, p := range f.Products {
		if p.Kind == kind && p.Paths[0] == dir {
			return true
		}
	}
	return false
}
