// Package adr finds architecture decision records kept as files and reads
// their parts. adr-as-issue fails such files, and `fitness adr-migrate` moves
// each into an `ADR` issue.
package adr

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Why a file is an ADR, as each rule names it.
const (
	RuleFolder   = "a Markdown file in an `adr` folder is an ADR"
	RuleTitle    = "a title starting with ADR makes the file an ADR"
	RuleSections = "`## Context`, `## Decision` and `## Consequences` sections make the file an ADR"
)

// DecisionSections are the parts of an ADR that state its decision, in order.
var DecisionSections = []string{"Context", "Decision", "Consequences"}

// issueURL matches one full GitHub issue URL.
var issueURL = regexp.MustCompile(`https://github\.com/[\w.-]+/[\w.-]+/issues/\d+`)

// pointer matches a text that is one full GitHub issue URL and nothing else.
var pointer = regexp.MustCompile(`^` + issueURL.String() + `$`)

// titleADR matches a top-level heading that starts with the word ADR.
var titleADR = regexp.MustCompile(`^ADR(?:[\s:]|-\d)`)

// IsPointer reports whether content is only a full issue URL, the text an
// ADR file keeps once its decision moved into that issue.
func IsPointer(content string) bool {
	return pointer.MatchString(strings.TrimSpace(content))
}

// FirstIssueURL is the first full issue URL in text, or "".
func FirstIssueURL(text string) string {
	return issueURL.FindString(text)
}

// Rule says which rule makes rel, holding content, an ADR file. Only
// Markdown outside `.github/`, where the issue template lives, counts, and a
// pointer to the issue is no ADR.
func Rule(rel, content string) (string, bool) {
	switch {
	case !candidate(rel, content):
		return "", false
	case inADRFolder(rel):
		return RuleFolder, true
	case titledADR(content):
		return RuleTitle, true
	case hasDecision(content):
		return RuleSections, true
	}
	return "", false
}

// candidate reports whether rel may be an ADR file: Markdown outside
// `.github/` that is no pointer to an issue.
func candidate(rel, content string) bool {
	return MayBe(rel) && !IsPointer(content)
}

// MayBe reports whether rel's path lets it be an ADR file, Markdown outside
// `.github/`, so a caller reads only those.
func MayBe(rel string) bool {
	return mdx.IsMarkdown(strings.ToLower(rel)) && !strings.HasPrefix(rel, ".github/")
}

// inADRFolder reports whether a folder of rel is named `adr`. The folder's
// README or index lists the ADRs and is not one.
func inADRFolder(rel string) bool {
	if name := strings.ToLower(path.Base(rel)); name == "readme.md" || name == "index.md" {
		return false
	}
	dirs := strings.Split(rel, "/")
	for _, dir := range dirs[:len(dirs)-1] {
		if strings.EqualFold(dir, "adr") {
			return true
		}
	}
	return false
}

// titledADR reports whether content's first top-level heading starts with
// the word ADR.
func titledADR(content string) bool {
	return titleADR.MatchString(Title(content))
}

// hasDecision reports whether content holds every decision section.
func hasDecision(content string) bool {
	for _, s := range DecisionSections {
		if _, ok := mdtemplate.SectionText(content, s); !ok {
			return false
		}
	}
	return true
}

// Title is the text of content's first top-level heading, or "".
func Title(content string) string {
	for _, h := range mdx.Headings(content) {
		if h.Level == 1 {
			return h.Text
		}
	}
	return ""
}
