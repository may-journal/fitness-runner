package check

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// leftBehind fails deleting a file a requirement doc names when neither the
// doc nor its tests changed with it.
func leftBehind(root, file string, tests testSet, changed map[string]bool) []string {
	docs := namingDocs(root, file)
	proving := provingTests(docs, tests)
	if len(docs) == 0 || slices.ContainsFunc(slices.Concat(docs, proving), func(f string) bool { return changed[f] }) {
		return nil
	}
	return []string{fmt.Sprintf("%s was deleted, leaving %s unchanged; change or delete them with it", file, behind(docs, proving))}
}

// namingDocs lists the requirement docs whose text names file by its whole
// path.
func namingDocs(root, file string) []string {
	named := regexp.MustCompile(`(^|[^\w./-])` + regexp.QuoteMeta(file) + `\b`)
	var docs []string
	for _, d := range requirementDocs(root) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(d)))
		if err == nil && named.Match(data) {
			docs = append(docs, d)
		}
	}
	return docs
}

// provingTests lists the test files that prove each doc's requirement.
func provingTests(docs []string, tests testSet) []string {
	var files []string
	for _, d := range docs {
		for _, f := range tests.numbers[docNumber(d)] {
			if !slices.Contains(files, f) {
				files = append(files, f)
			}
		}
	}
	return files
}

// behind names the docs and tests a deletion leaves behind.
func behind(docs, tests []string) string {
	text := "requirement doc " + joinList(docs)
	if len(tests) > 0 {
		text += " and test " + joinList(tests)
	}
	return text
}
