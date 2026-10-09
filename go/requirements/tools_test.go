package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// withJSTools gives repo the real eslint, prettier, and vitest installs a
// consumer repo has, by linking the pinned tests/tools/node_modules into it.
// Run `sh tests/tools/install.sh` once to install them.
func withJSTools(t *testing.T, repo string) {
	t.Helper()
	installer(t)
	tools, err := filepath.Abs(filepath.Join("..", "..", "tests", "tools", "node_modules"))
	mustDo(t, err)
	if _, err := os.Stat(filepath.Join(tools, ".bin")); err != nil {
		t.Fatalf("JavaScript tools are not installed; run `sh tests/tools/install.sh`: %v", err)
	}
	mustDo(t, os.Symlink(tools, filepath.Join(repo, "node_modules")))
}

// swiftlintTool fails the test unless the real swiftlint is on PATH.
func swiftlintTool(t *testing.T) {
	t.Helper()
	installer(t)
	if _, err := exec.LookPath("swiftlint"); err != nil {
		t.Fatal("swiftlint is not installed; install it from https://github.com/realm/SwiftLint")
	}
}
