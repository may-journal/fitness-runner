package runtimes

import (
	"path"
	"regexp"
)

var packageMain = regexp.MustCompile(`^package\s+main\b`)

const goCommand = "Go command"

// goCommand records a Go folder as one command the first time a product
// file in it declares package main. Library packages are source layers.
func (f *finder) goCommand(file string) {
	dir := path.Dir(file)
	if f.known(goCommand, dir) {
		return
	}
	lines, ok := f.lines(file)
	if !ok {
		return
	}
	for i, line := range lines {
		if packageMain.MatchString(line) {
			f.Products = append(f.Products, Product{Kind: goCommand, Name: path.Base(dir), Technology: "Go", Paths: []string{dir}, At: Location{file, i + 1}})
			return
		}
	}
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
