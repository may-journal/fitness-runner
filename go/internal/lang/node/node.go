// Package node is the Node language module's home. It holds what the Node
// checks share: when a repo counts as a JavaScript project, where an
// installed tool's binary is, which files are scripts, and the script
// extension sets each check reads. The packages in its folder hold the Node
// tools' own logic: biome, javascript, sharedconf and vitestconf.
package node

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"

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

// FindTool resolves an installed tool's binary the way npx would, without
// npx: the first executable node_modules/.bin/<name> walking up from root,
// then <name> on PATH. ok is false when neither has it.
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

// The extension sets below are named subsets of ScriptExts, so every check
// reads from one definition while keeping the files it has always read.
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
