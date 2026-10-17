package architecture

import (
	"path"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/runtimes"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// readers are the language modules' runtime readers, in registry order.
var readers = lang.Runtimes(langs.All)

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
	m.repo.kept = map[string]bool{}
	m.repo.dev = runtimes.DevScripts(root, files, readers)
	for _, file := range walkfs.WithoutGenerated(root, files) {
		m.repo.kept[file] = true
	}
	for _, file := range sources {
		m.Sources = append(m.Sources, Source{Path: file, Category: category(file, m.repo.kept[file], m.repo.dev[file]), Owners: m.owners(file)})
	}
}

// product reports a tracked file that is neither test, tool, nor
// generated, so manifests and sources alike can prove a runtime. A build
// manifest counts whatever its linguist attribute says.
func (m *Model) product(file string) bool {
	return category(file, m.repo.kept[file] || runtimes.Manifest(file, readers), m.repo.dev[file]) == "product"
}

// category sorts a tracked source: generated, test, tool, or product. dev
// marks a file DevScripts finds, under scripts/ or only scripts run.
func category(file string, kept, dev bool) string {
	lower := strings.ToLower(file)
	switch {
	case !kept:
		return "generated"
	case testFile(lower):
		return "test"
	case dev || tool(lower):
		return "tool"
	}
	return "product"
}

// tool reports a shell or Python script, or a root-level file a language
// module names a tool config. A config module deeper down is product code.
func tool(file string) bool {
	ext := path.Ext(file)
	if contains([]string{".sh", ".bash", ".zsh", ".py"}, ext) {
		return true
	}
	return !strings.Contains(file, "/") && lang.IsToolConfig(langs.All, file)
}

func testFile(lower string) bool {
	return testFolder(path.Dir(lower)) || lang.IsTestSource(langs.All, lower)
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

// ProductCode lists the product files of every language module that need
// architecture views: tracked, not generated, not tests, not a module's tool
// config, and not a file only manifest scripts run.
func ProductCode(root string) []string {
	dev := runtimes.DevScripts(root, walkfs.FilesByExt(root, ""), readers)
	var out []string
	for _, file := range walkfs.WithoutGenerated(root, walkfs.FilesByExt(root, lang.Exts(langs.All)...)) {
		if !lang.IsToolConfig(langs.All, file) && category(file, true, dev[file]) == "product" && !lang.IsTestFile(langs.All, file) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}
