// Command fitness-check-jscpd is the native soul of the jscpd check:
// duplicated lines above 1% of scanned lines fail the repo. Instead of
// shelling out to jscpd it runs internal/clonedetect — a generic lexer plus a
// rolling-hash detector with jscpd's min-lines/min-tokens semantics and
// the jscpd:ignore-start/-end escape hatch. Scan scope: every file under
// root minus the TS check's ignore globs, minus gitignored files (jscpd
// respects .gitignore by default), minus binary files, with the walker's
// standard directories pruned. Verdict and error format match the TS
// check; exact percentage parity with jscpd's per-language tokenizers is
// explicitly not the goal.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/clonedetect"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// jscpd's thresholds as the TS check passes them: --threshold 1
// --min-lines 5 --min-tokens 50.
const (
	threshold = 1.0
	minLines  = 5
	minTokens = 50
)

// ignoreGlobs is the TS check's --ignore set, verbatim.
var ignoreGlobs = []string{
	"**/*.md",
	"**/*.json",
	"**/*.lock",
	"**/*.test.*",
	"**/*.spec.*",
	"**/*_test.go",
}

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "jscpd"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := lexTargets(root, scanTargets(root))
	stats := clonedetect.Detect(files, clonedetect.Options{MinLines: minLines, MinTokens: minTokens})
	if pct := stats.Percentage(); pct > threshold {
		return checkkit.Fail(len(files), fmt.Sprintf(
			"ERROR: jscpd found too many duplicates (%.1f%%) over threshold (%.1f%%)",
			pct, threshold)), nil
	}
	return checkkit.Pass(len(files)), nil
}

// scanTargets walks every file under root (the empty suffix matches all,
// with walkfs's standard skip dirs pruned), then drops the ignore-glob
// matches and gitignored paths.
func scanTargets(root string) []string {
	var kept []string
	for _, rel := range walkfs.FilesByExt(root, "") {
		if matchesAny(ignoreGlobs, rel) {
			continue
		}
		kept = append(kept, rel)
	}
	return dropGitignored(kept, clonedetect.GitIgnored(root, kept))
}

// dropGitignored removes the paths present in ignored; an empty ignored set
// passes kept through untouched.
func dropGitignored(kept []string, ignored map[string]bool) []string {
	if len(ignored) == 0 {
		return kept
	}
	var out []string
	for _, rel := range kept {
		if !ignored[rel] {
			out = append(out, rel)
		}
	}
	return out
}

// lexTargets reads and lexes each path, skipping binary files and files
// that vanish mid-scan; what remains is what "scanned" means, so its
// length is the check's filesChecked.
func lexTargets(root string, paths []string) []clonedetect.File {
	lexed := par.Map(len(paths), 0, func(i int) *clonedetect.File {
		return lexOne(root, paths[i])
	})
	var files []clonedetect.File
	for _, f := range lexed {
		if f != nil {
			files = append(files, *f)
		}
	}
	return files
}

// lexOne reads and lexes a single path for the parallel pool; nil marks a
// binary or vanished file (skipped, not scanned).
func lexOne(root, rel string) *clonedetect.File {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || isBinary(raw) {
		return nil
	}
	content := string(raw)
	return &clonedetect.File{
		Path:   rel,
		Tokens: clonedetect.Lex(content),
		Lines:  clonedetect.CountLines(content),
	}
}

func matchesAny(globs []string, rel string) bool {
	for _, g := range globs {
		if clonedetect.MatchGlob(g, rel) {
			return true
		}
	}
	return false
}

// isBinary applies git's heuristic: a NUL byte in the first 8000 bytes.
func isBinary(raw []byte) bool {
	probe := raw
	if len(probe) > 8000 {
		probe = probe[:8000]
	}
	for _, b := range probe {
		if b == 0 {
			return true
		}
	}
	return false
}
