// Package gitx wraps the git invocations the runner and checks share.
package gitx

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// StagedFiles returns `git diff --cached --name-only` for root; nil on any git failure (not a repo, no git).
func StagedFiles(root string) []string {
	out, err := exec.Command("git", "-C", root, "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil
	}
	return nameList(string(out))
}

// StagedSince returns what the index changes since rev (`git diff --cached
// --name-only rev`); nil on any git failure.
func StagedSince(root, rev string) []string {
	out, err := exec.Command("git", "-C", root, "diff", "--cached", "--name-only", rev).Output()
	if err != nil {
		return nil
	}
	return nameList(string(out))
}

// ChangedSince returns the paths HEAD changes since its merge base with base
// (`git diff --name-only base...HEAD`); nil
// on any git failure, such as a base ref the clone never fetched.
func ChangedSince(root, base string) []string {
	out, err := exec.Command("git", "-C", root, "diff", "--name-only", base+"...HEAD").Output()
	if err != nil {
		return nil
	}
	return nameList(string(out))
}

// nameList splits `git diff --name-only` output into paths, dropping blank
// lines.
func nameList(out string) []string {
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			files = append(files, line)
		}
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

// DefaultBase returns origin's default branch as a ref, such as origin/main,
// falling back to origin/main when it is unknown.
func DefaultBase(root string) string {
	if base := knownDefaultBase(root); base != "" {
		return base
	}
	return "origin/main"
}

// knownDefaultBase is origin/HEAD's target, else a fetched origin/main or
// origin/master; empty when none is known.
func knownDefaultBase(root string) string {
	out, err := exec.Command("git", "-C", root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if ref := strings.TrimSpace(string(out)); err == nil && ref != "" {
		return ref
	}
	for _, ref := range []string{"origin/main", "origin/master"} {
		if exists(root, ref) {
			return ref
		}
	}
	return ""
}

// OwnCommits returns the commits reachable from tip that no origin branch
// other than origin/<branch> holds, newest first, so a branch stacked on a
// pushed branch, such as an epic branch, starts where that branch ends.
// Branch is the branch's name on origin. Without a branch or a pushed
// default branch it falls back to BranchCommits.
func OwnCommits(root, tip, branch string) []string {
	branch = strings.TrimPrefix(branch, "refs/heads/")
	if branch == "" || !exists(root, DefaultBase(root)) {
		return BranchCommits(root, tip, DefaultBase(root))
	}
	return notOnOrigin(root, tip, branch)
}

// notOnOrigin returns the commits reachable from tip that no origin branch
// holds, newest first, leaving origin/<keep> out of that set when keep is
// set; nil on any git failure.
func notOnOrigin(root, tip, keep string) []string {
	args := []string{"-C", root, "rev-list", tip, "--not"}
	if keep != "" {
		args = append(args, "--exclude=origin/"+keep)
	}
	out, err := exec.Command("git", append(args, "--remotes=origin")...).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// BranchOwnCommits returns the commits the checked-out work adds over its
// base, newest first, and whether a base was found. A pull request run in
// GitHub Actions starts from the fetched GITHUB_BASE_REF branch, as the
// local-branch check does. Locally, a branch counts the commits no other
// origin branch holds, whatever the default branch is called. The default
// branch, a detached HEAD, or any branch while the default is unknown
// counts only the commits origin lacks. Without origin refs, ok is false.
func BranchOwnCommits(root string) (commits []string, ok bool) {
	if base := FetchedBase(context.Background(), root, os.Getenv("GITHUB_BASE_REF")); base != "" {
		return BranchCommits(root, "HEAD", base), true
	}
	out, _ := exec.Command("git", "-C", root, "for-each-ref", "--count=1", "refs/remotes/origin/").Output()
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil, false
	}
	return notOnOrigin(root, "HEAD", featureBranch(root)), true
}

// featureBranch is the checked-out branch, or empty on the default branch, on
// a detached HEAD, or while the default branch is unknown.
func featureBranch(root string) string {
	branch, base := CurrentBranch(root), knownDefaultBase(root)
	if base == "" || "origin/"+branch == base {
		return ""
	}
	return branch
}

// FetchedBase is the commit origin's branch points to locally, or empty.
func FetchedBase(ctx context.Context, root, branch string) string {
	if branch == "" {
		return ""
	}
	out, _ := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch+"^{commit}").Output()
	return strings.TrimSpace(string(out))
}

// CurrentBranch returns the checked-out branch's short name; empty when
// HEAD is detached or git fails.
func CurrentBranch(root string) string {
	out, _ := exec.Command("git", "-C", root, "symbolic-ref", "--short", "-q", "HEAD").Output()
	return strings.TrimSpace(string(out))
}

func exists(root, ref string) bool {
	return exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", ref).Run() == nil
}

// BranchCommits returns the commits reachable from tip but not from base,
// newest first; nil on any git failure, such as a base the clone lacks.
func BranchCommits(root, tip, base string) []string {
	out, err := exec.Command("git", "-C", root, "rev-list", tip, "--not", base).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// Message returns one commit's full message; empty on failure.
func Message(root, rev string) string {
	out, err := exec.Command("git", "-C", root, "log", "-1", "--format=%B", rev).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// originSlugRe captures owner/repo from a GitHub remote URL in https or ssh
// form, with or without the .git suffix.
var originSlugRe = regexp.MustCompile(`github\.com[:/]([\w.-]+/[\w.-]+?)(?:\.git)?/?$`)

// OriginSlug returns origin's GitHub owner/repo in lower case; empty when
// origin is missing or not on GitHub.
func OriginSlug(root string) string {
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	m := originSlugRe.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

// Format returns one commit's `git log --format` output for format; empty on
// failure.
func Format(root, rev, format string) string {
	out, err := exec.Command("git", "-C", root, "log", "-1", "--format="+format, rev).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
