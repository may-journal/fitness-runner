// Package vitestconf locates and scans Vitest configuration for coverage
// exclude patterns — the Go port of the vitest-config helpers in
// @mayjournal/fitness-shared. The TS helpers evaluated JS/TS config files
// (require for .cjs/.js, jiti for the rest); Go cannot execute JavaScript,
// so this package scans the config source textually: comments are stripped
// string-aware and the string literals of the first coverage object carrying
// a direct exclude array are collected. Entries built from variables,
// spreads, imports, concatenation, or template expressions are unresolvable
// and skipped — the TS check ignored non-string entries at runtime, so
// skipping unresolvable expressions is the same judgment applied one level
// earlier. One consequence: a config file that exists always wins as the
// config source, where the TS fell through past files whose evaluation
// crashed or produced null.
package vitestconf

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigNames are the Vitest config filenames probed in this exact order —
// the TS VITEST_CONFIG_NAMES list.
var ConfigNames = []string{
	"vitest.config.cjs",
	"vitest.config.js",
	"vitest.config.mjs",
	"vitest.config.mts",
	"vitest.config.ts",
}

const packageJSON = "package.json"
const cspellJSON = "cspell.json"

// LoadExclude resolves the coverage exclude list the way the TS check's
// loadVitestConfig(root, getFitnessRunnerRoot()) call did: from root when it
// has a config source, else from the fitness-runner fallback root.
func LoadExclude(root string) []string {
	if exclude, found := LoadExcludeFromRoot(root); found {
		return exclude
	}
	exclude, _ := LoadExcludeFromRoot(FitnessRunnerRoot(root))
	return exclude
}

// LoadExcludeFromRoot resolves the coverage exclude list from a single
// directory: the first vitest.config.* file present (in ConfigNames order)
// is the config source, else the package.json "vitest" object. found is
// false when the directory has no config source at all.
func LoadExcludeFromRoot(root string) (exclude []string, found bool) {
	for _, name := range ConfigNames {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		ex, _ := CoverageExclude(string(data))
		return ex, true
	}
	return excludeFromPackageJSON(filepath.Join(root, packageJSON))
}

// excludeFromPackageJSON reads the "vitest" object from a package.json,
// mirroring the TS tryLoadPackageJsonVitest plus getCoverageBlock: the
// coverage block is test.coverage when present and non-null, else the
// top-level coverage; only string entries of an array exclude count.
func excludeFromPackageJSON(path string) ([]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, false
	}
	cfg, ok := pkg["vitest"].(map[string]any)
	if !ok {
		return nil, false
	}
	var block any
	if test, ok := cfg["test"].(map[string]any); ok {
		if c, exists := test["coverage"]; exists && c != nil {
			block = c
		}
	}
	if block == nil {
		block = cfg["coverage"]
	}
	cov, ok := block.(map[string]any)
	if !ok {
		return nil, true
	}
	list, ok := cov["exclude"].([]any)
	if !ok {
		return nil, true
	}
	var out []string
	for _, entry := range list {
		if s, ok := entry.(string); ok {
			out = append(out, s)
		}
	}
	return out, true
}

// FitnessRunnerRoot mirrors the TS getFitnessRunnerRoot: resolve the
// installed @mayjournal/fitness-shared package's "./cspell" export and
// return the outermost ancestor of that config directory containing
// cspell.json. The TS resolved the package from its own module URL; this
// binary is not Node, so it walks up from root through node_modules — same
// install, same answer. When no usable install is found it falls back to
// the outermost cspell.json ancestor of root, as the TS did with cwd.
func FitnessRunnerRoot(root string) string {
	if dir := sharedConfigDir(root); dir != "" {
		return outermostCspellAncestor(dir)
	}
	return outermostCspellAncestor(root)
}

