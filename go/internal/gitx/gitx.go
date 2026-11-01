// Package gitx wraps the git invocations the runner and checks share.
//
// Repo's methods return an error when git fails. Git describes the clone, not
// the code, so every such error is a checkkit.Internal error that a check
// reports as info. The package-level functions keep their older shape and
// read a failure as empty. Every call runs under Deadline.
package gitx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Repo is the root of a git work tree that git commands run in.
type Repo string

// git runs `git -C repo args` with stdin, when set, and returns its stdout,
// or an internal error naming git's stderr. A call ctx ends early is
// stopped; ending it at its deadline is a StalledError.
func (r Repo) git(ctx context.Context, stdin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", string(r)}, args...)...)
	// A tool git started may hold its output open after git is stopped.
	cmd.WaitDelay = time.Second
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return gitResult(ctx, checkkit.Exec(cmd), args)
}

// gitResult reads a finished git call: its stdout, or why it failed.
func gitResult(ctx context.Context, o checkkit.Outcome, args []string) (string, error) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "", checkkit.Internal(&StalledError{Args: args})
	case o.SpawnErr != nil:
		return "", checkkit.ToolMissing("git")
	case o.Err != nil:
		return "", checkkit.Internal(fmt.Errorf("git %s: %s", args[0], gitReason(o)))
	}
	return o.Stdout, nil
}

// gitReason names the remote when git refused to prompt, else git's last
// stderr line, else the run error.
func gitReason(o checkkit.Outcome) string {
	if remote, ok := RefusedPrompt(o.Stderr); ok {
		return PromptNote(remote)
	}
	lines := strings.Split(strings.TrimSpace(o.Stderr), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
		return last
	}
	return o.Err.Error()
}

// run runs one git call under Deadline.
func (r Repo) run(args ...string) (string, error) {
	return r.input("", args...)
}

// input runs one git call under Deadline, with stdin as its input.
func (r Repo) input(stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Deadline)
	defer cancel()
	return r.git(ctx, stdin, args...)
}

// names runs a git command that prints one path or commit per line.
func (r Repo) names(args ...string) ([]string, error) {
	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}
	return nameList(out), nil
}

// StagedFiles returns `git diff --cached --name-only`.
func (r Repo) StagedFiles() ([]string, error) {
	return r.names("diff", "--cached", "--name-only")
}

// StagedSince returns what the index changes since rev.
func (r Repo) StagedSince(rev string) ([]string, error) {
	return r.names("diff", "--cached", "--name-only", rev)
}

// ChangedSince returns the paths HEAD changes since its merge base with base.
func (r Repo) ChangedSince(base string) ([]string, error) {
	return r.names("diff", "--name-only", base+"...HEAD")
}

// FilesUnder returns the tracked and new, not ignored, files under dir
// (`git ls-files -co --exclude-standard -- dir`).
func (r Repo) FilesUnder(dir string) ([]string, error) {
	return r.names("ls-files", "-co", "--exclude-standard", "--", dir)
}

// BranchCommits returns the commits reachable from tip but not from base,
// newest first.
func (r Repo) BranchCommits(tip, base string) ([]string, error) {
	return r.names("rev-list", tip, "--not", base)
}

// Format returns one commit's `git log --format` output for format.
func (r Repo) Format(rev, format string) (string, error) {
	return r.run("log", "-1", "--format="+format, rev)
}

// Message returns one commit's full message.
func (r Repo) Message(rev string) (string, error) {
	return r.Format(rev, "%B")
}

// CurrentBranch returns the checked-out branch's short name; empty without
// an error when HEAD is detached.
func (r Repo) CurrentBranch() (string, error) {
	out, err := r.run("symbolic-ref", "--short", "-q", "HEAD")
	if err != nil && r.detached() {
		return "", nil
	}
	return strings.TrimSpace(out), err
}

