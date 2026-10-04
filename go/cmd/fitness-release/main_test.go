package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "version.txt"), []byte("1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_REF_NAME", "v1.0.0")
	for _, command := range []string{"tag", "verify-tag"} {
		if err := run([]string{command, "--root", root}); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
	}
}

func TestReleaseCommandErrors(t *testing.T) {
	root := t.TempDir()
	cases := [][]string{nil, {"unknown"}, {"tag", "--invalid"}, {"tag", "--root", root}, {"verify-tag", "--root", root}, {"assemble", "--root", root}, {"smoke", "--root", root}, {"verify-download", "--root", root}, {"bundle-hashes", "--root", root}}
	for _, args := range cases {
		if err := run(args); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}

func TestReleaseReportsActionOutcomes(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "version.txt"), []byte("1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_REF_NAME", "v1.0.0")
	if status := command([]string{"verify-tag", "--root", root}); status != 0 {
		t.Fatalf("verify-tag status %d", status)
	}
	requireSummary(t, summary, "✅ fitness-release verify-tag")
	t.Setenv("GITHUB_REF_NAME", "go/v0.1.2")
	if status := command([]string{"verify-tag", "--root", root}); status != 1 {
		t.Fatalf("mismatched tag status %d", status)
	}
	requireSummary(t, summary, "❌ fitness-release verify-tag")
}

func TestReleaseLocalDoesNotWriteSummary(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "false")
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	if status := command(nil); status != 1 {
		t.Fatalf("status %d", status)
	}
	if _, err := os.Stat(summary); !os.IsNotExist(err) {
		t.Fatalf("local command wrote a summary: %v", err)
	}
}

func TestReleaseFailureSurvivesSummaryError(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", t.TempDir())
	if status := command(nil); status != 1 {
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
