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

// changelogErrors holds a PR to one dated CHANGELOG.md entry, which its
// description links instead of copying. A PR that leaves the changelog alone
// is not judged here; a Release Please PR may add its version section.
func changelogErrors(t target, gh aftereffect.Client) []string {
	patch, err := changelogPatch(t, gh)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the %s diff of PR #%d: %v; check gh authentication and network access, then retry", changelog.File, t.Number, err)}
	}
	return entryErrors(patch, t.Body)
}

// entryErrors judges a PR's CHANGELOG.md diff and its description: one new
// dated entry or a release, a link to that heading, and no copied bullet.
func entryErrors(patch, body string) []string {
	if patch == "" {
		return nil
	}
	diff := changelog.ReadDiff(patch)
	errs := countErrors(diff)
	errs = append(errs, linkErrors(diff, body)...)
	return append(errs, copyErrors(diff, body)...)
}

// countErrors passes a release or exactly one new dated entry, and names the
// extras otherwise.
func countErrors(diff changelog.Diff) []string {
	added := slices.Clone(diff.Entries)
	if len(diff.Releases) > 0 || len(added) == 1 {
		return nil
	}
	if len(added) == 0 {
		return []string{"CHANGELOG.md: this PR adds no dated ### yyyy.mm.dd.HHMM entry; add one entry that describes the whole PR"}
	}
	slices.Sort(added)
	return []string{fmt.Sprintf("CHANGELOG.md: this PR adds %d dated entries; keep ### %s and move the bullets of ### %s into it",
		len(added), added[0], strings.Join(added[1:], ", ### "))}
}

// linkErrors fails a description whose CHANGELOG.md links all miss the
// heading the PR adds. A description with no such link is pr-structure's.
func linkErrors(diff changelog.Diff, body string) []string {
	want, links := diff.Anchors(), changelog.Links(body)
	if len(want) == 0 || len(links) == 0 {
		return nil
	}
	for _, link := range links {
		if slices.Contains(want, link) {
			return nil
		}
	}
	return []string{fmt.Sprintf("PR description links %s#%s; link the entry this PR adds, %s#%s", changelog.File, links[0], changelog.File, want[0])}
}

// copyErrors fails a description that repeats a bullet the PR adds to
// CHANGELOG.md; the description links the entry instead.
func copyErrors(diff changelog.Diff, body string) []string {
	text := strings.Join(strings.Fields(body), " ")
	var errs []string
	for _, bullet := range diff.Bullets {
		if line := strings.Join(strings.Fields(bullet), " "); strings.Contains(text, line) {
			errs = append(errs, fmt.Sprintf("PR description copies the %s line %q; link the entry instead", changelog.File, line))
		}
	}
	return errs
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
