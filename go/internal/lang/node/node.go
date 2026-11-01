// Package node holds what the Node checks share: JavaScript project detection,
// tool binary lookup, script files and extension sets.
package node

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Manifest is the file that marks a Node package.
const Manifest = "package.json"

// Manifests lists root's tracked package.json files, sorted.
func Manifests(root string) []string {
	return walkfs.FilesByExt(root, Manifest)
}

// Applies reports whether root is a JavaScript project, so the Node checks
// have something to judge. It is the one place that rule lives.
func Applies(root string) bool {
	return len(Manifests(root)) > 0
}

// HasSource reports whether root tracks JavaScript or TypeScript source, so
// ESLint and Vitest have code to judge. A package.json used only for
// formatting tools does not count, and neither does a tool config such as
// eslint.config.mjs or vitest.config.ts.
func HasSource(root string) bool {
	return slices.ContainsFunc(walkfs.FilesByExt(root, LintableExts...), func(p string) bool {
		return !isToolConfig(filepath.Base(p))
	})
}

// toolConfigs are the tools whose `<tool>.config.<ext>` file configures the
// tool rather than being product code.
var toolConfigs = []string{"eslint", "vitest", "vite", "jest", "prettier", "babel", "rollup", "webpack"}

// isToolConfig reports whether a file name is one of those tool configs.
func isToolConfig(name string) bool {
	return slices.ContainsFunc(toolConfigs, func(tool string) bool {
		return strings.HasPrefix(name, tool+".config.")
	})
}

// Ancestors lists root, made absolute when it can be, and each folder above
// it up to the filesystem root, nearest first.
func Ancestors(root string) []string {
	dir := root
	if abs, err := filepath.Abs(root); err == nil {
		dir = abs
	}
	var out []string
	for {
		out = append(out, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			return out
		}
		dir = parent
	}
}

// FindTool resolves a tool's binary without npx: the first node_modules/.bin
// walking up from root, then PATH.
func FindTool(root, name string) (bin string, ok bool) {
	for _, dir := range Ancestors(root) {
		if p := filepath.Join(dir, "node_modules", ".bin", name); executable(p) {
			return p, true
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, true
	}
	return "", false
}

// executable reports whether p is a file with an execute bit set.
func executable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// The extension sets below are named subsets of ScriptExts, so every
// check shares one definition.
var (
	// TypeScriptExts are the TypeScript source endings, without TSX.
	TypeScriptExts = subset(".cts", ".mts", ".ts")
	// SourceExts are the TypeScript and JavaScript source endings scanned
	// for directives and literals, without JSX.
	SourceExts = subset(".cts", ".mts", ".ts", ".cjs", ".js", ".mjs", ".tsx")
	// LintableExts are the endings ESLint lints.
	LintableExts = subset(".cjs", ".js", ".mjs", ".ts", ".tsx")
	// InventoryExts are the endings the architecture views list as
	// sources, without the module-only TypeScript endings.
	InventoryExts = subset(".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs")
)

// subset returns exts, panicking at start-up when one is not a script
// extension, so a set can never drift from ScriptExts.
func subset(exts ...string) []string {
	for _, e := range exts {
		if !slices.Contains(ScriptExts, e) {
			panic("node: " + e + " is not in ScriptExts")
		}
	}
	return exts
}
