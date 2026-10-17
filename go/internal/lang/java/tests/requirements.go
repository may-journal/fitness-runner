package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/views"
)

// Requirements finds Java requirement tests: a method test0001_1() owns
// 0001.1. A test is a method in a test source set's folder, such as
// src/test/java, that an annotation named Test or ending in Test marks, as
// JUnit and TestNG mark them, @ParameterizedTest included. Methods are
// read from tokens, so a name in a comment or string is no test.
var Requirements = lang.Tests{
	Scan:   scan,
	Name:   func(id string) string { return "test" + strings.Replace(id, ".", "_", 1) },
	Source: java.Fixture,
}

// testID matches a Java test named for an acceptance: test0001_1 is
// 0001.1.
var testID = regexp.MustCompile(`^test(\d{4})_([1-9]\d*)$`)

// scan reads the tests of every Java source in a test source set's
// folder; a repo with a Java project is written in Java.
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
		if marked(m.Annotations) {
			out = append(out, lang.Test{Name: m.Name, ID: idOf(m.Name), Loc: fmt.Sprintf("%s:%d", file, m.Line), File: file})
		}
	}
	return out
}

// marked reports an annotation named Test or ending in Test.
func marked(annotations []string) bool {
	for _, a := range annotations {
		if strings.HasSuffix(a, "Test") {
			return true
		}
	}
	return false
}

// idOf returns the acceptance ID a test name carries, or "".
func idOf(name string) string {
	m := testID.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}
