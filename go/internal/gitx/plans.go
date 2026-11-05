package gitx

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// repoSpellings are the two ways to name an issue of a named repo:
// `owner/repo#NN` and `https://github.com/owner/repo/issues/NN`.
const repoSpellings = `([\w.-]+)/([\w.-]+)#(\d+)|https://github\.com/([\w.-]+)/([\w.-]+)/issues/(\d+)`

// ClosingKeywords is GitHub's fixed set of nine issue-closing keywords.
const ClosingKeywords = `close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved`

// planRe matches the three Plan trailer spellings, such as `Plan #NN`. A
// trailer may put space after the `#`.
var planRe = regexp.MustCompile(`(?i)plan (?:#\s*(\d+)|` + repoSpellings + `)`)

// closedRe matches a closing keyword before an issue as GitHub reads one,
// such as `Closes #NN` or `Fixes: owner/repo#NN`, with digits right after `#`.
var closedRe = regexp.MustCompile(`(?i)\b(?:` + ClosingKeywords + `)[:\s]+(?:#(\d+)|` + repoSpellings + `)\b`)

// PlanRef is one Plan a trailer or a PR's closing keyword names. Owner and
// Repo are empty for a bare `#NN`, which names an issue of the pushing repo.
type PlanRef struct {
	Owner  string
	Repo   string
	Number string
}

// Qualified reports whether the reference named the Plan's repo.
func (p PlanRef) Qualified() bool {
	return p.Repo != ""
}

// Slug is the named repo as owner/repo, or "" for a bare reference.
func (p PlanRef) Slug() string {
	if !p.Qualified() {
		return ""
	}
	return p.Owner + "/" + p.Repo
}

// String spells the Plan as a reference would: `#NN` or `owner/repo#NN`.
func (p PlanRef) String() string {
	return p.Slug() + "#" + p.Number
}

// key is the case-insensitive identity of a Plan, so two spellings of one
// Plan count once.
func (p PlanRef) key() string {
	return strings.ToLower(p.String())
}

// PlanNumbers extracts the distinct Plans the trailers in text name, in
// first-seen order. A bare `#NN` keeps no repo, so it stays local.
func PlanNumbers(text string) []PlanRef {
	return distinctRefs(planRe, text)
}

// ClosedRefs extracts the distinct issues a PR body closes with a closing
// keyword, in first-seen order. A keyword in code or an HTML comment closes
// nothing, as on GitHub.
func ClosedRefs(body string) []PlanRef {
	return distinctRefs(closedRe, mdx.StripComments(mdx.StripCode(body)))
}

// distinctRefs reads each match of re in text into its PlanRef, keeping
// the first of each Plan.
func distinctRefs(re *regexp.Regexp, text string) []PlanRef {
	seen := map[string]bool{}
	var plans []PlanRef
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		p := planRef(m)
		if !seen[p.key()] {
			seen[p.key()] = true
			plans = append(plans, p)
		}
	}
	return plans
}

// planRef reads one planRe match into its PlanRef.
func planRef(m []string) PlanRef {
	switch {
	case m[1] != "":
		return PlanRef{Number: m[1]}
	case m[4] != "":
		return PlanRef{Owner: m[2], Repo: m[3], Number: m[4]}
	default:
		return PlanRef{Owner: m[5], Repo: m[6], Number: m[7]}
	}
}
