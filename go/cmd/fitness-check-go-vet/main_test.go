package main

import (
	"strings"
	"testing"
)

func TestVetFindsProblems(t *testing.T) {
	needGo(t)
	root := t.TempDir()
	write(t, root, "go.mod", "module m\n\ngo 1.21\n")
	write(t, root, "m.go", "package m\n\nimport \"fmt\"\n\nfunc F() { fmt.Printf(\"%d\\n\", \"x\") }\n")
	res, err := run(root, nil)
	if err != nil || res.Ok {
		t.Fatalf("run = %+v, %v; want a failure", res, err)
	}
	if !strings.Contains(strings.Join(res.Errors, "\n"), "m.go") {
		t.Errorf("errors should name the file: %v", res.Errors)
	}
}

func TestVetPassesCleanModule(t *testing.T) {
	needGo(t)
	root := t.TempDir()
	write(t, root, "go.mod", "module m\n\ngo 1.21\n")
	write(t, root, "m.go", "package m\n\nfunc F() int { return 1 }\n")
	res, err := run(root, nil)
	if err != nil || !res.Ok || res.FilesChecked != 1 {
		t.Errorf("run = %+v, %v; want a pass over 1 module", res, err)
	}
}
