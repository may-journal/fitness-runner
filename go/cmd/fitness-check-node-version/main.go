// Command fitness-check-node-version validates that the machine's active
// Node version satisfies the repo's .nvmrc — the Go port of the node-version
// check. One deviation from the TypeScript original: it asks `node --version`
// for the current version (the TS check read its own runtime's
// process.version; this binary is not Node), and a missing node binary is a
// clear failure instead of an impossibility.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "node-version"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	raw, err := os.ReadFile(filepath.Join(root, ".nvmrc"))
	if err != nil {
		return checkkit.Fail(1, "missing .nvmrc"), nil
	}
	current, err := currentNodeVersion()
	if err != nil {
		return checkkit.Fail(1, "node not found on PATH. Install Node.js (e.g. via nvm) to run the node-version check"), nil
	}
	return judge(strings.TrimSpace(string(raw)), current), nil
}

// judge compares the .nvmrc requirement against the current version string,
// mirroring the TS check: major-version comparison, current >= required
// passes, and an unparsable requirement reports "NaN" exactly like
// JavaScript did.
func judge(required, current string) checkkit.Result {
	requiredMajor, requiredOk := versionMajor(required)
	currentMajor, currentOk := versionMajor(current)
	if requiredOk && currentOk && currentMajor >= requiredMajor {
		return checkkit.Pass(1)
	}
	requiredText := "NaN"
	if requiredOk {
		requiredText = fmt.Sprintf("%d", requiredMajor)
	}
	return checkkit.Fail(1, fmt.Sprintf(
		"Node %s does not satisfy .nvmrc (requires %s.x). Run: nvm use", current, requiredText))
}

var versionRe = regexp.MustCompile(`^v?(\d+)`)

// versionMajor parses the leading major version from e.g. "v24.0.0" or "24".
func versionMajor(version string) (int, bool) {
	m := versionRe.FindStringSubmatch(strings.TrimSpace(version))
	if m == nil {
		return 0, false
	}
	var major int
	if _, err := fmt.Sscanf(m[1], "%d", &major); err != nil {
		return 0, false
	}
	return major, true
}

func currentNodeVersion() (string, error) {
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
