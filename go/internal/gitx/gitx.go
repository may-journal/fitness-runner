// Package gitx wraps the git invocations the runner and checks share.
package gitx

import (
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
// falling back to origin/main when origin/HEAD is unset.
func DefaultBase(root string) string {
	out, err := exec.Command("git", "-C", root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if ref := strings.TrimSpace(string(out)); err == nil && ref != "" {
		return ref
	}
	return "origin/main"
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
	out, err := exec.Command("git", "-C", root, "rev-list", tip, "--not", "--exclude=origin/"+branch, "--remotes=origin").Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
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
