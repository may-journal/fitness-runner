package mdtemplate

import (
	"strings"
	"testing"
)

var planSpec = Spec{
	Noun:        "plan",
	Placeholder: "REPLACE-ME",
	Sections: []Section{
		{Heading: "Background", Requires: Prose},
		{Heading: "What needs to happen", Requires: Checklist},
	},
	BanOpenQuestions: true,
}

var prSpec = Spec{
	Noun:        "PR",
	Placeholder: "REPLACE-ME",
	Sections: []Section{
		{Heading: "Background", Requires: Prose},
		{Heading: "Changelog", Requires: Bullet},
	},
}

const validPlan = `> A real one-line pitch a human would read.

## Background

Some context that explains the work in a sentence or two.

## What needs to happen

1. [ ] the first step
2. [ ] the second step
`

const validPR = `> Ship the thing so a reader gets the value.

## Background

Why this change exists and the context a reviewer needs.

## Changelog

- Feat: the new behavior in one line.
`

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		spec    Spec
		body    string
		ok      bool
		wantErr string // a substring that must appear in one error (when !ok)
	}{
		{"valid plan passes", planSpec, validPlan, true, ""},
		{"valid PR passes", prSpec, validPR, true, ""},
		{
			"missing background fails",
			planSpec, "> pitch\n\n## What needs to happen\n\n- [ ] step\n",
			false, "Missing `## Background` section",
		},
		{
			"missing checklist fails",
			planSpec, "> pitch\n\n## Background\n\ntext\n\n## What needs to happen\n\njust prose, no boxes\n",
			false, "`## What needs to happen` has no checklist item",
		},
		{
			"PR changelog needs a bullet",
			prSpec, "> pitch\n\n## Background\n\ntext\n\n## Changelog\n\njust prose, no bullets\n",
			false, "`## Changelog` has no list item",
		},
		{
			"extra H2 fails with plan noun and section list",
			planSpec, validPlan + "\n## Timeline\n\nnext week\n",
			false, "Unexpected `## Timeline` section — a plan has only Background and What needs to happen",
		},
		{
			"extra H2 fails with PR noun and section list",
			prSpec, validPR + "\n## Screenshots\n\nnone\n",
			false, "Unexpected `## Screenshots` section — a PR has only Background and Changelog",
		},
		{
			"open questions fails with pointed message when banned",
			planSpec, validPlan + "\n## Open Questions\n\n- what about X?\n",
			false, errOpenQ,
		},
		{
			"open questions is just an extra section when not banned",
			prSpec, validPR + "\n## Open Questions\n\n- what about X?\n",
			false, "Unexpected `## Open Questions` section",
		},
		{
			"placeholder pitch fails",
			planSpec, "> REPLACE-ME\n\n## Background\n\ntext\n\n## What needs to happen\n\n- [ ] step\n",
			false, errPlaceholder,
		},
		{
			"missing pitch fails",
			planSpec, "## Background\n\ntext\n\n## What needs to happen\n\n- [ ] step\n",
			false, errNoPitch,
		},
		{
			"heading and checkbox inside a fence do not count",
			planSpec, "> pitch\n\n## Background\n\n```\n## What needs to happen\n- [ ] fake\n```\n\n## What needs to happen\n\n- [ ] real\n",
			true, "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := Validate(tc.body, tc.spec)
			if ok := len(errs) == 0; ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", ok, tc.ok, errs)
			}
			if !tc.ok && !anyContains(errs, tc.wantErr) {
				t.Fatalf("errors %v missing %q", errs, tc.wantErr)
			}
		})
	}
}

func anyContains(errs []string, want string) bool {
	for _, e := range errs {
		if want != "" && strings.Contains(e, want) {
			return true
		}
	}
	return false
}
