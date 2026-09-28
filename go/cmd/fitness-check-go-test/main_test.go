package main

import (
	"strings"
	"testing"
)

func TestFailingTestIsReported(t *testing.T) {
	needGo(t)
	root := t.TempDir()
	write(t, root, "go.mod", "module m\n\ngo 1.21\n")
	write(t, root, "m_test.go", "package m\n\nimport \"testing\"\n\nfunc TestBroken(t *testing.T) { t.Error(\"boom\") }\n")
	res, err := run(root, nil)
	if err != nil || res.Ok {
		t.Fatalf("run = %+v, %v; want a failure", res, err)
	}
	joined := strings.Join(res.Errors, "\n")
	if !strings.Contains(joined, "TestBroken") || !strings.Contains(joined, "boom") {
		t.Errorf("errors should name the test and its message: %v", res.Errors)
	}
}

func TestFailureLine(t *testing.T) {
	for _, l := range []string{"--- FAIL: TestX (0.00s)", "FAIL\tm\t0.1s", "panic: oops", "    m_test.go:5: boom"} {
		if !failureLine(l) {
			t.Errorf("failureLine(%q) = false", l)
		}
	}
	if failureLine("ok  \tm\t0.1s") || failureLine("=== RUN   TestX") {
		t.Error("passing lines must not count as failures")
	}
}
