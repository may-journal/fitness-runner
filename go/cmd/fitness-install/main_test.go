package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
)

func TestRunHelpAndInvalidOptions(t *testing.T) {
	cases := []struct {
		args   []string
		status int
	}{
		{[]string{"--help"}, 0},
		{[]string{"--unknown"}, 1},
		{[]string{"--install-only", "--", "--all"}, 1},
		{[]string{"--version", "invalid"}, 1},
	}
	for _, tc := range cases {
		if status := run(tc.args); status != tc.status {
			t.Fatalf("%q: status %d, want %d", tc.args, status, tc.status)
		}
	}
}

func TestActionRejectsInvalidInput(t *testing.T) {
	t.Setenv("FITNESS_INSTALL_ONLY", "invalid")
	if status := run([]string{"--github-action"}); status != 1 {
		t.Fatalf("status %d", status)
	}
}

func TestInstalledActionOutputs(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "outputs")
	path := filepath.Join(directory, "path")
	t.Setenv("GITHUB_OUTPUT", output)
	t.Setenv("GITHUB_PATH", path)
	options := distribution.Options{InstallOnly: true, GitHubAction: true}
	if status := runInstalled(directory, options); status != 0 {
		t.Fatalf("status %d", status)
	}
	assertFile(t, output, "bin="+directory+"\n")
	assertFile(t, path, directory+"\n")
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s: %q, want %q", path, data, want)
	}
}

func TestInstalledFailures(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "missing", "output"))
	t.Setenv("GITHUB_PATH", filepath.Join(t.TempDir(), "path"))
	if status := runInstalled(t.TempDir(), distribution.Options{GitHubAction: true}); status != 1 {
		t.Fatalf("action status %d", status)
	}
	if status := runInstalled(t.TempDir(), distribution.Options{}); status != 1 {
		t.Fatalf("missing runner status %d", status)
	}
}

func TestHookRefusesOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "missing"))
	if status := runInstalled(t.TempDir(), distribution.Options{InstallHook: true}); status != 1 {
		t.Fatalf("hook status %d", status)
	}
}
