package gitx

import (
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// replayHeads are the refs git keeps while a commit replays another one.
var replayHeads = []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"}

// Replaying reports whether a local commit replays changes judged where they
// were first made: a merge, cherry-pick, or revert in progress, or an amend
// of a merge commit, whose second parent brought them. A pull request run,
// with GITHUB_BASE_REF set, never replays, so its whole diff is judged.
func Replaying(root string) bool {
	if os.Getenv("GITHUB_BASE_REF") != "" {
		return false
	}
	refs := replayHeads
	if Amending() {
		refs = append(slices.Clone(refs), "HEAD^2")
	}
	return slices.ContainsFunc(refs, func(ref string) bool { return exists(root, ref) })
}

// Amending reports whether the runner marked this commit as an amend of HEAD.
func Amending() bool {
	return os.Getenv("FITNESS_COMMIT_AMEND") == "1"
}

// CommitChanges is what the commit or pull request changes: its diff, nil
// for a full scan or a `--dir` folder, or for an amend everything staged
// since the amended commit's parent.
func CommitChanges(root string) []string {
	if !Amending() {
		return checkkit.DiffFiles()
	}
	return StagedSince(root, AmendParent(root))
}

// AmendParent is the revision an amended commit's changes start from: its
// parent, or the empty tree when it is the first commit.
func AmendParent(root string) string {
	if exists(root, "HEAD^") {
		return "HEAD^"
	}
	return emptyTree(root)
}

// emptyTree returns the ID of the empty tree in root's object format.
func emptyTree(root string) string {
	cmd := exec.Command("git", "-C", root, "hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = strings.NewReader("")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}
