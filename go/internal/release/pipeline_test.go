package release

import (
	"os"
	"path/filepath"
	"testing"
)

func testWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func testConfig(t *testing.T) Config {
	t.Helper()
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Changelog\n\n### 2026.10.04.0037\n\n- Current release\n\n### 2026.10.03.1200\n\n- Previous release\n")
	testWrite(t, filepath.Join(c.Root, "version.txt"), "1.0.0\n")
	testWrite(t, filepath.Join(c.Out, "stale"), "old output")
	return c
}

func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}
