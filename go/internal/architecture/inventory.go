package architecture

import (
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

var sourceExtensions = strings.Fields(".go .swift .js .jsx .ts .tsx .mjs .cjs .py .sh .bash .zsh .c .h .cc .cpp .rs .java .kt .m .mm")

func sourceFile(file string) bool {
	return contains(sourceExtensions, path.Ext(file))
}

// inventory lists every tracked source with its category and the elements
// whose claims cover it. Unowned sources stay listed; the component check
// decides which must have an owner.
func (m *Model) inventory(root string, files []string) {
	var sources []string
	for _, file := range files {
		if sourceFile(file) {
			sources = append(sources, file)
		}
	}
	m.keep(root, files)
	for _, file := range sources {
		m.Sources = append(m.Sources, Source{Path: file, Category: category(file, m.repo.kept[file]), Owners: m.owners(file)})
	}
}

// keep records the files .gitattributes does not mark generated.
func (m *Model) keep(root string, files []string) {
	m.repo.kept = map[string]bool{}
	for _, file := range walkfs.WithoutGenerated(root, files) {
		m.repo.kept[file] = true
	}
}

// product reports a tracked file that is neither test, tool, nor
// generated, so manifests and sources alike can prove a runtime.
func (m *Model) product(file string) bool {
	return category(file, m.repo.kept[file]) == "product"
}

func category(file string, kept bool) string {
	lower := strings.ToLower(file)
	switch {
	case !kept:
		return "generated"
	case testFolder(path.Dir(lower)) || strings.HasSuffix(lower, "_test.go"):
		return "test"
	case contains([]string{".sh", ".bash", ".zsh", ".py"}, path.Ext(lower)):
		return "tool"
	}
	return "product"
}

func testFolder(dir string) bool {
	for _, segment := range strings.Split(dir, "/") {
		if contains([]string{"test", "tests", "testdata"}, segment) {
			return true
		}
	}
	return false
}

func (m *Model) owners(file string) []string {
	var owners []string
	for _, claim := range m.Claims {
		if covers(claim.Path, file) && !contains(owners, claim.Owner) {
			owners = append(owners, claim.Owner)
		}
	}
	return owners
}
