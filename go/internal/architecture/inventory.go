package architecture

import (
	"path"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
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
	m.repo.kept = map[string]bool{}
	m.repo.dev = runtimes.DevScripts(root, files)
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
	return category(file, m.repo.kept[file] || runtimes.Manifest(file), m.repo.dev[file]) == "product"
}

// category sorts a tracked source: generated, test, tool, or product. dev
// marks a file only package.json scripts run.
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

// tool reports a shell or Python script, a source under the top-level
// scripts folder, or a root-level script tool's config file, such as
// rollup.config.mjs. A config module deeper down, such as
// src/app/app.config.ts, is product code.
func tool(file string) bool {
	ext := path.Ext(file)
	if contains([]string{".sh", ".bash", ".zsh", ".py"}, ext) || strings.HasPrefix(file, "scripts/") {
		return true
	}
	return contains(walkfs.ScriptExts, ext) && walkfs.ToolConfig(file) && !strings.Contains(file, "/")
}

func testFile(lower string) bool {
	return testFolder(path.Dir(lower)) || strings.HasSuffix(lower, "_test.go")
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

// ProductCode lists the Go, Swift, TypeScript, and JavaScript product files
// that need architecture views: tracked, not generated, not tests, and not
// tools such as vite.config.ts or a file only package.json scripts run.
func ProductCode(root string) []string {
	code := walkfs.WithoutGenerated(root, walkfs.FilesByExt(root, ".go", ".swift"))
	dev := runtimes.DevScripts(root, walkfs.FilesByExt(root, ""))
	var out []string
	for _, file := range append(code, walkfs.Scripts(root)...) {
		if category(file, true, dev[file]) == "product" && !walkfs.ScriptTest(file) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}
