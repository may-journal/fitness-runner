package main

import (
	"strings"
	"testing"
)

func TestJudge(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		ok   bool
	}{
		{"no trailer passes", "feat(x): do a thing\n\nSome body.\n", true},
		{"valid trailer passes", "feat(x): do a thing\n\nBody.\n\nPlan #63\n", true},
		{"body prose is not a trailer", "feat(x): do a thing\n\nPlan the rollout in 3 steps.\n", true},
		{"missing hash fails", "feat(x): do a thing\n\nPlan 63\n", false},
		{"lowercase fails", "feat(x): do a thing\n\nplan #63\n", false},
		{"colon form fails", "feat(x): do a thing\n\nPlan: 63\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.msg)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if !tc.ok && !strings.Contains(res.Errors[0], "Plan #<number>") {
				t.Fatalf("error should hint the fix: %q", res.Errors[0])
			}
		})
	}
}
