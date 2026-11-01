package tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/views"
)

// Requirements finds Java requirement tests, methods in a test source set that
// the frameworks folder's annotations mark, read from tokens.
var Requirements = lang.Tests{Scan: scan, Name: naming.Name, Source: java.Fixture, Reads: reads}

// naming names Java requirement tests: test0001_1 owns 0001.1.
const naming = lang.TestPrefix("test")

// scan reads the tests of every Java source in a test source set's folder.
func scan(root string) ([]lang.Test, []string, bool, error) {
	var tests []lang.Test
	var files []string
	for _, f := range java.Sources(root) {
		if !java.Fixture(f) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, nil, false, err
		}
		if found := fileTests(f, string(body)); len(found) > 0 {
			tests, files = append(tests, found...), append(files, f)
		}
	}
	return tests, files, java.Applies(root), nil
}

// fileTests lists the test methods one source declares.
func fileTests(file, src string) []lang.Test {
	var out []lang.Test
	for _, m := range views.Methods(src) {
		if runs(m) {
			out = append(out, lang.Test{Name: m.Name, ID: naming.ID(m.Name), Loc: fmt.Sprintf("%s:%d", file, m.Line), File: file})
		}
	}
	return out
}

// runs reports a method a framework runs as a test, by its own annotation
// or by the annotation of its class.
func runs(m views.Method) bool {
	if marked(m.Annotations, func(f framework) []string { return f.tests }) {
		return true
	}
	return m.Public && !m.Constructor && len(m.Annotations) == 0 && marked(m.Type, func(f framework) []string { return f.classes })
}

// marked reports whether any annotation is one a framework names.
func marked(annotations []string, names func(framework) []string) bool {
	return slices.ContainsFunc(frameworks(), func(f framework) bool {
		return slices.ContainsFunc(names(f), func(name string) bool { return slices.Contains(annotations, name) })
	})
}

// reads reports a file Scan reads: a Java test source or a build file.
func reads(file string) bool {
	return java.Fixture(file) && path.Ext(file) == java.Ext || slices.Contains(java.Manifests(), path.Base(file))
}
