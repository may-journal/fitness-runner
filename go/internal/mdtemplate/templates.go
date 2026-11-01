package mdtemplate

import (
	"path"
	"strings"
)

// IsTemplate reports whether a slash path names a markdown template of any
// kind: one that governs its neighbors, or a GitHub PR or Issue template.
// doc-template judges docs against these, and duplicate-code skips them,
// since a doc must repeat its template by design.
func IsTemplate(p string) bool {
	return strings.HasSuffix(p, ".md") && (IsConformanceTemplate(p) || isGithubTemplate(p))
}

// IsConformanceTemplate reports whether the file names a template that
// governs its neighbors: template.md, *.template.md, or any markdown file
// directly in a templates folder.
func IsConformanceTemplate(p string) bool {
	base := path.Base(p)
	return InTemplatesFolder(p) || base == "template.md" || strings.HasSuffix(base, ".template.md")
}

// InTemplatesFolder reports whether the file sits directly in a folder named
// templates.
func InTemplatesFolder(p string) bool {
	return path.Base(path.Dir(p)) == "templates"
}

// isGithubTemplate reports whether the file is a GitHub PR or Issue template.
func isGithubTemplate(p string) bool {
	if path.Base(p) == "PULL_REQUEST_TEMPLATE.md" {
		return true
	}
	return strings.Contains(p, ".github/ISSUE_TEMPLATE/") && strings.HasSuffix(p, ".md")
}
