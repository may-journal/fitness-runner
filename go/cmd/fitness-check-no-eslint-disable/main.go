package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// eslintDisableRe matches any eslint disable directive form (file, block,
// `-line`, and `-next-line`); the alternation order makes the match token
// the full directive, exactly like the TS regex.
var eslintDisableRe = regexp.MustCompile(`eslint-disable(-next-line|-line)?`)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-eslint-disable", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return walkfs.ScanFiles(root, node.SourceExts, scanContent), nil
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
