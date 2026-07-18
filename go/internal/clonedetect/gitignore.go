package clonedetect

import (
	"errors"
	"os/exec"
	"strings"
)

// GitIgnored returns the subset of relPaths (slash-separated, relative to
// root) that git ignores, asked in one batched `git check-ignore --stdin
// -z` call. Exit status 1 (no path ignored) is a normal empty answer; any
// other failure — not a git work tree, git missing — returns nil, meaning
// filter nothing, matching jscpd's behavior outside a repo.
func GitIgnored(root string, relPaths []string) map[string]bool {
	if len(relPaths) == 0 {
		return map[string]bool{}
	}
	cmd := exec.Command("git", "-C", root, "check-ignore", "--stdin", "-z")
	cmd.Stdin = strings.NewReader(strings.Join(relPaths, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return map[string]bool{}
		}
		return nil
	}
	ignored := make(map[string]bool)
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			ignored[p] = true
		}
	}
	return ignored
}