// sharedConfigDir walks up from root for the nearest installed
// node_modules/@mayjournal/fitness-shared and resolves its "./cspell"
// export's directory; empty when no install (or no usable export) exists.
func sharedConfigDir(root string) string {
	dir, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	for {
		pkgDir := filepath.Join(dir, "node_modules", "@mayjournal", "fitness-shared")
		if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
			return cspellExportDir(pkgDir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// cspellExportDir resolves the "./cspell" entry of the package's exports map
// to a directory — the TS dirname(import.meta.resolve('…/cspell')).
func cspellExportDir(pkgDir string) string {
	data, err := os.ReadFile(filepath.Join(pkgDir, packageJSON))
	if err != nil {
		return ""
	}
	var pkg struct {
		Exports map[string]any `json:"exports"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	target := exportTarget(pkg.Exports["./cspell"])
	if target == "" {
		return ""
	}
	return filepath.Dir(filepath.Join(pkgDir, target))
}

// exportTarget unwraps a package.json exports entry: a plain string, or a
// conditional object probed import → require → default.
func exportTarget(entry any) string {
	switch v := entry.(type) {
	case string:
		return v
	case map[string]any:
		for _, condition := range []string{"import", "require", "default"} {
			if s := exportTarget(v[condition]); s != "" {
				return s
			}
		}
	}
	return ""
}

// outermostCspellAncestor mirrors the TS walkUpWhile/findConfigRoot pair:
// walk up from start remembering the last directory containing cspell.json;
// start itself when no ancestor (start included) has one. The filesystem
// root is never tested, exactly like the TS loop.
func outermostCspellAncestor(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return start
	}
	last := dir
	for dir != filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, cspellJSON)); err == nil {
			last = dir
		}
		dir = filepath.Dir(dir)
	}
	return last
}

// CoverageExclude scans JS/TS config source text for the first coverage
// object literal carrying a direct exclude array and returns the array's
// clean string literals; found reports whether such an array exists.
func CoverageExclude(src string) (exclude []string, found bool) {
	tokens := lex(src)
	for i := 0; i+2 < len(tokens); i++ {
		if !isKey(tokens, i, "coverage") || !isPunct(tokens[i+1], ":") || !isPunct(tokens[i+2], "{") {
			continue
		}
		if ex, ok := excludeInObject(tokens, i+3); ok {
			return ex, true
		}
	}
	return nil, false
}

// excludeInObject scans an object literal's tokens (start is the index just
// past its opening brace) for a direct exclude key holding an array.
func excludeInObject(tokens []token, start int) ([]string, bool) {
	depth := 1
	for i := start; i < len(tokens); i++ {
		t := tokens[i]
		if t.kind == tokPunct {
			switch t.text {
			case "{", "[", "(":
				depth++
			case "}", "]", ")":
				depth--
				if depth == 0 {
					return nil, false
				}
			}
			continue
		}
		if depth == 1 && i+2 < len(tokens) && isKey(tokens, i, "exclude") &&
			isPunct(tokens[i+1], ":") && isPunct(tokens[i+2], "[") {
			return arrayStrings(tokens, i+3), true
		}
	}
	return nil, false
}

// arrayStrings collects the clean string-literal elements of an array (start
// is the index just past its opening bracket): a literal counts only when it
// sits directly between element delimiters — anything wrapped in a call,
// spread, or operator expression is unresolvable and skipped.
func arrayStrings(tokens []token, start int) []string {
	var out []string
	depth := 1
	for i := start; i < len(tokens); i++ {
		t := tokens[i]
		if t.kind == tokPunct {
			switch t.text {
			case "{", "[", "(":
				depth++
			case "}", "]", ")":
				depth--
				if depth == 0 {
					return out
				}
			}
			continue
		}
		if depth != 1 || t.kind != tokString || !t.lit || i+1 >= len(tokens) {
			continue
		}
		prev, next := tokens[i-1], tokens[i+1]
		if prev.kind == tokPunct && (prev.text == "[" || prev.text == ",") &&
			next.kind == tokPunct && (next.text == "," || next.text == "]") {
			out = append(out, t.text)
		}
	}
	return out
}

// isKey reports whether token i is an object key named name: an identifier
// or plain string literal in key position (start of input, or right after an
// opening brace or comma).
func isKey(tokens []token, i int, name string) bool {
	t := tokens[i]
	if t.kind != tokIdent && !(t.kind == tokString && t.lit) {
		return false
	}
	if t.text != name {
		return false
	}
	if i == 0 {
		return true
	}
	prev := tokens[i-1]
	return prev.kind == tokPunct && (prev.text == "{" || prev.text == ",")
}

func isPunct(t token, s string) bool {
	return t.kind == tokPunct && t.text == s
}
