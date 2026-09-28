package main

import (
	"reflect"
	"testing"
)

func TestPlanNumbers(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"single", "feat: x\n\nPlan #12\n", []string{"12"}},
		{"case and spacing", "chore: y\n\nplan #  7", []string{"7"}},
		{"distinct in first-seen order", "Plan #3\nPlan #9\nPlan #3", []string{"3", "9"}},
		{"none", "just a body, no plan reference", nil},
		{"not a bare number", "we plan 5 steps", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := planNumbers(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("planNumbers = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAllChoreDocsSubjects(t *testing.T) {
	cases := []struct {
		name     string
		subjects []string
		want     bool
	}{
		{"all chore/docs", []string{"chore(ci): x", "docs: y"}, true},
		{"one feat", []string{"chore: x", "feat: y"}, false},
		{"empty is vacuously true", nil, true},
		{"fix is not exempt", []string{"fix: bug"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allChoreDocsSubjects(tc.subjects); got != tc.want {
				t.Fatalf("allChoreDocsSubjects(%v) = %v, want %v", tc.subjects, got, tc.want)
			}
		})
	}
}
