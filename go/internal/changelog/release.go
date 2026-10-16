package changelog

import (
	"fmt"
	"os/exec"
	"strings"
)

// releaseTagGlob matches full release tags, such as v1.14.0, and not the
// moving major tag v1 or the Go module tag go/v1.14.0.
const releaseTagGlob = "v[0-9]*.[0-9]*.[0-9]*"

// PreviousTag is the newest release tag reachable from tag's parent. It is
// "" only for a first release: tag's commit has no parent, or no release
// tag comes before it. A missing tag or a shallow clone, which hides the
// history, is an error rather than a first release.
func PreviousTag(root, tag string) (string, error) {
	shallow, err := gitRun(root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(shallow) == "true" {
		return "", fmt.Errorf("the clone is shallow, so the previous release tag is unknown; fetch the full history and tags, such as with fetch-depth: 0")
	}
	commits, err := gitRun(root, "rev-list", "--parents", "-n", "1", tag+"^{commit}")
	if err != nil {
		return "", err
	}
	if len(strings.Fields(commits)) < 2 {
		return "", nil
	}
	return NewestTag(root, tag+"^")
}

// NewestTag is the newest release tag reachable from ref, or "" when none is.
func NewestTag(root, ref string) (string, error) {
	tags, err := gitRun(root, "tag", "--list", releaseTagGlob, "--merged", ref)
	if err != nil || strings.TrimSpace(tags) == "" {
		return "", err
	}
	newest, err := gitRun(root, "describe", "--tags", "--abbrev=0", "--match", releaseTagGlob, ref)
	return strings.TrimSpace(newest), err
}

// StagedReleasedEdits names each entry file the index edits or deletes
// that the newest release tag reachable from HEAD already holds, with that
// tag. A repo without release tags has none.
func StagedReleasedEdits(root string) ([]string, error) {
	tag, err := headReleaseTag(root)
	if err != nil || tag == "" {
		return nil, err
	}
	var errs []string
	for _, p := range entryPaths(gitOut(root, "diff", "--cached", "--name-only", "--no-renames", "--diff-filter=MD", "--", EntryDir)) {
		if exec.Command("git", "-C", root, "cat-file", "-e", tag+":"+p).Run() == nil {
			errs = append(errs, ReleasedEditError(p, tag))
		}
	}
	return errs, nil
}

// headReleaseTag is the newest release tag reachable from HEAD, or "" before
// the first commit or release.
func headReleaseTag(root string) (string, error) {
	if gitOut(root, "rev-parse", "-q", "--verify", "HEAD") == "" {
		return "", nil
	}
	return NewestTag(root, "HEAD")
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

// ReleasedEditError says the entry file at p, which release tag already
// holds, must stay as it shipped.
func ReleasedEditError(p, tag string) string {
	return p + " shipped in " + tag + ", so its notes are published; leave it as it is and add a new " + EntryDir + "/<yyyy-mm-dd-HHMM>-<change>.md entry file instead"
}
