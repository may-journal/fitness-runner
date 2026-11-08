package metadata

import (
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/adr"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

// adrShape is an ADR issue's sections, for a repo without an `ADR` issue
// template; a repo's own template replaces it. The issue's state is its
// status, so no section holds one.
var adrShape = mdtemplate.Spec{
	Noun:             "adr",
	Placeholder:      "REPLACE-ME",
	BanOpenQuestions: true,
	Sections:         adrSections(),
}

// adrSections is one prose section per ADR issue heading.
func adrSections() []mdtemplate.Section {
	var out []mdtemplate.Section
	for _, h := range adr.IssueSections {
		out = append(out, mdtemplate.Section{Heading: h})
	}
	return out
}

// adrTitle matches the `[ADR] Short decision` title form.
var adrTitle = regexp.MustCompile(`^\[ADR\] \S`)

// errADRTitle is the message for an ADR issue's title.
const errADRTitle = "the title must read `[ADR] Short decision`"

// adrErrors judges an ADR issue's title form.
func adrErrors(t target) []string {
	if !adrTitle.MatchString(t.Title) {
		return []string{errADRTitle}
	}
	return nil
}
