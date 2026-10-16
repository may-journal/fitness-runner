// Package lang is the registry of modules: what each language brings to a
// run, and the repo-wide checks every repo gets. A module names how to spot
// its language, the files it reads and writes, the tools it runs, and its
// checks. The check catalog, the lockfiles cspell skips, and the spell
// dictionaries are read from here, so adding a language means adding one
// module.
package lang

import (
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/nodejs"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Module is one entry in the registry. A nil Detect means every repo has it.
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
	// Checks are its check names, in the order their rows print.
	Checks []string
	// Dicts are the embedded spell dictionary files its words come from.
	Dicts []string
}

// Repo holds the checks every repo runs, whatever its languages.
var Repo = Module{
	Name: "repo",
	Checks: []string{
		"read-repo-first", "semantic-commit", "plan-trailer", "issue-link-once",
		"commit-attribution", "changelog", "changelog-updated", "changelog-bullets",
		"release-changelog", "cspell", "jscpd", "gitignore-why", "no-plans-dir",
		"doc-template", "requirements", "architecture-contract", "architecture-actors",
		"architecture-containers", "architecture-components", "architecture-code",
		"architecture-journey", "no-contrastive-reframing", "repeated-string-literals",
		"markdown-front-matter", "markdown-filename-kebab-case",
		"markdown-filename-camel-case", "markdown-links", "markdown-no-bold-italic",
		"text-readability", "prose-budget", "mermaid-callouts", "mermaid-callout-why",
		"mermaid-diagram-prose", "mermaid-diagram-table-gap", "mermaid-legend",
		"mermaid-level-bleed", "plan-structure", "pr-structure", "pr-closes-issue",
		"issue-checklist", "local-branch",
	},
	Dicts: []string{
		"en-us.txt", "software-terms.txt", "html.txt", "companies.txt", "aws.txt",
		"css.txt", "fullstack.txt", "filetypes.txt", "shell.txt", "git.txt",
	},
}

// Go is the Go language: a repo with a go.mod outside test fixtures.
var Go = Module{
	Name:      "go",
	Detect:    func(root string) bool { return len(toolchain.Modules(root)) > 0 },
	Exts:      []string{".go"},
	Manifests: []string{"go.mod"},
	Lockfiles: []string{"go.sum"},
	Tools:     []string{"go", "gofmt"},
	Checks:    []string{"go-complexity", "go-vet", "gofmt", "go-test"},
	Dicts:     []string{"golang.txt"},
}

// BuildOutput keeps build output out of git; it runs in every repo, and its
// row prints between the Go and Swift rows, where it always has.
var BuildOutput = Module{
	Name:   "build-output",
	Checks: []string{"build-output-untracked"},
}

// Swift is the Swift language: a repo with tracked .swift sources.
var Swift = Module{
	Name:      "swift",
	Detect:    func(root string) bool { return len(walkfs.FilesByExt(root, ".swift")) > 0 },
	Exts:      []string{".swift"},
	Manifests: []string{"Package.swift"},
	Lockfiles: []string{"Package.resolved"},
	Tools:     []string{"swiftlint"},
	Checks:    []string{"swiftlint"},
}

// Node is JavaScript and TypeScript: a repo with a tracked package.json.
var Node = Module{
	Name:      "node",
	Detect:    nodejs.Applies,
	Exts:      walkfs.ScriptExts,
	Manifests: []string{nodejs.Manifest},
	Lockfiles: []string{
		"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
		"bun.lock", "bun.lockb", "deno.lock",
	},
	Tools: []string{"eslint", "prettier", "biome", "vitest"},
	Checks: []string{
		"eslint", "prettier", "no-eslint-disable", "node-version",
		"dependency-currency", "vitest-coverage-exclude", "vitest-coverage-full",
	},
	Dicts: []string{"node.txt", "npm.txt", "typescript.txt"},
}

// Modules is the registry, in the order the check rows print.
var Modules = []Module{Repo, Go, BuildOutput, Swift, Node}

// OptIn names the check that runs only when a repo asks for it, so it is in
// no module's catalog.
const OptIn = "go-test-coverage"

// Checks returns the default check catalog: every module's checks, in
// registry order.
func Checks() []string { return collect(func(m Module) []string { return m.Checks }) }

// Lockfiles returns every module's lockfile names, in registry order.
func Lockfiles() []string { return collect(func(m Module) []string { return m.Lockfiles }) }

// Dicts returns every module's spell dictionary files, in registry order.
func Dicts() []string { return collect(func(m Module) []string { return m.Dicts }) }

// collect joins one field across the registry.
func collect(field func(Module) []string) []string {
	var out []string
	for _, m := range Modules {
		out = append(out, field(m)...)
	}
	return slices.Clip(out)
}
