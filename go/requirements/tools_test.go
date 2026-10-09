package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// withJSTools gives repo the real eslint, prettier, and vitest installs a
// consumer repo has, by linking the pinned tests/tools/node_modules into it.
// Run `sh tests/tools/install.sh` once to install them.
func withJSTools(t *testing.T, repo string) {
	t.Helper()
	offMachine(t)
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
	offMachine(t)
	if _, err := exec.LookPath("swiftlint"); err != nil {
		t.Fatal("swiftlint is not installed; install it from https://github.com/realm/SwiftLint")
	}
}

// pathWithout returns a PATH setting that holds every real tool on the
// caller's PATH except names, so a run meets a machine without them. A
// plain /usr/bin:/bin is not enough: Linux runners keep Go and Node there.
func pathWithout(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		linkTools(dir, d, names)
	}
	return "PATH=" + dir
}

// linkTools links each entry of src into dir, except names and entries an
// earlier PATH folder already supplied.
func linkTools(dir, src string, names []string) {
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		link := filepath.Join(dir, e.Name())
		if _, err := os.Lstat(link); err == nil || slices.Contains(names, e.Name()) {
			continue
		}
		_ = os.Symlink(filepath.Join(src, e.Name()), link)
	}
}

// offMachine skips the test in a quick local run (FITNESS_QUICK set): it
// reaches GitHub, the npm registry, or a real external tool. Every CI smoke
// job still runs it.
func offMachine(t *testing.T) {
	t.Helper()
	installer(t)
	if os.Getenv("FITNESS_QUICK") != "" {
		t.Skip("quick run: GitHub, network, and real-tool tests run in CI")
	}
}
