package metadata

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/changelog"
)

// changelogErrors holds a PR to one dated CHANGELOG.md entry. A PR that
// leaves the changelog alone, or a Release Please PR, is not judged here.
func changelogErrors(t target, gh aftereffect.Client) []string {
	patch, err := changelogPatch(t, gh)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the %s diff of PR #%d: %v; check gh authentication and network access, then retry", changelog.File, t.Number, err)}
	}
	return entryErrors(patch)
}

// entryErrors judges a PR's CHANGELOG.md diff: it passes no change, a
// release, or exactly one new dated entry, and names the extras otherwise.
func entryErrors(patch string) []string {
	added, release := changelog.DiffEntries(patch)
	if patch == "" || release || len(added) == 1 {
		return nil
	}
	if len(added) == 0 {
		return []string{"CHANGELOG.md: this PR adds no dated ### yyyy.mm.dd.HHMM entry; add one entry that describes the whole PR"}
	}
	slices.Sort(added)
	return []string{fmt.Sprintf("CHANGELOG.md: this PR adds %d dated entries; keep ### %s and move the bullets of ### %s into it",
		len(added), added[0], strings.Join(added[1:], ", ### "))}
}

// changelogPatch is the local branch's diff when branch-check supplied one,
// else the PR's diff as GitHub holds it.
func changelogPatch(t target, gh aftereffect.Client) (string, error) {
	if t.ChangelogPatch != nil {
		return *t.ChangelogPatch, nil
	}
	return gh.FilePatch(t.Number, changelog.File)
}

// branchChangelogPatch is the CHANGELOG.md diff from the PR's base commit to
// the local tip, as GitHub will show it once the tip is pushed.
func branchChangelogPatch(ctx context.Context, root, tip string, pr *branchPR) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", root, "diff", pr.BaseSHA+"..."+tip, "--", changelog.File).Output()
	if err != nil {
		return "", fmt.Errorf("cannot diff %s from base commit %s: %w", changelog.File, pr.BaseSHA, err)
	}
	return string(out), nil
}
