package main

import "testing"

// TestRunWiresPRSpec confirms run threads a body through prSpec: a
// template-shaped PR description passes and a shapeless one fails. The
// validation rules themselves are covered in internal/mdtemplate.
func TestRunWiresPRSpec(t *testing.T) {
	const validPR = "> Ship the thing so a reader gets the value.\n\n" +
		"## Background\n\nwhy this exists\n\n## Changelog\n\n- Feat: the new behavior\n"

	t.Setenv("FITNESS_CTX_MESSAGE", validPR)
	if res, err := run("", nil); err != nil || !res.Ok {
		t.Fatalf("valid PR: got %+v, err %v", res, err)
	}

	t.Setenv("FITNESS_CTX_MESSAGE", "no summary, no sections")
	if res, err := run("", nil); err != nil || res.Ok {
		t.Fatalf("invalid PR: expected failure, got %+v, err %v", res, err)
	}
}
