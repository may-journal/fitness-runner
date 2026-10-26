package check

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// trailer matches the one exception: a `Requirements: none, <reason>` line
// with a reason after the comma.
var trailer = regexp.MustCompile(`(?m)^Requirements: none, *\S.*$`)

// exemption reports which files a trailer exempts. A commit message the
// runner provides exempts every file when it carries the trailer. A pull
// request, with base set, exempts the files that its commits carrying the
// trailer change. A run with neither exempts nothing.
func exemption(root, base string) func(string) bool {
	if msg, ok := checkkit.CtxMessage(); ok {
		all := trailer.MatchString(msg)
		return func(string) bool { return all }
	}
	files := map[string]bool{}
	if base != "" {
		files = trailerFiles(root, base)
	}
	return func(f string) bool { return files[f] }
}

// trailerFiles lists the files changed by each commit since base whose
// message carries the trailer.
func trailerFiles(root, base string) map[string]bool {
	files := map[string]bool{}
	for _, c := range gitx.BranchCommits(root, "HEAD", base) {
		if !trailer.MatchString(gitx.Message(root, c)) {
			continue
		}
		out, _ := gitOut(root, "diff-tree", "--no-commit-id", "--name-only", "-r", "-M", "--root", c)
		for _, f := range strings.Split(out, "\n") {
			files[f] = true
		}
	}
	return files
}
