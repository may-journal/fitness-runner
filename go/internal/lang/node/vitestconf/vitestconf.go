// Package vitestconf scans Vitest config source text, since Go cannot run
// JavaScript, for coverage exclude patterns and skips entries it cannot resolve.
package vitestconf

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/javascript"
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

// LoadExclude resolves the coverage exclude list from root, else from the
// fallback root of fitness-runner.
func LoadExclude(root string) []string {
	if exclude, found := LoadExcludeFromRoot(root); found {
		return exclude
	}
	exclude, _ := LoadExcludeFromRoot(FitnessRunnerRoot(root))
	return exclude
}

// LoadExcludeFromRoot resolves the coverage exclude list from one directory's
// first vitest.config.* file, else its package.json "vitest" object.
func LoadExcludeFromRoot(root string) (exclude []string, found bool) {
	for _, name := range ConfigNames {
		data, err := comments.CodeFile(root, name)
		if err != nil {
			continue
		}
		ex, _ := CoverageExclude(string(data))
		return ex, true
	}
	return excludeFromPackageJSON(filepath.Join(root, packageJSON))
}

// excludeFromPackageJSON reads the string entries of the coverage exclude array
// in a package.json "vitest" object.
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

// readVitestObject reads the "vitest" object of the package.json at path, with
// ok false when it is missing or not an object.
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

// FitnessRunnerRoot returns the outermost cspell.json ancestor of the installed
// @mayjournal/fitness-shared config directory, else of root.
func FitnessRunnerRoot(root string) string {
	if dir := sharedConfigDir(root); dir != "" {
		return outermostCspellAncestor(dir)
	}
	return outermostCspellAncestor(root)
}

// sharedConfigDir resolves the "./cspell" export directory of the nearest
// installed @mayjournal/fitness-shared, or empty.
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

// outermostCspellAncestor returns the highest directory above start holding
// cspell.json, else start itself.
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

// CoverageExclude returns the clean string literals of the first coverage
// object's direct exclude array in JS/TS config source.
func CoverageExclude(src string) (exclude []string, found bool) {
	tokens := javascript.Lex(src)
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
func isCoverageOpener(tokens []javascript.Token, i int) bool {
	return isKey(tokens, i, "coverage") && isPunct(tokens[i+1], ":") && isPunct(tokens[i+2], "{")
}

// excludeInObject scans an object literal's tokens (start is the index just
// past its opening brace) for a direct exclude key holding an array.
func excludeInObject(tokens []javascript.Token, start int) ([]string, bool) {
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
func isExcludeOpener(tokens []javascript.Token, i int) bool {
	return i+2 < len(tokens) && isKey(tokens, i, "exclude") &&
		isPunct(tokens[i+1], ":") && isPunct(tokens[i+2], "[")
}

// tokenDepthDelta returns the nesting-depth change a token causes: +1 for
// opening punctuation, -1 for closing, 0 for everything else.
func tokenDepthDelta(t javascript.Token) int {
	if t.Kind != javascript.Punct {
		return 0
	}
	switch t.Text {
	case "{", "[", "(":
		return 1
	case "}", "]", ")":
		return -1
	}
	return 0
}

// arrayStrings collects the string literals sitting directly between element
// delimiters of the array opened before start.
func arrayStrings(tokens []javascript.Token, start int) []string {
	var out []string
	depth := 1
	for i := start; i < len(tokens); i++ {
		depth += tokenDepthDelta(tokens[i])
		if depth == 0 {
			return out
		}
		if depth == 1 && isArrayElement(tokens, i) {
			out = append(out, tokens[i].Text)
		}
	}
	return out
}

// isArrayElement reports whether tokens[i] is a clean string literal directly
// between array element delimiters.
func isArrayElement(tokens []javascript.Token, i int) bool {
	t := tokens[i]
	if t.Kind != javascript.String || !t.Lit || i+1 >= len(tokens) {
		return false
	}
	return isPunctEither(tokens[i-1], "[", ",") && isPunctEither(tokens[i+1], ",", "]")
}

// isKey reports whether token i is an object key named name in key position.
func isKey(tokens []javascript.Token, i int, name string) bool {
	t := tokens[i]
	return isKeyToken(t) && t.Text == name && inKeyPosition(tokens, i)
}

// isKeyToken reports whether t can serve as an object key: an identifier or
// a plain (fully literal) string.
func isKeyToken(t javascript.Token) bool {
	return t.Kind == javascript.Ident || (t.Kind == javascript.String && t.Lit)
}

// inKeyPosition reports whether index i is object-key position: the start of
// input, or right after an opening brace or comma.
func inKeyPosition(tokens []javascript.Token, i int) bool {
	return i == 0 || isPunctEither(tokens[i-1], "{", ",")
}

// isPunctEither reports whether t is a punctuation token whose text is a or b.
func isPunctEither(t javascript.Token, a, b string) bool {
	return t.Kind == javascript.Punct && (t.Text == a || t.Text == b)
}

func isPunct(t javascript.Token, s string) bool {
	return t.Kind == javascript.Punct && t.Text == s
}
