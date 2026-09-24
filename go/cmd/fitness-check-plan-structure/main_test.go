package main

import "testing"

// TestRunWiresPlanSpec confirms run threads a body through planSpec: a
// template-shaped plan passes and a shapeless one fails. The validation rules
// themselves are covered in internal/mdtemplate.
func TestRunWiresPlanSpec(t *testing.T) {
	const validPlan = "> A real one-line pitch a human would read.\n\n" +
		"## Background\n\ncontext\n\n## What needs to happen\n\n- [ ] step\n"

	t.Setenv("FITNESS_CTX_MESSAGE", validPlan)
	if res, err := run("", nil); err != nil || !res.Ok {
		t.Fatalf("valid plan: got %+v, err %v", res, err)
	}

	t.Setenv("FITNESS_CTX_MESSAGE", "no pitch, no sections")
	if res, err := run("", nil); err != nil || res.Ok {
		t.Fatalf("invalid plan: expected failure, got %+v, err %v", res, err)
	}
}
