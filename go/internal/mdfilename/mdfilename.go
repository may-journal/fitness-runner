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

// githubName reports whether a markdown path is a template name GitHub requires
// in the root, docs or .github, which a repo cannot rename.
func githubName(relPath string) bool {
	switch path.Dir(relPath) {
	case ".", "docs", ".github":
	default:
		return false
	}
	base := path.Base(relPath)
	return base == "pull_request_template.md" || base == "issue_template.md"
}

// exempt reports whether a markdown path is a capitalized doc or a name
// GitHub requires, neither of which the convention judges.
func exempt(relPath string) bool {
	return capsDoc.MatchString(path.Base(relPath)) || githubName(relPath)
}

// Validate returns the error for a non-conforming markdown path under the
// convention, or "" when the basename is valid or exempt.
func Validate(relPath string, c Convention) string {
	if exempt(relPath) || c.Pattern.MatchString(path.Base(relPath)) {
		return ""
	}
	return fmt.Sprintf("%s: filename must be %s", relPath, c.Label)
}

// Detect returns the convention most markdown basenames follow, with kebab-case
// winning a tie, since plain lowercase and capitalized names cast no vote.
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
// for an exempt doc or a name fitting both.
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

// Run judges in-scope .md files under root by c only when c is the convention
// detected from the whole repo.
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
