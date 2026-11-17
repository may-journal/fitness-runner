package checkkit

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// The virtual paths of the documents that are not repo files, which a
// check's Files pattern names as it names files. A check reads one through
// `--body-file`, or a commit message through FITNESS_CTX_MESSAGE.
const (
	// CommitMessagePath is the commit message a commit-msg hook judges.
	CommitMessagePath = ".git/COMMIT_EDITMSG"
	// HistoryPath is the branch's history, which every run of the repo's
	// files has and a piped text lacks.
	HistoryPath = ".git/logs/HEAD"
	// PullBodyPath is a pull request description.
	PullBodyPath = ".fitness/github/pull/body.md"
	// CommentBodyPath is an issue or pull request comment.
	CommentBodyPath = ".fitness/github/comment/body.md"
	// BlobPath is a text piped to `fitness --blob`, unless `--as` names
	// another path.
	BlobPath = ".fitness/blob.md"
)

// BodyPattern finds the virtual path of every GitHub body and piped text.
const BodyPattern = `^\.fitness/(github/|blob\.md$)`

// GitPattern finds the commit message and branch history paths.
const GitPattern = `^\.git/(COMMIT_EDITMSG|logs/HEAD)$`

// IssueBodyPath is the virtual path of an issue body of kind label kind,
// such as Plan. An unlabeled issue's kind is `issue`.
func IssueBodyPath(kind string) string {
	if kind == "" {
		kind = "issue"
	}
	return ".fitness/github/issue/" + strings.ReplaceAll(strings.ToLower(kind), " ", "-") + ".md"
}

// Effective maps d's deprecated Scope and Blob onto Files, Exclude and
// Requires, as docs/check-protocol.md lists, keeping d's own Files.
func Effective(d Describe) Describe {
	if d.Files == "" {
		d.Files = scopePattern(d)
	}
	// A check never meets a virtual document it does not read.
	d.Exclude = excluding(d.Exclude, BodyPattern, JudgesText(d))
	d.Exclude = excluding(d.Exclude, GitPattern, finds(d.Files, ".git/", CommitMessagePath, HistoryPath))
	d.Requires = scopeRequires(d)
	return d
}

// excluding is exclude, plus pattern unless the check reads what it finds.
func excluding(exclude, pattern string, reads bool) string {
	if reads || strings.Contains(exclude, pattern) {
		return exclude
	}
	return either(exclude, pattern)
}

// scopeRequires is d's Requires plus history for scope git, or github for
// scope github.
func scopeRequires(d Describe) []Requirement {
	switch d.Scope {
	case ScopeGit:
		return with(d.Requires, RequiresHistory)
	case ScopeGitHub:
		return with(d.Requires, RequiresGitHub)
	}
	return d.Requires
}

// JudgesText reports whether d reads a text through `--body-file`. It sets
// Blob or scope blob, or its own Files finds the body paths by their folder.
func JudgesText(d Describe) bool {
	return d.Blob || d.Scope == ScopeBlob || finds(d.Files, ".fitness/", PullBodyPath, CommentBodyPath, IssueBodyPath(""), BlobPath)
}

// finds reports whether pattern names virtual folder. Its literal prefix
// starts there, or it finds one of paths and misses the file name alone.
func finds(pattern, folder string, paths ...string) bool {
	re, err := regexp.Compile(pattern)
	if pattern == "" || err != nil {
		return false
	}
	prefix, _ := re.LiteralPrefix()
	return strings.HasPrefix(prefix, folder) || slices.ContainsFunc(paths, func(p string) bool {
		return re.MatchString(p) && !re.MatchString(path.Base(p))
	})
}

// scopePattern is the Files pattern d's Scope stands for, or "" for every path.
func scopePattern(d Describe) string {
	switch {
	case d.Scope == ScopeBlob:
		return BodyPattern
	case d.Scope == ScopeGit && d.Blob:
		return either(GitPattern, BodyPattern)
	case d.Scope == ScopeGit:
		return GitPattern
	}
	return ""
}

// either is the pattern finding what a or b finds; an empty a adds nothing.
func either(a, b string) string {
	if a == "" {
		return b
	}
	return "(?:" + a + ")|(?:" + b + ")"
}

// with is requires plus r, once.
func with(requires []Requirement, r Requirement) []Requirement {
	if slices.Contains(requires, r) {
		return requires
	}
	return append(slices.Clone(requires), r)
}
