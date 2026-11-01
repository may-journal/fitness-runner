package architecture

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/runtimes"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// readers are the language modules' runtime readers, in registry order.
var readers = lang.Runtimes(langs.All)

// otherSources are the endings of the languages no module covers, by the
// category the views give them.
var otherSources = lang.Sources()

// sourceExtensions are every ending the views list: each module's
// inventory, then the other languages' product and tool sources.
var sourceExtensions = slices.Concat(lang.Inventory(langs.All), otherSources["product"], otherSources["tool"])

func sourceFile(file string) bool {
	return contains(sourceExtensions, path.Ext(file))
}

// inventory lists every tracked source or manifest with its category and the
// elements whose claims cover it. The component check decides which unowned
// sources need an owner.
func (m *Model) inventory(root string, files []string) {
	classed := inventoried(files)
	var sources []string
	for _, file := range classed {
		if sourceFile(file) {
			sources = append(sources, file)
		}
	}
	m.repo.kept = map[string]bool{}
	m.repo.dev = runtimes.DevScripts(root, classed, readers)
	for _, file := range walkfs.WithoutGenerated(root, classed) {
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
	case testFile(file):
		return "test"
	case dev || tool(lower):
		return "tool"
	}
	return "product"
}

// tool reports a script in a tools language of the sources folder, or a root
// file a language module names a tool config.
func tool(file string) bool {
	if contains(otherSources["tool"], path.Ext(file)) {
		return true
	}
	return !strings.Contains(file, "/") && lang.IsToolConfig(langs.All, file)
}

// testFile reports a source in a test folder, by any case, or one a
// language module says only tests build.
func testFile(file string) bool {
	return testFolder(path.Dir(strings.ToLower(file))) || lang.IsTestSource(langs.All, file)
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

// ProductCode lists tracked product files of language modules needing
// architecture views, excluding generated files, tests and tool config.
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
