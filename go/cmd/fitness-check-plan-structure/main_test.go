package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validPlan = `> A real one-line pitch a human would read.

## Background

Some context that explains the work in a sentence or two.

## What needs to happen

1. [ ] the first step
2. [ ] the second step
`

func TestJudge(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		ok      bool
		wantErr string // a substring that must appear in one error (when !ok)
	}{
		{"valid plan passes", validPlan, true, ""},
		{
			"missing background fails",
			"> pitch\n\n## What needs to happen\n\n- [ ] step\n",
			false, errNoBackground,
		},
		{
			"missing checklist fails",
			"> pitch\n\n## Background\n\ntext\n\n## What needs to happen\n\njust prose, no boxes\n",
			false, errNoChecklist,
		},
		{
			"extra H2 fails",
			validPlan + "\n## Timeline\n\nnext week\n",
			false, "Unexpected `## Timeline`",
		},
		{
			"open questions fails with pointed message",
			validPlan + "\n## Open Questions\n\n- what about X?\n",
			false, errOpenQ,
		},
		{
			"placeholder pitch fails",
			"> REPLACE-ME\n\n## Background\n\ntext\n\n## What needs to happen\n\n- [ ] step\n",
			false, errPlaceholder,
		},
		{
			"missing pitch fails",
			"## Background\n\ntext\n\n## What needs to happen\n\n- [ ] step\n",
			false, errNoPitch,
		},
		{
			"heading and checkbox inside a fence do not count",
			"> pitch\n\n## Background\n\n```\n## What needs to happen\n- [ ] fake\n```\n\n## What needs to happen\n\n- [ ] real\n",
			true, "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.body)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if !tc.ok && !containsSubstr(res.Errors, tc.wantErr) {
				t.Fatalf("errors %v missing %q", res.Errors, tc.wantErr)
			}
		})
	}
}

func TestRunReadsBodyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(path, []byte(validPlan), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := run("", []string{"--body-file", path})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected pass, got %+v", res)
	}
}

func TestRunReadsContextMessage(t *testing.T) {
	t.Setenv("FITNESS_CTX_MESSAGE", validPlan)
	res, err := run("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected pass, got %+v", res)
	}
}

func TestRunPassesWithNoInput(t *testing.T) {
	if err := os.Unsetenv("FITNESS_CTX_MESSAGE"); err != nil {
		t.Fatal(err)
	}
	res, err := run("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("expected inert pass with 0 files, got %+v", res)
	}
}

func containsSubstr(errs []string, want string) bool {
	for _, e := range errs {
		if want != "" && contains(e, want) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
