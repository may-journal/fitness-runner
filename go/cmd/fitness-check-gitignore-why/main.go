package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// noComment is the violation message tail, verbatim from the TS enUS catalog.
const noComment = "has no explanatory # comment on the line above"

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "gitignore-why", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// run judges every tracked `.gitignore`, in any folder, since git reads each
// one; a repo with none passes with no file checked.
func run(root string, _ []string) (checkkit.Result, error) {
	var violations []string
	files := walkfs.InScope(gitignoreFiles(root))
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		violations = append(violations, findViolations(file, string(raw))...)
	}
	if len(violations) > 0 {
		return checkkit.Fail(len(files), violations...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// gitignoreFiles lists the tracked files named `.gitignore` under root.
func gitignoreFiles(root string) []string {
	var out []string
	for _, file := range walkfs.FilesByExt(root, ".gitignore") {
		if path.Base(file) == ".gitignore" {
			out = append(out, file)
		}
	}
	return out
}

// lineKind classifies one raw .gitignore line: blank, a `#` comment, or an
// actual ignore pattern.
type lineKind int

const (
	kindBlank lineKind = iota
	kindComment
	kindPattern
)

// classifyLine sorts a raw line into blank, comment (starts with `#`), or
// pattern (any other non-blank line).
func classifyLine(line string) lineKind {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return kindBlank
	case strings.HasPrefix(trimmed, "#"):
		return kindComment
	default:
		return kindPattern
	}
}

// isExplanatoryComment reports whether the line is a non-empty explanatory
// comment — a `#` followed by real text, not a bare `#`.
func isExplanatoryComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#") &&
		strings.TrimSpace(strings.TrimLeft(trimmed, "#")) != ""
}

// findViolations names every pattern line of the `.gitignore` at file that
// no explanatory `#` comment precedes, mirroring the TS message template
// exactly.
func findViolations(file, content string) []string {
	lines := strings.Split(content, "\n")
	var errors []string
	for i, line := range lines {
		if classifyLine(line) != kindPattern {
			continue
		}
		above := ""
		if i > 0 {
			above = lines[i-1]
		}
		if !isExplanatoryComment(above) {
			errors = append(errors, fmt.Sprintf(
				"%s:%d: pattern \"%s\" %s", file, i+1, strings.TrimSpace(line), noComment))
		}
	}
	return errors
}
