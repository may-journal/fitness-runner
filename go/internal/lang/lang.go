// Package lang defines what a language module brings to a run, and orders
// what the modules claim against what is installed or embedded. A module
// names how to spot its language, the files it reads and writes, the tools
// it runs, its checks, and its spell dictionaries. Everything no module
// claims is repo-wide, so a new repo-wide check or dictionary needs no list
// entry. The modules themselves live in the langs package.
package lang

import (
	"embed"
	"io/fs"
	"slices"
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
