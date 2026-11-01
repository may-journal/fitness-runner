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

// Show returns the file at path, slash-separated from the repo root, as
// commit rev holds it, and whether rev holds it. An empty rev holds none.
func Show(root, rev, path string) (string, bool) {
	if rev == "" {
		return "", false
	}
	return show(root, rev+":"+path)
}

// FileAt is Show without the found flag: empty when rev lacks the file.
func FileAt(root, rev, path string) string {
	out, _ := Show(root, rev, path)
	return out
}

// Judged returns the file at path as a scoped run judges it. A pull request
// reads it at HEAD; a commit reads the index, which holds what it records.
func Judged(root, path string) (string, bool) {
	if os.Getenv("GITHUB_BASE_REF") != "" {
		return show(root, "HEAD:"+path)
	}
	return show(root, ":"+path)
}

func show(root, object string) (string, bool) {
	out, err := Repo(root).run("show", object)
	return out, err == nil
}

// Renames maps each path the scoped run renames since base to its path at
// base: a pull request's commits, or else the index.
func Renames(root, base string) map[string]string {
	args := []string{"diff", "-z", "--name-status", "-M", "--diff-filter=R", "--cached", base}
	if os.Getenv("GITHUB_BASE_REF") != "" {
		args = []string{"diff", "-z", "--name-status", "-M", "--diff-filter=R", base, "HEAD"}
	}
	out, _ := Repo(root).run(args...)
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	renames := map[string]string{}
	for i := 0; i+2 < len(fields); i += 3 {
		renames[fields[i+2]] = fields[i+1]
	}
	return renames
}
