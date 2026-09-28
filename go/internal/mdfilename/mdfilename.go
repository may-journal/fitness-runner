// Package mdfilename validates markdown basenames against a filename
// convention — the Go port of the TypeScript runMarkdownFilenameCheck shared
// core. The markdown-filename-kebab-case and markdown-filename-camel-case
// check binaries are each a thin main over Run with their own Convention:
// same function, two flavors. Both run in every repo; Detect picks the
// repo's convention from its existing names, and only that flavor judges.
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

// capsDoc matches capitalized doc basenames (README.md, LICENSE.md,
// AGENTS.md, CODE_OF_CONDUCT.md, ...), which are exempt from the convention.
var capsDoc = regexp.MustCompile(`^[A-Z0-9_]+\.md$`)

// Validate returns the error for a non-conforming markdown path under the
// convention, or "" when the basename is valid or exempt.
func Validate(relPath string, c Convention) string {
	base := path.Base(relPath)
	if capsDoc.MatchString(base) || c.Pattern.MatchString(base) {
		return ""
	}
	return fmt.Sprintf("%s: filename must be %s", relPath, c.Label)
}

// Detect returns the convention most of the markdown basenames follow. A
// hyphenated name counts for kebab-case and a name with capitals for
// camelCase; plain lowercase names fit both and capitalized docs are exempt,
// so neither counts. Kebab-case wins a tie, including a repo with no votes.
func Detect(files []string) Convention {
	kebab, camel := 0, 0
	for _, f := range files {
		switch leaning(path.Base(f)) {
		case Kebab.Label:
			kebab++
		case Camel.Label:
			camel++
		}
	}
	if camel > kebab {
		return Camel
	}
	return Kebab
}

// leaning returns the label of the one convention a basename matches, or ""
// when it is an exempt capitalized doc or fits both or neither.
func leaning(base string) string {
	if capsDoc.MatchString(base) {
		return ""
	}
	k, c := Kebab.Pattern.MatchString(base), Camel.Pattern.MatchString(base)
	switch {
	case k == c:
		return ""
	case k:
		return Kebab.Label
	}
	return Camel.Label
}

// Run applies one convention to every in-scope .md file under root
// (capitalized doc basenames exempt), but only when it is the repo's detected
// convention; otherwise the other flavor judges and this one passes with zero
// files. Detection always reads every markdown file, so a scoped run judges by
// the whole repo's convention. filesChecked is the number of files judged.
func Run(root string, c Convention) checkkit.Result {
	all := walkfs.FilesByExt(root, ".md")
	if Detect(all).Label != c.Label {
		return checkkit.Pass(0)
	}
	files := walkfs.InScope(all)
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