// detached reports whether HEAD names a commit but no branch.
func (r Repo) detached() bool {
	_, err := r.run("rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

// OriginSlug returns origin's lower-case GitHub owner/repo, empty when origin
// is not on GitHub, and an error without origin.
func (r Repo) OriginSlug() (string, error) {
	out, err := r.run("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	m := originSlugRe.FindStringSubmatch(strings.TrimSpace(out))
	if m == nil {
		return "", nil
	}
	return strings.ToLower(m[1]), nil
}

// Health is what a probe found wrong with a clone for checks that read its
// history or its GitHub origin.
type Health struct {
	NoGit, NotRepo, Shallow, NoOrigin, Detached bool
}

// Probe checks in order that git runs, root is a work tree, history is full,
// origin exists, and HEAD is on a branch.
func (r Repo) Probe() Health {
	if _, err := exec.LookPath("git"); err != nil {
		return Health{NoGit: true}
	}
	if _, err := r.run("rev-parse", "--is-inside-work-tree"); err != nil {
		return Health{NotRepo: true}
	}
	shallow, _ := r.run("rev-parse", "--is-shallow-repository")
	_, originErr := r.run("remote", "get-url", "origin")
	_, branchErr := r.run("symbolic-ref", "-q", "HEAD")
	return Health{
		Shallow:  strings.TrimSpace(shallow) == "true",
		NoOrigin: originErr != nil,
		Detached: branchErr != nil,
	}
}

// Problems names each failure the probe found, empty for a healthy clone.
func (h Health) Problems() []string {
	var out []string
	for _, p := range []struct {
		bad bool
		msg string
	}{
		{h.NoGit, "git is not installed or not on PATH"},
		{h.NotRepo, "not a git repository"},
		{h.Shallow, "shallow clone: history is incomplete"},
		{h.NoOrigin, "no origin remote"},
		{h.Detached, "detached HEAD: no branch is checked out"},
	} {
		if p.bad {
			out = append(out, p.msg)
		}
	}
	return out
}

// Err is an internal error naming every problem, nil for a healthy clone.
func (h Health) Err() error {
	if problems := h.Problems(); len(problems) > 0 {
		return checkkit.Internal(errors.New(strings.Join(problems, "; ")))
	}
	return nil
}

// StagedFiles returns `git diff --cached --name-only` for root; nil on any git failure (not a repo, no git).
func StagedFiles(root string) []string {
	files, _ := Repo(root).StagedFiles()
	return files
}

// StagedSince returns what the index changes since rev (`git diff --cached
// --name-only rev`); nil on any git failure.
func StagedSince(root, rev string) []string {
	files, _ := Repo(root).StagedSince(rev)
	return files
}

// ChangedSince returns the paths HEAD changes since its merge base with base,
// or nil on any git failure.
func ChangedSince(root, base string) []string {
	files, _ := Repo(root).ChangedSince(base)
	return files
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
	return Message(root, "HEAD")
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
	out, err := Repo(root).run("symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if ref := strings.TrimSpace(out); err == nil && ref != "" {
		return ref
	}
	for _, ref := range []string{"origin/main", "origin/master"} {
		if exists(root, ref) {
			return ref
		}
	}
	return ""
}

// OwnCommits returns the newest-first commits from tip that no origin branch
// other than origin/<branch> holds, falling back to BranchCommits without one.
func OwnCommits(root, tip, branch string) []string {
	branch = strings.TrimPrefix(branch, "refs/heads/")
	if branch == "" || !exists(root, DefaultBase(root)) {
		return BranchCommits(root, tip, DefaultBase(root))
	}
	return notOnOrigin(root, tip, branch)
}

// notOnOrigin returns the newest-first commits from tip that no origin branch
// holds, excluding origin/<keep>, or nil on any git failure.
func notOnOrigin(root, tip, keep string) []string {
	args := []string{"rev-list", tip, "--not"}
	if keep != "" {
		args = append(args, "--exclude=origin/"+keep)
	}
	commits, _ := Repo(root).names(append(args, "--remotes=origin")...)
	return commits
}

// BranchOwnCommits returns the newest-first commits the checked-out work adds
// over its base, with ok false when there are no origin refs.
func BranchOwnCommits(root string) (commits []string, ok bool) {
	if base := FetchedBase(context.Background(), root, os.Getenv("GITHUB_BASE_REF")); base != "" {
		return BranchCommits(root, "HEAD", base), true
	}
	out, _ := Repo(root).run("for-each-ref", "--count=1", "refs/remotes/origin/")
	if len(strings.TrimSpace(out)) == 0 {
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
	out, _ := Repo(root).git(ctx, "", "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch+"^{commit}")
	return strings.TrimSpace(out)
}

// CurrentBranch returns the checked-out branch's short name; empty when
// HEAD is detached or git fails.
func CurrentBranch(root string) string {
	branch, _ := Repo(root).CurrentBranch()
	return branch
}

func exists(root, ref string) bool {
	_, err := Repo(root).run("rev-parse", "-q", "--verify", ref)
	return err == nil
}

// BranchCommits returns the newest-first commits from tip but not base, or nil
// on any git failure.
func BranchCommits(root, tip, base string) []string {
	commits, _ := Repo(root).BranchCommits(tip, base)
	return commits
}

// Message returns one commit's full message; empty on failure.
func Message(root, rev string) string {
	return Format(root, rev, "%B")
}

// originSlugRe captures owner/repo from a GitHub remote URL in https or ssh
// form, with or without the .git suffix.
var originSlugRe = regexp.MustCompile(`github\.com[:/]([\w.-]+/[\w.-]+?)(?:\.git)?/?$`)

// OriginSlug returns origin's GitHub owner/repo in lower case; empty when
// origin is missing or not on GitHub.
func OriginSlug(root string) string {
	slug, _ := Repo(root).OriginSlug()
	return slug
}

// Format returns one commit's `git log --format` output for format; empty on
// failure.
func Format(root, rev, format string) string {
	out, _ := Repo(root).Format(rev, format)
	return out
}
