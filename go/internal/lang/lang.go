// Package lang defines what a language module brings to a run, and orders
// what the modules claim against what is installed or embedded. A module
// names how to spot its language, the files it reads and writes, the tools
// it runs, its checks, its spell dictionaries, the words that name it as a
// technology, and how it proves a runtime. Everything no module claims is
// repo-wide, so a new repo-wide check or dictionary needs no list entry. The
// modules themselves live in the langs package, and each language's own code
// in a folder named for it beside that package.
package lang

import (
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Module is one language's entry in the registry.
type Module struct {
	// Name labels the module.
	Name string
	// Detect reports whether the repo at root uses the module's language.
	Detect func(root string) bool
	// Exts are the source file endings the language owns.
	Exts []string
	// Manifests are the file names that mark a project of the language.
	Manifests []string
	// Lockfiles are the file names its package managers write on install.
	Lockfiles []string
	// Tools are the programs its checks run.
	Tools []string
	// Checks are the check names the module claims.
	Checks []string
	// Dicts are the embedded spell dictionary files the module claims.
	Dicts []string
	// Technology is the name the language's runtime products give their
	// technology.
	Technology string
	// TechnologyWords are the lowercase words an architecture container's
	// technology string may use to name the language.
	TechnologyWords []string
	// Runtime reads the language's manifests and sources for the runtimes
	// a repo ships.
	Runtime runtimes.Reader
	// Views reads the language's sources for the architecture views:
	// declarations and relations for the code view, and, when set, the
	// references that prove the components view's edges.
	Views codemodel.Reader
	// Inventory are the source endings the architecture views list as the
	// language's sources.
	Inventory []string
	// Tests finds the language's requirement tests.
	Tests Tests
	// Scans are the source endings a repo-wide check reads in the
	// language, by check name.
	Scans map[string][]string
	// ToolConfig, when set, reports a source file that configures a tool,
	// such as a bundler config, rather than product code.
	ToolConfig func(file string) bool
	// TestFile, when set, reports a file the language's test runner finds
	// by its name or folder.
	TestFile func(file string) bool
}

// Test is one test declaration and the acceptance ID its name carries, if
// any. Loc is file:line.
type Test struct {
	Name, ID, Loc, File string
}

// Tests is how a module finds its requirement tests.
type Tests struct {
	// Scan reads the repo's tests and the files holding them, and reports
	// whether the repo is written in the language.
	Scan func(root string) (tests []Test, files []string, written bool, err error)
	// Name is the test name that owns an acceptance ID, such as Test0001_1.
	Name func(id string) string
	// Source, when set, reports a lowercase source path only tests build,
	// which the architecture views class as a test.
	Source func(lower string) bool
	// Numbers, when set, lists the requirement numbers a test file's name
	// carries, or none for a file not named for requirements.
	Numbers func(file string) []string
}

// TestFileNumbers lists the requirement numbers a test file's name
// carries, from the first module that reads any.
func TestFileNumbers(modules []Module, file string) []string {
	for _, m := range modules {
		if m.Tests.Numbers == nil {
			continue
		}
		if numbers := m.Tests.Numbers(file); numbers != nil {
			return numbers
		}
	}
	return nil
}

// ScanExts returns every module's source endings for the check, in
// registry order.
func ScanExts(modules []Module, check string) []string {
	var out []string
	for _, m := range modules {
		out = append(out, m.Scans[check]...)
	}
	return out
}

// Exts returns every module's source endings, in registry order.
func Exts(modules []Module) []string {
	var out []string
	for _, m := range modules {
		out = append(out, m.Exts...)
	}
	return out
}

// IsTestFile reports whether any module's test runner finds file by name.
func IsTestFile(modules []Module, file string) bool {
	return slices.ContainsFunc(modules, func(m Module) bool { return m.TestFile != nil && m.TestFile(file) })
}

// IsTestSource reports whether a module classes the lowercase source path
// as built only for tests.
func IsTestSource(modules []Module, lower string) bool {
	return slices.ContainsFunc(modules, func(m Module) bool { return m.Tests.Source != nil && m.Tests.Source(lower) })
}

// IsToolConfig reports whether the module owning file's ending says it
// configures a tool.
func IsToolConfig(modules []Module, file string) bool {
	return slices.ContainsFunc(modules, func(m Module) bool {
		return m.ToolConfig != nil && slices.Contains(m.Exts, path.Ext(file)) && m.ToolConfig(file)
	})
}

// Runtimes returns every module's runtime reader, in registry order.
func Runtimes(modules []Module) []runtimes.Reader {
	out := make([]runtimes.Reader, 0, len(modules))
	for _, m := range modules {
		out = append(out, m.Runtime)
	}
	return out
}

// Views returns every module's architecture view reader, in registry
// order, leaving out modules that read none.
func Views(modules []Module) []codemodel.Reader {
	var out []codemodel.Reader
	for _, m := range modules {
		if m.Views.Read != nil {
			out = append(out, m.Views)
		}
	}
	return out
}

// Inventory returns every module's inventory endings, in registry order.
func Inventory(modules []Module) []string {
	var out []string
	for _, m := range modules {
		out = append(out, m.Inventory...)
	}
	return out
}

// TechnologyWords maps each module's technology name to its words.
func TechnologyWords(modules []Module) map[string][]string {
	out := map[string][]string{}
	for _, m := range modules {
		out[m.Technology] = m.TechnologyWords
	}
	return out
}

// ignores holds one file per check name kept out of the default catalog;
// each file says why.
//
//go:embed ignores
var ignores embed.FS

// Ignored returns the names in the ignores folder, sorted.
func Ignored() []string {
	entries, err := fs.ReadDir(ignores, "ignores")
	if err != nil {
		panic("lang: " + err.Error())
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// sources holds one file per language no module covers whose files the
// architecture views still list. Each says why in comments, then names a
// role, tool or product, and the language's source endings.
//
//go:embed sources
var sources embed.FS

// Sources maps each role to the endings of the languages no module covers
// that play it, in file name order.
func Sources() map[string][]string {
	entries, err := fs.ReadDir(sources, "sources")
	if err != nil {
		panic("lang: " + err.Error())
	}
	out := map[string][]string{}
	for _, e := range entries {
		body, err := fs.ReadFile(sources, path.Join("sources", e.Name()))
		if err != nil {
			panic("lang: " + err.Error())
		}
		role, exts := sourceLine(string(body))
		out[role] = append(out[role], exts...)
	}
	return out
}

// sourceLine reads the role and endings from a sources file's first line
// that is not a comment.
func sourceLine(body string) (string, []string) {
	for _, line := range strings.Split(body, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && !strings.HasPrefix(fields[0], "#") {
			return fields[0], fields[1:]
		}
	}
	return "", nil
}

// Catalog orders the default checks: the installed checks no module claims
// and no ignore file names, sorted, then each module's claimed checks,
// sorted, in registry order.
func Catalog(installed []string, modules []Module) []string {
	return order(installed, Ignored(), modules, func(m Module) []string { return m.Checks })
}

// Dicts orders the dictionary files: the embedded files no module claims,
// sorted, then each module's claimed files, sorted, in registry order.
func Dicts(embedded []string, modules []Module) []string {
	return order(embedded, nil, modules, func(m Module) []string { return m.Dicts })
}

// Lockfiles returns every module's lockfile names, in registry order.
func Lockfiles(modules []Module) []string {
	var out []string
	for _, m := range modules {
		out = append(out, m.Lockfiles...)
	}
	return out
}

// order lists the names in found that no module claims and skip leaves
// out, sorted, then each module's claims, sorted.
func order(found, skip []string, modules []Module, claims func(Module) []string) []string {
	claimed := claimedBy(modules, claims)
	var out []string
	for _, name := range found {
		if !claimed[name] && !slices.Contains(skip, name) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	for _, m := range modules {
		out = append(out, slices.Sorted(slices.Values(claims(m)))...)
	}
	return out
}

// claimedBy is the set of names any module claims.
func claimedBy(modules []Module, claims func(Module) []string) map[string]bool {
	claimed := map[string]bool{}
	for _, m := range modules {
		for _, name := range claims(m) {
			claimed[name] = true
		}
	}
	return claimed
}
