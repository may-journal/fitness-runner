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
	cfg, ok := readVitestObject(path)
	if !ok {
		return nil, false
	}
	cov, ok := coverageBlockOf(cfg).(map[string]any)
	if !ok {
		return nil, true
	}
	list, ok := cov["exclude"].([]any)
	if !ok {
		return nil, true
	}
	return stringEntries(list), true
}

// readVitestObject reads the "vitest" object of the package.json at path;
// ok is false when the file is unreadable or malformed or the key does not
// hold an object — the TS tryLoadPackageJsonVitest contract.
func readVitestObject(path string) (map[string]any, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, false
	}
	cfg, ok := pkg["vitest"].(map[string]any)
	return cfg, ok
}

// stringEntries returns the string elements of a decoded JSON array,
// skipping everything else — the TS check's only-strings-count rule.
func stringEntries(list []any) []string {
	var out []string
	for _, entry := range list {
		if s, ok := entry.(string); ok {
			out = append(out, s)
		}
	}
	return out
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
		if isDir(pkgDir) {
			return cspellExportDir(pkgDir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
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
		if !isCoverageOpener(tokens, i) {
			continue
		}
		if ex, ok := excludeInObject(tokens, i+3); ok {
			return ex, true
		}
	}
	return nil, false
}

// isCoverageOpener reports whether tokens[i:] opens a coverage object
// literal: the key "coverage", a colon, and an opening brace. The caller
// guarantees tokens i+1 and i+2 exist.
func isCoverageOpener(tokens []token, i int) bool {
	return isKey(tokens, i, "coverage") && isPunct(tokens[i+1], ":") && isPunct(tokens[i+2], "{")
}

// excludeInObject scans an object literal's tokens (start is the index just
// past its opening brace) for a direct exclude key holding an array.
func excludeInObject(tokens []token, start int) ([]string, bool) {
	depth := 1
	for i := start; i < len(tokens); i++ {
		depth += tokenDepthDelta(tokens[i])
		if depth == 0 {
			return nil, false
		}
		if depth == 1 && isExcludeOpener(tokens, i) {
			return arrayStrings(tokens, i+3), true
		}
	}
	return nil, false
}

// isExcludeOpener reports whether tokens[i:] opens an exclude array: the key
// "exclude", a colon, and an opening bracket, with both punctuation tokens
// in range.
func isExcludeOpener(tokens []token, i int) bool {
	return i+2 < len(tokens) && isKey(tokens, i, "exclude") &&
		isPunct(tokens[i+1], ":") && isPunct(tokens[i+2], "[")
}

// tokenDepthDelta returns the nesting-depth change a token causes: +1 for
// opening punctuation, -1 for closing, 0 for everything else.
func tokenDepthDelta(t token) int {
	if t.kind != tokPunct {
		return 0
	}
	switch t.text {
	case "{", "[", "(":
		return 1
	case "}", "]", ")":
		return -1
	}
	return 0
}

// arrayStrings collects the clean string-literal elements of an array (start
// is the index just past its opening bracket): a literal counts only when it
// sits directly between element delimiters — anything wrapped in a call,
// spread, or operator expression is unresolvable and skipped.
func arrayStrings(tokens []token, start int) []string {
	var out []string
	depth := 1
	for i := start; i < len(tokens); i++ {
		depth += tokenDepthDelta(tokens[i])
		if depth == 0 {
			return out
		}
		if depth == 1 && isArrayElement(tokens, i) {
			out = append(out, tokens[i].text)
		}
	}
	return out
}

// isArrayElement reports whether tokens[i] is a clean string literal sitting
// directly between array element delimiters: the opening bracket or a comma
// before it, a comma or the closing bracket after it.
func isArrayElement(tokens []token, i int) bool {
	t := tokens[i]
	if t.kind != tokString || !t.lit || i+1 >= len(tokens) {
		return false
	}
	return isPunctEither(tokens[i-1], "[", ",") && isPunctEither(tokens[i+1], ",", "]")
}

// isKey reports whether token i is an object key named name: an identifier
// or plain string literal in key position (start of input, or right after an
// opening brace or comma).
func isKey(tokens []token, i int, name string) bool {
	t := tokens[i]
	return isKeyToken(t) && t.text == name && inKeyPosition(tokens, i)
}

// isKeyToken reports whether t can serve as an object key: an identifier or
// a plain (fully literal) string.
func isKeyToken(t token) bool {
	return t.kind == tokIdent || (t.kind == tokString && t.lit)
}

// inKeyPosition reports whether index i is object-key position: the start of
// input, or right after an opening brace or comma.
func inKeyPosition(tokens []token, i int) bool {
	return i == 0 || isPunctEither(tokens[i-1], "{", ",")
}

// isPunctEither reports whether t is a punctuation token whose text is a or b.
func isPunctEither(t token, a, b string) bool {
	return t.kind == tokPunct && (t.text == a || t.text == b)
}

func isPunct(t token, s string) bool {
	return t.kind == tokPunct && t.text == s
}
