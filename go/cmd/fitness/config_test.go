package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// checkDirectories lists the sibling fitness-check-* directories.
func checkDirectories(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join("..", "fitness-check-*"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no check directories found: %v", err)
	}
	return dirs
}

func TestAllChecksHasEveryCheckBinary(t *testing.T) {
	for _, d := range checkDirectories(t) {
		name := filepath.Base(d)[len("fitness-check-"):]
		if !slices.Contains(allChecks, name) {
			t.Errorf("allChecks is missing %s; every check runs in every repo", name)
		}
	}
	for _, name := range allChecks {
		if _, err := os.Stat(filepath.Join("..", "fitness-check-"+name)); err != nil {
			t.Errorf("allChecks names %s, which has no binary", name)
		}
	}
}

func TestDisabledSet(t *testing.T) {
	if len(disabledSet(nil)) != 0 {
		t.Error("no config must disable nothing")
	}
	set := disabledSet(&conf.Config{DisabledChecks: []string{"jscpd"}})
	if !set["jscpd"] || len(set) != 1 {
		t.Errorf("disabledSet = %v", set)
	}
}
