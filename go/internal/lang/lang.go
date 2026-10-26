// Package lang defines what a language module brings to a run, and orders
// what the modules claim against what is installed or embedded. A module
// names how to spot its language, the files it reads and writes, the tools
// it runs, its checks, its spell dictionaries, the words that name it as a
// technology, and how it proves a runtime. Everything no module claims is
// repo-wide, so a new repo-wide check or dictionary needs no list entry. The
// modules themselves live in the langs package, and each language's own code
// in a folder named for it beside that package.
//
//fitness:topic languages
package lang

import (
	"embed"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

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
	// Lockfiles are the file names its package managers write on install,
	// each a name or a path.Match pattern for one.
	Lockfiles []string
	// Wrappers are the names of the files its build tools' wrappers write,
	// and a repo commits as written, to pin the tool's version.
	Wrappers []string
	// BuildOutput, when set, lists the tracked files in a repo that are
	// build output the language's projects write, which git must not track.
	BuildOutput func(root string) []string
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
// any.
type Test struct {
	// Name is the test's name as declared.
	Name string
	// ID is the acceptance ID the name carries, or "" for none.
	ID string
	// Loc is where the test is declared, as file:line.
	Loc string
	// File is the path of the file declaring the test.
	File string
}

// Tests is how a module finds its requirement tests.
type Tests struct {
	// Scan, when set, reads the repo's tests and the files holding them,
	// and reports whether the repo is written in the language. A module
	// without it finds no requirement tests yet.
	Scan func(root string) (tests []Test, files []string, written bool, err error)
	// Name is the test name that owns an acceptance ID, such as Test0001_1.
	Name func(id string) string
	// Source, when set, reports a source path, as tracked, only tests
	// build, which the architecture views class as a test. It gets the
	// path's own case, since a language may tell folders apart by it.
	Source func(file string) bool
	// Numbers, when set, lists the requirement numbers a test file's name
	// carries, or none for a file not named for requirements.
	Numbers func(file string) []string
	// Tools, when set, judges how the repo's tests are written and run,
	// given the files Scan found tests in. It returns one error per test
	// file or run that uses a tool the language's requirement tests may
	// not use.
	Tools func(root string, files []string) ([]string, error)
}

// TestPrefix is how a module whose test names carry IDs names them: the
// prefix, then the acceptance ID with _ for its dot, so Test0001_1 or
// test0001_1 owns 0001.1.
type TestPrefix string

// prefixedID reads the ID after a test name's prefix: 0001_1 is 0001.1.
var prefixedID = regexp.MustCompile(`^(\d{4})_([1-9]\d*)$`)

// Name is the test name that owns id.
func (p TestPrefix) Name(id string) string {
	return string(p) + strings.Replace(id, ".", "_", 1)
}

// ID returns the acceptance ID a test name carries, or "".
func (p TestPrefix) ID(name string) string {
	rest, ok := strings.CutPrefix(name, string(p))
	m := prefixedID.FindStringSubmatch(rest)
	if !ok || m == nil {
		return ""
	}
	return m[1] + "." + m[2]
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

// Testing returns the modules that find requirement tests, in registry
// order.
func Testing(modules []Module) []Module {
	var out []Module
	for _, m := range modules {
		if m.Tests.Scan != nil {
			out = append(out, m)
		}
	}
	return out
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

// IsTestSource reports whether a module classes the source path as built
// only for tests.
func IsTestSource(modules []Module, file string) bool {
	return slices.ContainsFunc(modules, func(m Module) bool { return m.Tests.Source != nil && m.Tests.Source(file) })
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

// IgnoredWhy is the reason the ignores folder gives for keeping check out
// of the default catalog, its comment lines joined, or "" when it names
// no such check.
func IgnoredWhy(check string) string {
	body, err := fs.ReadFile(ignores, path.Join("ignores", check))
	if err != nil {
		return ""
	}
	var words []string
	for _, line := range strings.Split(string(body), "\n") {
		if text, ok := strings.CutPrefix(strings.TrimSpace(line), "#"); ok {
			words = append(words, strings.Fields(text)...)
		}
	}
	return strings.Join(words, " ")
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
		if fields := dataLine(string(body)); len(fields) > 0 {
			out[fields[0]] = append(out[fields[0]], fields[1:]...)
		}
	}
	return out
}

// dataLine reads the fields of a folder file's first line that is not a
// comment.
func dataLine(body string) []string {
	for _, line := range strings.Split(body, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && !strings.HasPrefix(fields[0], "#") {
			return fields
		}
	}
	return nil
}

// lockfiles holds one file per ecosystem no module covers whose package
// manager writes a lockfile. Each says why in comments, then names its
// lockfiles.
//
//go:embed lockfiles
var lockfiles embed.FS

// otherLockfiles lists the lockfile names in the lockfiles folder, in file
// name order, read once.
var otherLockfiles = sync.OnceValue(func() []string {
	entries, err := fs.ReadDir(lockfiles, "lockfiles")
	if err != nil {
		panic("lang: " + err.Error())
	}
	var out []string
	for _, e := range entries {
		body, err := fs.ReadFile(lockfiles, path.Join("lockfiles", e.Name()))
		if err != nil {
			panic("lang: " + err.Error())
		}
		out = append(out, dataLine(string(body))...)
	}
	return out
})

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

// Lockfiles returns every module's lockfile names, in registry order, then
// those of the ecosystems no module covers.
func Lockfiles(modules []Module) []string {
	var out []string
	for _, m := range modules {
		out = append(out, m.Lockfiles...)
	}
	return append(out, otherLockfiles()...)
}

// IsLockfile returns a test, built once, of whether a file's base name is
// a lockfile a package manager writes, by any module's names or patterns
// or the lockfiles folder.
func IsLockfile(modules []Module) func(file string) bool {
	patterns := Lockfiles(modules)
	return func(file string) bool {
		base := path.Base(file)
		return slices.ContainsFunc(patterns, func(pattern string) bool {
			matched, err := path.Match(pattern, base)
			return err == nil && matched
		})
	}
}

// IsToolWritten returns a test, built once, of whether a file is one a
// tool writes and a repo commits as written: a lockfile, or a build
// wrapper file any module names. Its words and repeats are the tool's.
func IsToolWritten(modules []Module) func(file string) bool {
	lockfile := IsLockfile(modules)
	var wrappers []string
	for _, m := range modules {
		wrappers = append(wrappers, m.Wrappers...)
	}
	return func(file string) bool {
		return lockfile(file) || slices.Contains(wrappers, path.Base(file))
	}
}

// BuildOutput returns the tracked build output files every module finds
// in the repo at root, in registry order.
func BuildOutput(modules []Module, root string) []string {
	var out []string
	for _, m := range modules {
		if m.BuildOutput != nil {
			out = append(out, m.BuildOutput(root)...)
		}
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
