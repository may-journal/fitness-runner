package main

import "testing"

func TestAmendArgs(t *testing.T) {
	cases := map[string]bool{
		"git commit --amend --no-edit": true,
		"git commit -m fix(x): y":      false,
		"git commit --amended-note":    false,
		"":                             false,
	}
	for args, want := range cases {
		if got := amendArgs(args); got != want {
			t.Errorf("amendArgs(%q) = %v, want %v", args, got, want)
		}
	}
}
