// Package mdfilename validates markdown basenames against a filename
// convention — the Go port of the TypeScript runMarkdownFilenameCheck shared
// core. The markdown-filename-kebab-case and markdown-filename-camel-case
// check binaries are each a thin main over Run with their own Convention:
// same function, two flavors.
package mdfilename

import (
	"fmt"
	"path"
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Convention is one markdown filename convention: the pattern a basename
// must match to pass, and the human label used in error messages.
type Convention struct {
	// Label appears in the error message, e.g. "kebab-case".
	Label string
	// Pattern is the regexp the full basename (including ".md") must match.
	Pattern *regexp.Regexp
}

// Kebab is kebab-case: lowercase alphanumeric segments joined by single
// hyphens, e.g. api-design.md.
var Kebab = Convention{
	Label:   "kebab-case",
	Pattern: regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.md$`),
}

// Camel is camelCase: starts lowercase, then letters/digits, e.g.
// releaseNotes.md, adr001.md.
var Camel = Convention{
	Label:   "camelCase",
	Pattern: regexp.MustCompile(`^[a-z][a-zA-Z0-9]*\.md$`),
}

// allowedBasenames are standard root filenames always allowed regardless of
// case (conventional OSS docs).
var allowedBasenames = map[string]bool{
	"CHANGELOG.md":       true,
	"CODE_OF_CONDUCT.md": true,
	"CONTRIBUTING.md":    true,
	"LICENSE.md":         true,
	"README.md":          true,
	"SECURITY.md":        true,
}

// Validate returns the error for a non-conforming markdown path under the
// convention, or "" when the basename is valid or exempt.
func Validate(relPath string, c Convention) string {
	base := path.Base(relPath)
	if allowedBasenames[base] || c.Pattern.MatchString(base) {
		return ""
	}
	return fmt.Sprintf("%s: filename must be %s", relPath, c.Label)
}

// Run applies one convention to every .md file under root (standard root
// docs exempt); filesChecked is the number of markdown files walked.
func Run(root string, c Convention) checkkit.Result {
	files := walkfs.FilesByExt(root, ".md")
	var errors []string
	for _, file := range files {
		if msg := Validate(file, c); msg != "" {
			errors = append(errors, msg)
		}
	}
	if len(errors) > 0 {
		return checkkit.Fail(len(files), errors...)
	}
	return checkkit.Pass(len(files))
}
