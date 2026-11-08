package metadata

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

// adrShape is an ADR issue's sections, for a repo without an `ADR` issue
// template; a repo's own template replaces it.
var adrShape = mdtemplate.Spec{
	Noun:             "adr",
	Placeholder:      "REPLACE-ME",
	BanOpenQuestions: true,
	Sections: []mdtemplate.Section{
		{Heading: "Status"}, {Heading: "Context"}, {Heading: "Decision"}, {Heading: "Consequences"},
	},
}

// adrTitle matches the `[ADR] Short decision` title form.
var adrTitle = regexp.MustCompile(`^\[ADR\] \S`)

// adrStatus matches a Status of one word, where Superseded may name the
// ADR issue that replaces it by its full URL.
var adrStatus = regexp.MustCompile(`^(?:Proposed|Accepted|Superseded(?: by https://github\.com/[\w.-]+/[\w.-]+/issues/\d+)?)$`)

// Messages for an ADR issue's title and Status.
const (
	errADRTitle  = "the title must read `[ADR] Short decision`"
	errADRStatus = "`## Status` must be one word: Proposed, Accepted or Superseded; a superseded ADR may add `by` and the full URL of the ADR issue that replaces it"
)

// adrErrors judges an ADR issue's title form and its one-word Status. A
// missing Status section is the template shape's finding, not this one.
func adrErrors(t target) []string {
	var errs []string
	if !adrTitle.MatchString(t.Title) {
		errs = append(errs, errADRTitle)
	}
	if status, ok := mdtemplate.SectionText(t.Body, "Status"); ok && !adrStatus.MatchString(strings.TrimSpace(status)) {
		errs = append(errs, errADRStatus)
	}
	return errs
}
