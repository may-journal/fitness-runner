package swift

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// library is the module Swift requirement tests drive the app through,
// and libraryRepo the home of its usage guide.
const (
	library     = "UITestingLibrary"
	libraryRepo = "https://github.com/may-journal/apple-ui-testing-library"
)

// systemLibrary is the library's add-on for system UI, which only an
// opt-in end-to-end test target links. Its SystemSession runs on XCTest,
// so a test that imports it may import XCTest and boot a simulator too.
const systemLibrary = "UITestingLibrarySystem"

// hosted is how every failure says to write the test instead.
const hosted = "write it with Swift Testing and " + library + ", hosted in the app, as " + libraryRepo + " shows"

// heavyFrameworks are UI test frameworks a requirement test may not
// import, since apple-ui-testing-library does their job in the app's own
// process.
var heavyFrameworks = map[string]bool{
	"ViewInspector": true, "SnapshotTesting": true, "iOSSnapshotTestCase": true, "FBSnapshotTestCase": true,
	"KIF": true, "EarlGrey": true, "Hammer": true,
}

var (
	// swiftImport matches an import declaration, attributes and a symbol
	// kind allowed; group 1 is the top module.
	swiftImport = regexp.MustCompile(`(?m)^[ \t]*(?:@\w+[ \t]+)*import[ \t]+(?:(?:class|struct|enum|protocol|typealias|func|var|let)[ \t]+)?(\w+)`)
	// xcuiType names an Xcode UI test type, such as XCUIApplication.
	xcuiType = regexp.MustCompile(`\bXCUI[A-Z]\w*`)
	// simctl matches a string literal that runs the simulator control
	// tool: "simctl" alone, as a process argument, or a simctl command.
	simctl = regexp.MustCompile(`"simctl"|\bsimctl\s+(?:boot|shutdown|erase|create|clone|delete|install|uninstall|launch|terminate|spawn|privacy|push|io|list)\b`)
)

// tools judges each Swift test file's imports and APIs. A requirement test
// runs on Swift Testing with UITestingLibrary in the app's process, never
// on XCTest, Xcode UI tests, a simulator, or another UI test framework.
// Only a file that imports UITestingLibrarySystem, the opt-in end-to-end
// mode, may use XCTest and simctl; it drives the app through that
// library's SystemSession, never XCUIApplication itself.
func tools(root string, files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	importers, err := libraryImporters(root, walkfs.FilesByExt(root, Ext))
	if err != nil {
		return nil, err
	}
	var errs []string
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, err
		}
		errs = append(errs, fileTools(f, string(data), reaches(f, importers))...)
	}
	return errs, nil
}

// fileTools judges one test file. reached reports whether a Swift file in
// the test's folder or below, such as a shared helper, imports the library.
func fileTools(file, src string, reached bool) []string {
	code := blankSwift(src)
	imports := importLines(code)
	_, e2e := imports[systemLibrary]
	errs := importErrors(file, code, imports, e2e)
	errs = append(errs, callErrors(file, src, code, e2e)...)
	if _, own := imports[library]; !own && !e2e && !reached {
		errs = append(errs, fmt.Sprintf("%s: nothing in %s imports %s; %s", file, folderOf(file), library, hosted))
	}
	return errs
}

// importErrors fails each import a requirement test may not use, top to
// bottom.
func importErrors(file, code string, imports map[string]int, e2e bool) []string {
	var errs []string
	for _, name := range sortedNames(imports) {
		if msg := importError(name, e2e); msg != "" {
			errs = append(errs, at(file, code, imports[name], msg))
		}
	}
	return errs
}

// callErrors fails the first Xcode UI test type, and outside an end-to-end
// test the first XCTestCase subclass and the first simctl run, which a
// string literal names.
func callErrors(file, src, code string, e2e bool) []string {
	var errs []string
	if m := xcuiType.FindStringIndex(code); m != nil {
		errs = append(errs, at(file, code, m[0], "drives Xcode UI tests through "+code[m[0]:m[1]]+", which launches the app in a separate process; "+hosted))
	}
	if e2e {
		return errs
	}
	if offset := xcTestCaseClass(code); offset >= 0 {
		errs = append(errs, at(file, code, offset, "subclasses XCTestCase; "+hosted))
	}
	if m := simctl.FindStringIndex(blankComments(src)); m != nil {
		errs = append(errs, at(file, code, m[0], "boots a simulator through simctl outside the end-to-end mode; "+hosted))
	}
	return errs
}

// xcTestCaseClass is the offset of the first class inheriting XCTestCase,
// or -1.
func xcTestCaseClass(code string) int {
	for _, m := range swiftClass.FindAllStringSubmatchIndex(code, -1) {
		if xcTestCase.MatchString(code[m[4]:m[5]]) {
			return m[0]
		}
	}
	return -1
}

// importError is the failure for importing module, or "" when a
// requirement test may import it. An end-to-end test may import XCTest.
func importError(module string, e2e bool) string {
	switch {
	case module == "XCTest" && !e2e:
		return "imports XCTest; only an end-to-end test that imports " + systemLibrary + " may use it, so " + hosted
	case heavyFrameworks[module]:
		return "imports " + module + ", a UI test framework requirement tests may not use; " + hosted
	}
	return ""
}

// importLines maps each module blanked code imports to the offset of its
// first import.
func importLines(code string) map[string]int {
	out := map[string]int{}
	for _, m := range swiftImport.FindAllStringSubmatchIndex(code, -1) {
		name := code[m[2]:m[3]]
		if _, seen := out[name]; !seen {
			out[name] = m[2]
		}
	}
	return out
}

// libraryImporters lists the Swift files that import the library.
func libraryImporters(root string, sources []string) ([]string, error) {
	var out []string
	for _, f := range sources {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, err
		}
		if _, ok := importLines(blankSwift(string(data)))[library]; ok {
			out = append(out, f)
		}
	}
	return out, nil
}

// reaches reports whether a library importer sits in the test file's own
// folder or below it, as a shared helper does. A test at the root reaches
// the library only through its own import.
func reaches(file string, importers []string) bool {
	dir := path.Dir(file)
	return dir != "." && slices.ContainsFunc(importers, func(f string) bool { return strings.HasPrefix(f, dir+"/") })
}

// folderOf names a file's folder as the failure shows it.
func folderOf(file string) string {
	return path.Dir(file) + "/"
}

// at formats a failure at the line holding offset.
func at(file, code string, offset int, msg string) string {
	return fmt.Sprintf("%s:%d: %s", file, strings.Count(code[:offset], "\n")+1, msg)
}

// sortedNames lists a map's keys by the offset they map to, so failures
// read top to bottom.
func sortedNames(offsets map[string]int) []string {
	names := make([]string, 0, len(offsets))
	for n := range offsets {
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int { return offsets[a] - offsets[b] })
	return names
}
