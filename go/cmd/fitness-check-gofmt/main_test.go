package main

import (
	"strings"
	"testing"
)

func TestUnformattedFileIsReported(t *testing.T) {
	needGo(t)
	root := t.TempDir()
	write(t, root, "ok.go", "package m\n\nfunc F() int { return 1 }\n")
	write(t, root, "bad.go", "package m\nfunc   G( ) int {return 2}\n")
	write(t, root, "x/testdata/fixture.go", "package   fixture\n")
	res, err := run(root, nil)
	if err != nil || res.Ok {
		t.Fatalf("run = %+v, %v; want a failure", res, err)
	}
	if len(res.Errors) != 1 || !strings.HasPrefix(res.Errors[0], "bad.go:") {
		t.Errorf("errors = %v; want only bad.go", res.Errors)
	}
}
