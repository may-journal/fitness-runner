package gitx

import (
	"os"
	"strings"
)

// ScopeBase is the commit a scoped run's changes replace: the merge base
// with a pull request's base branch, the parent of an amended commit, or
// HEAD for staged files. It is empty when there is none, as in a first
// commit.
func ScopeBase(root string) string {
	r := Repo(root)
	var out string
	switch {
	case os.Getenv("GITHUB_BASE_REF") != "":
		out, _ = r.run("merge-base", "HEAD", "origin/"+os.Getenv("GITHUB_BASE_REF"))
	case os.Getenv("FITNESS_COMMIT_AMEND") == "1":
		out, _ = r.run("rev-parse", "-q", "--verify", "HEAD^")
	default:
		out, _ = r.run("rev-parse", "-q", "--verify", "HEAD")
	}
	return strings.TrimSpace(out)
}

// Show returns the content of a git object such as rev:path, and whether
// git found it. A rev of ":" names the index.
func Show(root, object string) (string, bool) {
	out, err := Repo(root).run("show", object)
	return out, err == nil
}

// FileAt returns the file at path, slash-separated from the repo root, as
// commit rev holds it. It is empty when rev is empty or lacks the file.
func FileAt(root, rev, path string) string {
	if rev == "" {
		return ""
	}
	out, _ := Repo(root).run("show", rev+":"+path)
	return out
}
