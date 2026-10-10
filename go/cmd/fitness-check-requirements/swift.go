package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// swiftFunc matches a function declaration; group 2 is set when it takes
	// no parameters, as XCTest requires.
	swiftFunc = regexp.MustCompile(`\bfunc\s+(\w+)\s*\(\s*(\))?`)
	// swiftClass matches a class declaration with an inheritance clause, up to
	// the brace that opens its body.
	swiftClass = regexp.MustCompile(`\bclass\s+([A-Za-z_]\w*)(?:\s*<[^{]*>)?\s*:([^{;]*)\{`)
	// xcTestCase names the XCTest base class in an inheritance clause.
	xcTestCase = regexp.MustCompile(`\bXCTestCase\b`)
	// swiftTestMark is Swift Testing's @Test attribute.
	swiftTestMark = regexp.MustCompile(`@(?:Testing\.)?Test\b`)
	// swiftTestID matches a Swift test named for an acceptance: test0001_1 is
	// 0001.1.
	swiftTestID = regexp.MustCompile(`^test(\d{4})_([1-9]\d*)$`)
)

// scanSwift reads every Swift file and returns its tests: XCTest methods named
// test… in a class that inherits XCTestCase, and functions marked @Test.
// Comments and string literals are blanked first, so a name inside them is
// never a test.
func scanSwift(root string, files []string) ([]goTest, error) {
	var tests []goTest
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, err
		}
		tests = append(tests, swiftFileTests(f, blankSwift(string(data)))...)
	}
	return tests, nil
}

// swiftFileTests returns the tests declared in one file's blanked code.
func swiftFileTests(file, code string) []goTest {
	funcs := swiftFunc.FindAllStringSubmatchIndex(code, -1)
	starts := make([]int, len(funcs))
	for i, m := range funcs {
		starts[i] = m[0]
	}
	bodies := xcTestBodies(code)
	var tests []goTest
	for i, owner := range enclosing(code, starts) {
		m := funcs[i]
		name := code[m[2]:m[3]]
		if isSwiftTest(code, m, bodies[owner] && strings.HasPrefix(name, "test")) {
			loc := fmt.Sprintf("%s:%d", file, strings.Count(code[:m[0]], "\n")+1)
			tests = append(tests, goTest{name: name, id: swiftIDOf(name), loc: loc, file: file, swift: true})
		}
	}
	return tests
}

// isSwiftTest reports whether the function at m runs as a test: marked @Test,
// or an XCTest method that takes no parameters.
func isSwiftTest(code string, m []int, xcMethod bool) bool {
	head := code[strings.LastIndexAny(code[:m[0]], "{};")+1 : m[0]]
	return swiftTestMark.MatchString(head) || xcMethod && m[4] >= 0
}

// xcTestBodies marks the offset of each brace that opens a class inheriting
// XCTestCase.
func xcTestBodies(code string) map[int]bool {
	bodies := map[int]bool{}
	for _, m := range swiftClass.FindAllStringSubmatchIndex(code, -1) {
		if !swiftKeywords[code[m[2]:m[3]]] && xcTestCase.MatchString(code[m[4]:m[5]]) {
			bodies[m[1]-1] = true
		}
	}
	return bodies
}

// swiftKeywords follow `class` as a modifier, not a class name.
var swiftKeywords = map[string]bool{"func": true, "var": true, "let": true, "subscript": true}

// enclosing returns, for each ascending offset in at, the offset of the
// innermost brace open there, or -1 at the top level.
func enclosing(code string, at []int) []int {
	stack := []int{-1}
	out := make([]int, 0, len(at))
	for i := 0; i < len(code) && len(out) < len(at); i++ {
		if i == at[len(out)] {
			out = append(out, stack[len(stack)-1])
		}
		stack = braces(stack, code[i], i)
	}
	return out
}

// braces pushes an opening brace's offset and pops on a closing one.
func braces(stack []int, c byte, i int) []int {
	switch {
	case c == '{':
		return append(stack, i)
	case c == '}' && len(stack) > 1:
		return stack[:len(stack)-1]
	}
	return stack
}

// swiftIDOf returns the acceptance ID a Swift test name carries, or "".
func swiftIDOf(name string) string {
	m := swiftTestID.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}

// swiftTestFiles lists the files that declare at least one Swift test.
func swiftTestFiles(tests []goTest) []string {
	seen := map[string]bool{}
	for _, t := range tests {
		seen[t.file] = true
	}
	return sortedKeys(seen)
}
