package check

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
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// name is the check's name, which language modules key their scans by.
const name = "build-output-untracked"

// Messages verbatim from the TS enUS catalog.
const (
	msgNotIgnored   = "Add dist/ to .gitignore (git check-ignore reports dist is not git-ignored)"
	msgRemovePrefix = "Remove tracked files:"
)

// importSpecifierRe matches the specifier of `from '…'`, `import('…')`, and
// `require('…')`.
var importSpecifierRe = regexp.MustCompile(`\b(?:from|import|require)\s*\(?\s*['"]([^'"]+)['"]`)

// distSpecifierRe is true when a specifier reaches into a dist/ build-output
// path; the ../dist alternative deliberately has no trailing slash.
var distSpecifierRe = regexp.MustCompile(`(^|/)dist/|\.\./dist`)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: name, Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files, dist, output := scope(root)
	if !dist && len(output) == 0 {
		return checkkit.Pass(0), nil
	}
	imports, err := importErrors(root, files)
	if err != nil {
		return checkkit.Result{}, err
	}
	if errors := append(trackingErrors(root, dist, output), imports...); len(errors) > 0 {
		return checkkit.Fail(len(files)+1, errors...), nil
	}
	return checkkit.Pass(len(files) + 1), nil
}

// importErrors reports rule B over every scanned source file.
func importErrors(root string, files []string) ([]string, error) {
	var errors []string
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			return nil, err
		}
		errors = append(errors, scanFileForDistImports(file, string(raw))...)
	}
	return errors, nil
}

// scope reads the source files to scan for imports, whether dist rules apply,
// and tracked files under the language modules' build output.
func scope(root string) (files []string, dist bool, output []string) {
	files = collectSourceFiles(root)
	return files, applies(root, files), trackedOutput(root)
}

// applies reports whether the dist rules have anything to judge: scanned
// source, a dist directory on disk, or tracked dist files. A docs-only repo
// has none, so the check passes clean there and is safe in the default list.
func applies(root string, files []string) bool {
	if len(files) > 0 {
		return true
	}
	if _, err := os.Stat(filepath.Join(root, "dist")); err == nil {
		return true
	}
	return len(trackedDistFiles(root)) > 0
}

// trackingErrors reports rule A: dist must be git-ignored when its rules
// apply, and no build output may have tracked files. The not-ignored error
// always precedes the tracked-files one, which names dist and module
// output files together.
func trackingErrors(root string, dist bool, output []string) []string {
	var errors []string
	if dist && !isDistIgnored(root) {
		errors = append(errors, msgNotIgnored)
	}
	if tracked := dedupeSorted(trackedDistFiles(root), output); len(tracked) > 0 {
		errors = append(errors, fmt.Sprintf(
			"%s %s (git rm --cached)", msgRemovePrefix, strings.Join(tracked, ", ")))
	}
	return errors
}

// trackedOutput returns the tracked build output files each language
// module finds in the repo, sorted.
func trackedOutput(root string) []string {
	return dedupeSorted(lang.BuildOutput(langs.All, root))
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

// collectSourceFiles returns the sorted source files under root, tests
// included, in the endings each language module scans for this check.
func collectSourceFiles(root string) []string {
	return walkfs.FilesByExt(root, lang.ScanExts(langs.All, name)...)
}

// scanFileForDistImports reports rule B: `path:line imports build output` for
// each import reaching into dist, quoted like JSON.stringify.
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
