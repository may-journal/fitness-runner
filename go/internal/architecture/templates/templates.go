// Package templates ships the five architecture view templates inside the
// binary, so doc-template judges every repo's views against them, whether
// or not the repo keeps a copy of its own.
//
//fitness:container cli
//fitness:component kit Checks share it as one engine, so no check keeps its own copy.
package templates

import (
	"embed"
	"path"
)

//go:embed *.md
var files embed.FS

// roots are the folders that hold architecture views.
var roots = []string{"architecture", "docs/architecture"}

// For returns the shipped template that governs the view at a slash path,
// and its name in messages. ok is false for a path that is no view.
func For(view string) (name, body string, ok bool) {
	dir, base := path.Split(view)
	if !isRoot(path.Clean(dir)) {
		return "", "", false
	}
	b, err := files.ReadFile(base)
	if err != nil {
		return "", "", false
	}
	return "the shipped " + base, string(b), true
}

func isRoot(dir string) bool {
	for _, root := range roots {
		if dir == root {
			return true
		}
	}
	return false
}
