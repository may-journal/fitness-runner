// Command fitness-check-jscpd is the native soul of the jscpd check:
// duplicated lines above 1% of scanned lines fail the repo. Instead of
// shelling out to jscpd it runs internal/clonedetect — a generic lexer plus a
// rolling-hash detector with jscpd's min-lines/min-tokens semantics and
// the jscpd:ignore-start/-end escape hatch. Scan scope: every tracked file
// (internal/walkfs), minus files .gitattributes marks linguist-generated
// and binary files, which have no lines to compare. Verdict and error format
// match the TS check; exact percentage parity with jscpd's per-language
// tokenizers is explicitly not the goal.
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

// scanTargets lists every tracked file under root (the empty suffix
// matches all), minus those .gitattributes marks linguist-generated, such
// as lock files.
func scanTargets(root string) []string {
	return walkfs.WithoutGenerated(root, walkfs.FilesByExt(root, ""))
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
