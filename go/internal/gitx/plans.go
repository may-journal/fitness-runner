package gitx

import (
	"regexp"
	"strings"
)

// planRe matches the three Plan trailer spellings: `Plan #NN`,
// `Plan owner/repo#NN` and `Plan https://github.com/owner/repo/issues/NN`.
var planRe = regexp.MustCompile(`(?i)plan (?:#\s*(\d+)|([\w.-]+)/([\w.-]+)#(\d+)|https://github\.com/([\w.-]+)/([\w.-]+)/issues/(\d+))`)

// PlanRef is one Plan a trailer names. Owner and Repo are empty for a bare
// `Plan #NN`, which names an issue of the pushing repo.
type PlanRef struct {
	Owner  string
	Repo   string
	Number string
}

// Qualified reports whether the trailer named the Plan's repo.
func (p PlanRef) Qualified() bool {
	return p.Repo != ""
}

// Slug is the named repo as owner/repo, or "" for a bare trailer.
func (p PlanRef) Slug() string {
	if !p.Qualified() {
		return ""
	}
	return p.Owner + "/" + p.Repo
}

// String spells the Plan as a trailer would: `#NN` or `owner/repo#NN`.
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
	seen := map[string]bool{}
	var plans []PlanRef
	for _, m := range planRe.FindAllStringSubmatch(text, -1) {
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
