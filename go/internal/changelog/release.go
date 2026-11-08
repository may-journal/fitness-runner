package changelog

import (
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// releaseTagGlob matches full release tags, such as v1.14.0, and not the
// moving major tag v1 or the Go module tag go/v1.14.0.
const releaseTagGlob = "v[0-9]*.[0-9]*.[0-9]*"

// PreviousTag is the newest release tag reachable from tag's parent, or "" for
// a first release. A missing tag or a shallow clone is an error.
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

// StagedReleasedEdits names each entry file the index edits or deletes that
// the newest release tag reachable from HEAD already holds, with that tag. It
// skips an edit that only drops front matter.
func StagedReleasedEdits(root string) ([]string, error) {
	tag, err := headReleaseTag(root)
	if err != nil || tag == "" {
		return nil, err
	}
	patches := filePatches(gitOut(root, "diff", "--cached", "--no-renames", "--no-color", "--no-ext-diff", "--src-prefix=a/", "--dst-prefix=b/", "--diff-filter=MD", "--", EntryDir))
	edited, err := shipped(root, tag, slices.Sorted(maps.Keys(patches)))
	var errs []string
	for _, p := range edited {
		if !DropsOnlyFrontMatter(patches[p]) {
			errs = append(errs, ReleasedEditError(p, tag))
		}
	}
	return errs, err
}

// shipped keeps the entry files tag holds, asking git about all of them in
// one call that reads no file. Its failure is the error.
func shipped(root, tag string, paths []string) ([]string, error) {
	objects := make([]string, len(paths))
	for i, p := range paths {
		objects[i] = tag + ":" + p
	}
	held, err := gitx.HasAll(root, objects)
	return slices.DeleteFunc(paths, func(p string) bool { return !held[tag+":"+p] }), err
}

// filePatches splits a unified diff into each entry file's own patch, read
// from one git call instead of one per file.
func filePatches(diff string) map[string]string {
	patches := map[string]string{}
	for _, patch := range strings.SplitAfter(diff, "\ndiff --git ") {
		patch = strings.TrimSuffix(strings.TrimPrefix(patch, "diff --git "), "\ndiff --git ")
		names, _, _ := strings.Cut(strings.TrimPrefix(patch, "a/"), "\n")
		if p, _, ok := strings.Cut(names, " b/"); ok && IsEntryPath(p) {
			patches[p] = "diff --git " + patch
		}
	}
	return patches
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
	paths, err := AddedPaths(root, since, tag)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, p := range paths {
		text, err := gitRun(root, "show", tag+":"+p)
		if err != nil {
			return nil, err
		}
		entries = append(entries, Entry{Path: p, Text: text})
	}
	SortEntries(entries)
	return entries, nil
}

// AddedPaths returns the paths of the entry files ref holds that since
// lacked, oldest name first. An empty since lists every entry file ref holds.
func AddedPaths(root, since, ref string) ([]string, error) {
	args := []string{"diff", "--name-only", "--no-renames", "--diff-filter=A", since, ref, "--", EntryDir}
	if since == "" {
		args = []string{"ls-tree", "--name-only", ref, EntryDir + "/"}
	}
	list, err := gitRun(root, args...)
	return entryPaths(list), err
}

// FetchCommit makes commit sha readable in root, fetching it from origin
// when the clone lacks it, such as a branch pushed after checkout.
func FetchCommit(root, sha string) error {
	if _, err := gitRun(root, "cat-file", "-e", sha+"^{commit}"); err == nil {
		return nil
	}
	_, err := gitRun(root, "fetch", "--quiet", "origin", sha)
	return err
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
