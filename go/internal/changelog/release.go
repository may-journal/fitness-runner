package changelog

import (
	"fmt"
	"os/exec"
	"strings"
)

// releaseTagGlob matches full release tags, such as v1.14.0, and not the
// moving major tag v1 or the Go module tag go/v1.14.0.
const releaseTagGlob = "v[0-9]*.[0-9]*.[0-9]*"

// PreviousTag is the newest release tag reachable from tag's parent, or ""
// for the first release.
func PreviousTag(root, tag string) string {
	return gitOut(root, "describe", "--tags", "--abbrev=0", "--match", releaseTagGlob, tag+"^")
}

// AddedSince returns the entry files tag holds that since lacked, read at
// tag, newest first. An empty since means the first release, which ships
// every entry file tag holds.
func AddedSince(root, since, tag string) ([]Entry, error) {
	args := []string{"diff", "--name-only", "--no-renames", "--diff-filter=A", since, tag, "--", EntryDir}
	if since == "" {
		args = []string{"ls-tree", "--name-only", tag, EntryDir + "/"}
	}
	list, err := gitRun(root, args...)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, p := range entryPaths(list) {
		text, err := gitRun(root, "show", tag+":"+p)
		if err != nil {
			return nil, err
		}
		entries = append(entries, Entry{Path: p, Text: text})
	}
	SortEntries(entries)
	return entries, nil
}

// gitRun returns git's output in root, or an error naming the command and
// what git said.
func gitRun(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exit.Stderr))
		}
		return "", fmt.Errorf("git %s: %v %s", strings.Join(args, " "), err, stderr)
	}
	return string(out), nil
}
