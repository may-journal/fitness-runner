package architecture

import (
	"path"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/runtimes"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Unmoved reports a scoped run whose changed files, such as tests, cannot move
// the views, so Load returns an empty model.
func Unmoved(root string) bool {
	changed := checkkit.ChangedFiles()
	if changed == nil || slices.ContainsFunc(changed, movesByName) {
		return false
	}
	return !slices.ContainsFunc(walkfs.WithoutGenerated(root, changed), shippedSource)
}

// movesByName reports a file whose name alone can move the views, such as a
// view, requirement doc, manifest or .gitattributes.
func movesByName(file string) bool {
	return viewFile(file) != "" || requirementFile(file) || runtimes.Manifest(file, readers) || path.Base(file) == ".gitattributes"
}

// shippedSource reports a source that is not a test, so a product or tool
// source once generated files are left out.
func shippedSource(file string) bool {
	return sourceFile(file) && !testFile(file)
}

// requirementFile reports a requirement doc the model reads.
func requirementFile(file string) bool {
	return strings.HasPrefix(file, "docs/requirements/") && strings.HasSuffix(file, ".md") && path.Base(file) != "template.md"
}

// inventoried lists the files the inventory classes: every source and
// every manifest, so the generated and dev-script passes skip the rest.
func inventoried(files []string) []string {
	var out []string
	for _, file := range files {
		if sourceFile(file) || runtimes.Manifest(file, readers) {
			out = append(out, file)
		}
	}
	return out
}
