package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

const doc = "# Changelog\n\n## Changes\n\n### 2020.01.01.0000\n\n- Feat: x\n\n### 2019.01.01.0000\n\n- Fix: y\n"

func TestRestampFirstHeadingOnly(t *testing.T) {
	got := restamp(doc, "2026.07.18.1400")
	if !strings.Contains(got, "### 2026.07.18.1400") {
		t.Fatal("first heading not restamped")
	}
	if !strings.Contains(got, "### 2019.01.01.0000") {
		t.Fatal("second heading must stay untouched")
	}
	if strings.Count(got, "2026.07.18.1400") != 1 {
		t.Fatal("exactly one heading restamps")
	}
}

func TestStampOnlyWhenStaged(t *testing.T) {
	dir := gitRepo(t)
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 14, 5, 0, 0, time.Local)

	expectStamp(t, dir, now, false, "unstaged changelog must be a no-op")

	// staged: restamp + re-stage
	gitOutput(t, dir, "add", "CHANGELOG.md")
	expectStamp(t, dir, now, true, "staged changelog must restamp")
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "### 2026.07.18.1405") {
		t.Fatalf("heading not restamped: %s", raw)
	}
	if staged := gitOutput(t, dir, "diff", "--cached", "--name-only"); !strings.Contains(staged, "CHANGELOG.md") {
		t.Fatal("restamped changelog must be re-staged")
	}
	if diff := gitOutput(t, dir, "diff", "--name-only"); strings.TrimSpace(diff) != "" {
		t.Fatalf("working tree must match the index after re-stage: %s", diff)
	}
}

// expectStamp runs stamp and requires no error and the given changed flag.
func expectStamp(t *testing.T, dir string, now time.Time, want bool, label string) {
	t.Helper()
	changed, err := stamp(dir, now)
	if err != nil || changed != want {
		t.Fatalf("%s: changed=%v err=%v", label, changed, err)
	}
}

// gitOutput runs git in dir and returns its output, failing on error.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func TestFormatTimestamp(t *testing.T) {
	got := formatTimestamp(time.Date(2026, 7, 18, 9, 7, 0, 0, time.Local))
	if got != "2026.07.18.0907" {
		t.Fatalf("got %s", got)
	}
}
