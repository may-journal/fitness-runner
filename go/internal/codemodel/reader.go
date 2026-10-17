package codemodel

import (
	"path"
	"slices"
	"strings"
)

// Repo is what a reader reads: the tracked files, the sources the views
// sort, and their lines.
type Repo struct {
	// Files lists every tracked file, sorted.
	Files []string
	// Code lists the sources that are not tests, in path order.
	Code []string
	// Product lists the product sources, in path order.
	Product []string
	// Read returns a tracked file's lines; nil when it cannot be read.
	Read func(file string) []string
}

// Reader reads one language's sources for the architecture views.
type Reader struct {
	// Language names the language in messages, such as "unsupported
	// evidence", and scopes the names its nodes answer to.
	Language string
	// Exts are the source endings the reader reads.
	Exts []string
	// Read adds the declarations of the repo's sources that are not tests
	// to x, and returns how to relate them.
	Read func(repo Repo, x *Index) Code
	// References, when set, lists each line of a product source that
	// reaches another path, so the components view can prove its edges,
	// with an error per source it cannot read.
	References func(repo Repo) ([]Reference, []string)
	// Strip, set with References, returns a file's lines with comments
	// removed, so a word in a comment proves nothing.
	Strip func(lines []string) []string
}

// Code is what a reader keeps from reading declarations, to relate them.
type Code struct {
	// Relate observes the relations between the graph's nodes in the
	// reader's files.
	Relate func(g *Graph)
	// Realizes, when set, decides whether a type realizes an interface,
	// for a language whose types satisfy interfaces without stating them.
	Realizes func(t, i *Decl) bool
}

// Reads reports whether the reader reads file.
func (r Reader) Reads(file string) bool {
	return slices.Contains(r.Exts, path.Ext(file))
}

// Of is the reader of file among readers, and whether there is one.
func Of(readers []Reader, file string) (Reader, bool) {
	for _, r := range readers {
		if r.Reads(file) {
			return r, true
		}
	}
	return Reader{}, false
}

// Series joins the readers' languages for a message, such as "A, B, and C"
// with "and".
func Series(readers []Reader, conjunction string) string {
	names := make([]string, len(readers))
	for i, r := range readers {
		names[i] = r.Language
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " " + conjunction + " " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", " + conjunction + " " + names[len(names)-1]
}

// FilesWith keeps the files with one of the endings, in order.
func FilesWith(files []string, exts ...string) []string {
	var kept []string
	for _, f := range files {
		if slices.Contains(exts, path.Ext(f)) {
			kept = append(kept, f)
		}
	}
	return kept
}
