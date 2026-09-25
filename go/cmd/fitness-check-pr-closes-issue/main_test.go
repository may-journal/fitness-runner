package main

import (
	"reflect"
	"testing"
)

const rule1 = "PR description has no GitHub closing keyword (close/fix/resolve + #NN) — every PR must close at least one issue on merge"

func rule2(n int) string {
	return "PR says it implements #" + itoa(n) + " but no closing keyword (close/fix/resolve + #" + itoa(n) + ") closes it"
}

// itoa is a tiny local int-to-string so the expected messages read plainly.
func itoa(n int) string {
	return map[int]string{5: "5", 6: "6", 12: "12", 63: "63", 67: "67"}[n]
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{name: "closes passes", body: "> pitch\n\nCloses #12", want: nil},
		{name: "fixes passes", body: "Fixes #12", want: nil},
		{name: "resolves passes", body: "Resolves #12", want: nil},
		{name: "past-tense verbs pass", body: "closed #12, fixed #5, resolved #6", want: nil},
		{name: "colon form passes", body: "Fixes: #12", want: nil},
		{name: "references only fail rule 1", body: "addresses #12 and part of #6", want: []string{rule1}},
		{name: "no issue mention fails rule 1", body: "Just prose, nothing linked.", want: []string{rule1}},
		{
			name: "implements without closure fails both rules",
			body: "Implements #67",
			want: []string{rule1, rule2(67)},
		},
		{name: "implements with closure passes", body: "Implements #67\n\nCloses #67", want: nil},
		{
			name: "multiple implemented, one unclosed fails on that one",
			body: "Implements #5. Implements #6.\n\nCloses #5.",
			want: []string{rule2(6)},
		},
		{name: "plan line with closure passes", body: "Plan #63 for context.\n\nResolves #63.", want: nil},
		{
			name: "plan line without closure fails both rules",
			body: "Plan #63 for context.",
			want: []string{rule1, rule2(63)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validate(tc.body)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("validate(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// TestRunWiring confirms run threads the context-inline body through validate;
// the inert no-input path is bodycheck's own concern and is tested there.
func TestRunWiring(t *testing.T) {
	t.Setenv("FITNESS_CTX_MESSAGE", "> pitch\n\nCloses #1")
	if res, err := run("", nil); err != nil || !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("valid PR: got %+v, err %v", res, err)
	}

	t.Setenv("FITNESS_CTX_MESSAGE", "addresses #1 only")
	if res, err := run("", nil); err != nil || res.Ok {
		t.Fatalf("references-only PR: expected failure, got %+v, err %v", res, err)
	}
}
