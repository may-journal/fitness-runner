// Command fitness-check-build-output-untracked validates that dist/ build
// output stays out of git and out of the import graph — the Go port of the
// build-output-untracked check. Three rules: dist must be git-ignored, no
// file under a dist directory may be git-tracked (one combined error names
// them all), and no .ts/.mts/.cts source file may import from a dist path.
// One deviation from the TypeScript original: only git's stdout is parsed
// for tracked files (the TS exec helper folded stderr into the output of a
// failing command), so a failing git command contributes no phantom tracked
// files.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Messages verbatim from the TS enUS catalog.
const (
	msgNotIgnored   = "Add dist/ to .gitignore (git check-ignore reports dist is not git-ignored)"
	msgRemovePrefix = "Remove tracked files:"
)

// sourceExtensions are the TypeScript source extensions the import scan covers.
var sourceExtensions = []string{".cts", ".mts", ".ts"}

// importSpecifierRe matches the specifier of `from '…'`, `import('…')`, and
// `require('…')`.
var importSpecifierRe = regexp.MustCompile(`\b(?:from|import|require)\s*\(?\s*['"]([^'"]+)['"]`)

// distSpecifierRe is true when a specifier reaches into a dist/ build-output
// path; the ../dist alternative deliberately has no trailing slash.
var distSpecifierRe = regexp.MustCompile(`(^|/)dist/|\.\./dist`)

// testFileRe matches a test/spec source file (fixtures legitimately contain
// dist import specifiers).
var testFileRe = regexp.MustCompile(`\.(?:test|spec)\.(?:c|m)?ts$`)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "build-output-untracked"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errors := distTrackingErrors(root)
	files := collectSourceFiles(root)
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			return checkkit.Result{}, err
		}
		errors = append(errors, scanFileForDistImports(file, string(raw))...)
	}
	if len(errors) > 0 {
		return checkkit.Fail(len(files)+1, errors...), nil
	}
	return checkkit.Pass(len(files) + 1), nil
}

// distTrackingErrors reports rule A: dist must be git-ignored and have no
// tracked files. The not-ignored error always precedes the tracked-files one.
func distTrackingErrors(root string) []string {
	var errors []string
	if !isDistIgnored(root) {
		errors = append(errors, msgNotIgnored)
	}
	if tracked := trackedDistFiles(root); len(tracked) > 0 {
		errors = append(errors, fmt.Sprintf(
			"%s %s (git rm --cached)", msgRemovePrefix, strings.Join(tracked, ", ")))
	}
	return errors
}

// isDistIgnored reports whether `git check-ignore -q dist` exits 0 in root
// (dist is git-ignored).
func isDistIgnored(root string) bool {
	cmd := exec.Command("git", "check-ignore", "-q", "dist")
	cmd.Dir = root
	return cmd.Run() == nil
}

// trackedDistFiles returns the git-tracked files under dist — the root-level
// `dist` pathspec plus the literal nested `**/dist/**` one — deduped and
// sorted.
func trackedDistFiles(root string) []string {
	direct := gitLines(root, "ls-files", "--", "dist")
	nested := gitLines(root, "ls-files", "--", "**/dist/**")
	return dedupeSorted(direct, nested)
}

// gitLines runs git with cwd root and returns the trimmed, non-empty lines
// of its stdout; a failing command yields whatever stdout it produced.
func gitLines(root string, args ...string) []string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, _ := cmd.Output()
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// dedupeSorted merges the given lists into one sorted list without duplicates.
func dedupeSorted(lists ...[]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, list := range lists {
		for _, item := range list {
			if !seen[item] {
				seen[item] = true
				out = append(out, item)
			}
		}
	}
	sort.Strings(out)
	return out
}

// collectSourceFiles returns the sorted, deduped source files (.ts/.mts/.cts)
// under root, skip dirs pruned and test/spec files excluded.
func collectSourceFiles(root string) []string {
	var out []string
	for _, file := range walkfs.FilesByExt(root, sourceExtensions...) {
		if !testFileRe.MatchString(file) {
			out = append(out, file)
		}
	}
	return out
}

// scanFileForDistImports reports rule B: a `path:line imports build output`
// error for every import specifier on any line that reaches into dist,
// specifier quoted exactly like JSON.stringify did.
func scanFileForDistImports(relPath, content string) []string {
	var errors []string
	for index, line := range strings.Split(content, "\n") {
		for _, match := range importSpecifierRe.FindAllStringSubmatch(line, -1) {
			if spec := match[1]; distSpecifierRe.MatchString(spec) {
				errors = append(errors, fmt.Sprintf(
					"%s:%d imports build output: %s", relPath, index+1, jsonQuote(spec)))
			}
		}
	}
	return errors
}

// jsonQuote renders s as a JSON string literal the way JSON.stringify does —
// no HTML escaping of <, >, or &.
func jsonQuote(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}
