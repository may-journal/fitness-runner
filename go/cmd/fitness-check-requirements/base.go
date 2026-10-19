package check

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
)

// baseRev is the commit a scoped run's changes replace: the merge base with
// a pull request's base branch, the parent of an amended commit, or HEAD for
// staged files. It is empty when there is none, as in a first commit.
func baseRev(root string) string {
	if base := os.Getenv("GITHUB_BASE_REF"); base != "" {
		return gitLine(root, "merge-base", "HEAD", "origin/"+base)
	}
	if os.Getenv("FITNESS_COMMIT_AMEND") == "1" {
		return gitLine(root, "rev-parse", "-q", "--verify", "HEAD^")
	}
	return gitLine(root, "rev-parse", "-q", "--verify", "HEAD")
}

// gitLine is git's one line of output in root, or empty when git fails.
func gitLine(root string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// failedAtBase lists the files the requirements check failed at the base
// commit rev. This same check judges a checkout of rev, outside any hook or
// scope, and a file fails when an error names it. The list is empty when rev
// is empty or the checkout cannot be judged.
func failedAtBase(root, rev string) map[string]bool {
	failed := map[string]bool{}
	if rev == "" {
		return failed
	}
	dir, err := os.MkdirTemp("", "fitness-requirements-base-")
	if err != nil {
		return failed
	}
	defer os.RemoveAll(dir)
	for _, e := range baseErrors(root, rev, dir) {
		if file, _, ok := strings.Cut(e, ":"); ok {
			failed[file] = true
		}
	}
	return failed
}

// baseErrors checks out rev into dir and returns what this check reports
// there, as a full scan with no pull request or commit scope.
func baseErrors(root, rev, dir string) []string {
	self, err := os.Executable()
	if err != nil || checkout(root, rev, dir) != nil {
		return nil
	}
	cmd := exec.Command(self, "--root", dir)
	cmd.Env = append(toolchain.CleanEnv(os.Environ()), "GITHUB_BASE_REF=")
	out, _ := cmd.Output()
	var res checkkit.Result
	if json.Unmarshal(out, &res) != nil {
		return nil
	}
	return res.Errors
}

// checkout writes rev's tree into dir as a repository of its own that
// borrows root's objects, so root's index and work tree stay untouched.
func checkout(root, rev, dir string) error {
	if err := gitIn(dir, "init", "-q"); err != nil {
		return err
	}
	objects := gitLine(root, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	if err := os.WriteFile(filepath.Join(dir, ".git", "objects", "info", "alternates"), []byte(objects+"\n"), 0o600); err != nil {
		return err
	}
	if err := gitIn(dir, "read-tree", rev); err != nil {
		return err
	}
	return gitIn(dir, "checkout-index", "-a")
}

// gitIn runs git in dir without the hook variables that point at the
// repository being committed.
func gitIn(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = toolchain.CleanEnv(os.Environ())
	return cmd.Run()
}
