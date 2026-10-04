package gocoverage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNamedWrapperFails(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wrapper_test.go", `package sample
import "testing"
func TestGrouped(t *testing.T) {
	t.Run("hidden duplicate", hiddenDuplicate)
}
func hiddenDuplicate(t *testing.T) {}
`)
	failures, err := namedWrapperFailures([]Package{{Dir: root, TestGoFiles: []string{"wrapper_test.go"}}})
	if err != nil || len(failures) != 1 {
		t.Fatalf("failures = %v, error = %v", failures, err)
	}
	if !strings.Contains(failures[0], "TestGrouped passes named helper hiddenDuplicate") {
		t.Fatal(failures)
	}
}

func TestAnonymousWrapperIsAllowed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "table_test.go", `package sample
import "testing"
func TestTable(t *testing.T) {
	t.Run("case", func(t *testing.T) {})
}
`)
	failures, err := scanNamedWrappers(filepath.Join(root, "table_test.go"))
	if err != nil || len(failures) != 0 {
		t.Fatalf("failures = %v, error = %v", failures, err)
	}
}
