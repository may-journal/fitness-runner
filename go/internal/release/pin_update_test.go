package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func preparePinsFixture(t *testing.T) Config {
	t.Helper()
	c := pinFixture(t)
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Changelog\n\n## Changes\n\n### 2026.10.04.1140\n\n- Existing notes\n")
	if err := c.saveMetadata(Metadata{Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPreparePinsProducesFilesAndOutputs(t *testing.T) {
	c := preparePinsFixture(t)
	output := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", output)
	if err := c.PreparePins(); err != nil {
		t.Fatal(err)
	}
	assertRootPins(t, c, "1.0.0")
	text := pinTestRead(t, filepath.Join(c.Root, "CHANGELOG.md"))
	if !strings.Contains(text, pinNotes("1.0.0")) || !strings.Contains(text, "- Existing notes") {
		t.Fatalf("changelog: %s", text)
	}
	if got := pinTestRead(t, output); got != "version=1.0.0\nchanged=true\n" {
		t.Fatalf("step outputs: %s", got)
	}
}

func pinTestRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPreparePinsRepeatDoesNotAddChangelog(t *testing.T) {
	c := preparePinsFixture(t)
	if err := c.PreparePins(); err != nil {
		t.Fatal(err)
	}
	before := pinTestRead(t, filepath.Join(c.Root, "CHANGELOG.md"))
	output := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", output)
	if err := c.PreparePins(); err != nil {
		t.Fatal(err)
	}
	if after := pinTestRead(t, filepath.Join(c.Root, "CHANGELOG.md")); after != before {
		t.Fatal("repeat changed the changelog")
	}
	if got := pinTestRead(t, output); got != "version=1.0.0\n" {
		t.Fatalf("repeat reported changes: %s", got)
	}
}

func TestPreparePinsRejectsBadChangelogBeforeWriting(t *testing.T) {
	c := preparePinsFixture(t)
	path := filepath.Join(c.Root, "action.yml")
	before := pinTestRead(t, path)
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Missing Changes section\n")
	requireError(t, c.PreparePins())
	if pinTestRead(t, path) != before {
		t.Fatal("wrote pins before validating the changelog")
	}
}

func TestPreparePinsRejectsDowngradeBeforeWriting(t *testing.T) {
	c := preparePinsFixture(t)
	path := filepath.Join(c.Root, "action.yml")
	before := pinTestRead(t, path)
	testWrite(t, filepath.Join(c.Root, "README.md"), "Pin a version with `@v2.0.0`.\n")
	requireError(t, c.PreparePins())
	if pinTestRead(t, path) != before {
		t.Fatal("wrote pins before detecting a downgrade")
	}
}

func TestPreparePinsRejectsInvalidMetadata(t *testing.T) {
	c := preparePinsFixture(t)
	if err := c.saveMetadata(Metadata{Version: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	requireError(t, c.PreparePins())
}
