// Package gitx wraps the git invocations the runner and checks share.
package gitx

import (
	"os/exec"
	"strings"
)

// StagedFiles returns `git diff --cached --name-only` for root with
// node_modules entries removed; nil on any git failure (not a repo, no git).
func StagedFiles(root string) []string {
	out, err := exec.Command("git", "-C", root, "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" || strings.Contains(line, "node_modules") {
			continue
		}
		files = append(files, line)
	}
	return files
}

// HeadMessage returns the full HEAD commit message; empty on failure.
func HeadMessage(root string) string {
	out, err := exec.Command("git", "-C", root, "log", "-1", "--pretty=%B").Output()
	if err != nil {
		return ""
	}
	return string(out)
}
