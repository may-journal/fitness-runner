// Command fitness-check-no-eslint-disable fails when any JS/TS source file
// contains an eslint disable directive (file, block, `-line`, or
// `-next-line` form) — the Go port of the no-eslint-disable check. Each hit
// reports the file, its 1-based line, and the directive token that matched.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// sourceExtensions are the source file extensions scanned for eslint disable
// directives (test files included) — the TS SOURCE_FILE_EXTENSIONS set.
var sourceExtensions = []string{".cts", ".mts", ".ts", ".cjs", ".js", ".mjs", ".tsx"}

// eslintDisableRe matches any eslint disable directive form (file, block,
// `-line`, and `-next-line`); the alternation order makes the match token
// the full directive, exactly like the TS regex.
var eslintDisableRe = regexp.MustCompile(`eslint-disable(-next-line|-line)?`)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-eslint-disable"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errs, fileCount, err := walkfs.ScanFiles(root, sourceExtensions, func(file, content string) []string {
		return scanContent(file, content)
	})
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errs) > 0 {
		return checkkit.Fail(fileCount, errs...), nil
	}
	return checkkit.Pass(fileCount), nil
}

// scanContent returns one "path:line: directive" error per line in content
// that contains an eslint disable directive — first match per line, 1-based
// line numbers.
func scanContent(relPath, content string) []string {
	var errs []string
	for i, line := range strings.Split(content, "\n") {
		if match := eslintDisableRe.FindString(line); match != "" {
			errs = append(errs, fmt.Sprintf("%s:%d: %s", relPath, i+1, match))
		}
	}
	return errs
}
