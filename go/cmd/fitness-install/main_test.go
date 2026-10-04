package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestInstallReportsActionOutcomes(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	bin := t.TempDir()
	if status := runInstalled(bin, distribution.Options{InstallOnly: true}); status != 0 {
		t.Fatalf("install-only status %d", status)
	}
	requireSummary(t, summary, "Verified Fitness binaries installed successfully.")
	if status := runInstalled(bin, distribution.Options{}); status != 1 {
		t.Fatalf("missing executable status %d", status)
	}
	requireSummary(t, summary, "❌ fitness-install")
}

func TestInstallLocalDoesNotWriteSummary(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "false")
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	if status := runInstalled(t.TempDir(), distribution.Options{InstallOnly: true}); status != 0 {
		t.Fatalf("status %d", status)
	}
	if _, err := os.Stat(summary); !os.IsNotExist(err) {
		t.Fatalf("local command wrote a summary: %v", err)
	}
}

func TestInstallFailureSurvivesSummaryError(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", t.TempDir())
	if status := run([]string{"--unknown"}); status != 1 {
		t.Fatalf("status %d", status)
	}
}

func requireSummary(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("summary %q does not contain %q", data, want)
	}
}
