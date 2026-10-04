package main

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestRunnerEnvironment(t *testing.T) {
	cases := []struct {
		name, path, want string
	}{
		{"project tools", "/project tools/bin", "PATH=/verified bundle/bin" + string(os.PathListSeparator) + "/project tools/bin"},
		{"empty", "", "PATH=/verified bundle/bin"},
		{"unset", "", "PATH=/verified bundle/bin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", tc.path)
			t.Setenv("FITNESS_PROJECT_SETTING", "preserved")
			if tc.name == "unset" {
				os.Unsetenv("PATH")
			}
			before := os.Environ()
			env := runnerEnvironment("/verified bundle/bin")
			assertRunnerEnvironment(t, env, before, tc.want)
		})
	}
}

func assertRunnerEnvironment(t *testing.T, env, before []string, want string) {
	t.Helper()
	if strings.Count("\n"+strings.Join(env, "\n"), "\nPATH=") != 1 {
		t.Fatalf("expected one PATH entry: %q", env)
	}
	if !slices.Contains(env, want) {
		t.Fatalf("missing %q", want)
	}
	if !slices.Contains(env, "FITNESS_PROJECT_SETTING=preserved") {
		t.Fatal("project setting lost")
	}
	if !slices.Equal(before, os.Environ()) {
		t.Fatal("caller environment changed")
	}
}
