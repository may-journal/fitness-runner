package check

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// docsDir holds the requirement docs, one per file.
const docsDir = "docs/requirements/"

// numberedDoc matches a requirement doc file: 0014-kebab-title.md.
var numberedDoc = regexp.MustCompile(`^(\d{4})-.+\.md$`)

// docNumber returns the requirement number of a doc path, or "".
func docNumber(f string) string {
	if !strings.HasPrefix(f, docsDir) {
		return ""
	}
	if m := numberedDoc.FindStringSubmatch(path.Base(f)); m != nil {
		return m[1]
	}
	return ""
}

// requirementDocs lists the repo's requirement docs.
func requirementDocs(root string) []string {
	var docs []string
	for _, f := range walkfs.FilesByExt(root, ".md") {
		if docNumber(f) != "" {
			docs = append(docs, f)
		}
	}
	return docs
}

// productExts are the endings of product code: Go, Swift, and shell scripts.
var productExts = []string{".go", ".swift", ".sh"}

// notProductDirs are folders whose files are vendored, fixtures, or build
// output, never product code.
var notProductDirs = []string{"vendor", "third_party", "Pods", "Carthage", "node_modules", "testdata", "dist", ".build", "DerivedData"}

// generatedGo matches the line Go's convention puts in a generated file.
var generatedGo = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

// testSet is what the language modules know of the repo's tests.
// It holds the files that declare tests, and the test files that prove each
// requirement number.
type testSet struct {
	files   map[string]bool
	numbers map[string][]string
}

// scanTests reads every language module's tests, as the requirements check
// pairs them.
func scanTests(root string) (testSet, error) {
	s := testSet{files: map[string]bool{}, numbers: map[string][]string{}}
	for _, m := range lang.Testing(langs.All) {
		tests, files, _, err := m.Tests.Scan(root)
		if err != nil {
			return s, err
		}
		for _, f := range files {
			s.files[f] = true
			s.prove(f, lang.TestFileNumbers(langs.All, f)...)
		}
		for _, t := range tests {
			s.prove(t.File, idNumber(t.ID)...)
		}
	}
	return s, nil
}

// idNumber is the requirement number an acceptance ID carries, if any.
func idNumber(id string) []string {
	if len(id) < 4 {
		return nil
	}
	return []string{id[:4]}
}

// prove records that file proves each of numbers.
func (s testSet) prove(file string, numbers ...string) {
	for _, n := range numbers {
		if !slices.Contains(s.numbers[n], file) {
			s.numbers[n] = append(s.numbers[n], file)
		}
	}
}

// isTest reports whether f is a test.
// A test is a file a module finds tests in, or a Go test file.
// A Swift file in a test target's `…Tests` folder also counts.
// It still counts once deleted.
func (s testSet) isTest(f string) bool {
	return s.files[f] || lang.IsTestSource(langs.All, f) || swiftTestTarget(f)
}

// swiftTestTarget reports whether a Swift file sits in a folder, or has a
// name, that ends in Tests, as Xcode names test targets.
func swiftTestTarget(f string) bool {
	if path.Ext(f) != ".swift" {
		return false
	}
	return slices.ContainsFunc(strings.Split(strings.TrimSuffix(f, ".swift"), "/"), func(part string) bool {
		return strings.HasSuffix(part, "Tests")
	})
}

// anyTest reports whether a test changed.
func (s testSet) anyTest(changed map[string]bool) bool {
	for f := range changed {
		if s.isTest(f) {
			return true
		}
	}
	return false
}

// productChanges keeps the changes to product code: Go, Swift, and shell
// files that are not tests, vendored, generated, or build output.
func productChanges(root string, tests testSet, changes []change) []change {
	var candidates []string
	for _, c := range changes {
		if isProductPath(c.path) && !tests.isTest(c.path) && !isGeneratedGo(root, c.path) {
			candidates = append(candidates, c.path)
		}
	}
	kept := walkfs.WithoutGenerated(root, candidates)
	build := lang.BuildOutput(langs.All, root)
	return slices.DeleteFunc(slices.Clone(changes), func(c change) bool {
		return !slices.Contains(kept, c.path) || slices.Contains(build, c.path)
	})
}

// isProductPath reports whether f has a product ending outside every
// vendored, fixture, and build output folder.
func isProductPath(f string) bool {
	if !slices.Contains(productExts, path.Ext(f)) {
		return false
	}
	return !slices.ContainsFunc(strings.Split(path.Dir(f), "/"), func(dir string) bool {
		return slices.Contains(notProductDirs, dir)
	})
}

// isGeneratedGo reports whether a Go file on disk says it is generated.
func isGeneratedGo(root, f string) bool {
	if path.Ext(f) != ".go" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
	return err == nil && generatedGo.Match(data)
}
