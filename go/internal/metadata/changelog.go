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

// changelogErrors holds a PR to one changelog/ entry file, which its
// description links instead of copying. A PR that leaves the changelog alone
// is not judged here; only a Release Please PR may change CHANGELOG.md, by
// adding its version section.
func changelogErrors(t target, gh aftereffect.Client) []string {
	patch, err := changelogPatch(t, gh)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the %s diff of PR #%d: %v; check gh authentication and network access, then retry", changelog.File, t.Number, err)}
	}
	files, err := entryFiles(t, gh)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the %s/ files of PR #%d: %v; check gh authentication and network access, then retry", changelog.EntryDir, t.Number, err)}
	}
	return entryErrors(patch, files, t.Body)
}

// entryErrors judges a PR's CHANGELOG.md diff, its entry files, and its
// description. A release PR adds a version section and links it. Any other
// change to an existing CHANGELOG.md fails, since entries live in entry
// files; a PR may create the file. Editing or deleting an entry file already
// pending on the base amends that entry, so it needs no new one.
func entryErrors(patch string, files []aftereffect.FileChange, body string) []string {
	diff := changelog.ReadDiff(patch)
	if len(diff.Releases) > 0 {
		return sectionErrors(diff, body)
	}
	var errs []string
	if patch != "" && !changelog.CreatesFile(patch) {
		errs = append(errs, msgSectionEdit)
	}
	return append(errs, oneEntryErrors(addedEntries(files), body)...)
}

// oneEntryErrors passes at most one new entry file, which the description
// links and does not copy, and names the extras.
func oneEntryErrors(added []aftereffect.FileChange, body string) []string {
	switch {
	case len(added) > 1:
		return []string{extraEntriesError(added)}
	case len(added) == 1:
		return addedEntryErrors(added[0], body)
	}
	return nil
}

// msgSectionEdit fails a PR that adds or edits a CHANGELOG.md section.
const msgSectionEdit = "this PR changes " + changelog.File + "; only the release PR writes it, so record this PR's change in one " + changelog.EntryDir + "/<yyyy-mm-dd-HHMM>-<change>.md entry file instead"

// extraEntriesError names the entry file to keep and the ones to fold into it.
func extraEntriesError(added []aftereffect.FileChange) string {
	var names []string
	for _, f := range added {
		names = append(names, f.Filename)
	}
	return fmt.Sprintf("this PR adds %d changelog entries; keep %s and move the bullets of %s into it",
		len(names), names[0], strings.Join(names[1:], ", "))
}

// sectionErrors judges a release PR: the description links the version
// section it adds and copies no bullet.
func sectionErrors(diff changelog.Diff, body string) []string {
	return append(linkErrors(diff, body), copyErrors(changelog.File, diff.Bullets, body)...)
}

// addedEntries are the entry files among a PR's changed files that it adds.
func addedEntries(files []aftereffect.FileChange) []aftereffect.FileChange {
	var added []aftereffect.FileChange
	for _, f := range files {
		if f.Status == "added" {
			added = append(added, f)
		}
	}
	return added
}

// addedEntryErrors fails an entry file whose name lacks its heading's date,
// and a description that does not link it or copies one of its bullets.
func addedEntryErrors(f aftereffect.FileChange, body string) []string {
	var errs []string
	if msg := changelog.NameError(f.Filename, changelog.AddedText(f.Patch)); msg != "" {
		errs = append(errs, msg)
	}
	links := changelog.EntryLinks(body)
	if !slices.Contains(links, f.Filename) {
		fix := fmt.Sprintf("link the entry this PR adds in ## Changelog: [Changelog entry](…/%s)", f.Filename)
		if len(links) == 0 {
			errs = append(errs, "PR description has no "+changelog.EntryDir+"/ entry link; "+fix)
		} else {
			errs = append(errs, fmt.Sprintf("PR description links %s; %s", links[0], fix))
		}
	}
	return append(errs, copyErrors(f.Filename, changelog.ReadDiff(f.Patch).Bullets, body)...)
}

// entryFiles is the local branch's entry files when branch-check supplied
// them, else the PR's as GitHub holds them, keeping only entry files.
func entryFiles(t target, gh aftereffect.Client) ([]aftereffect.FileChange, error) {
	files := t.EntryFiles
	if files == nil {
		read, err := gh.DirFiles(t.Number, changelog.EntryDir)
		if err != nil {
			return nil, err
		}
		files = &read
	}
	return slices.DeleteFunc(slices.Clone(*files), func(f aftereffect.FileChange) bool {
		return !changelog.IsEntryPath(f.Filename)
	}), nil
}

// linkErrors fails a description that does not link the release section the
// PR adds to CHANGELOG.md, naming that section's anchor.
func linkErrors(diff changelog.Diff, body string) []string {
	want, links := diff.Anchors(), changelog.Links(body)
	if len(want) == 0 {
		return nil
	}
	for _, link := range links {
		if slices.Contains(want, link) {
			return nil
		}
	}
	fix := fmt.Sprintf("link the entry this PR adds in ## Changelog: [CHANGELOG entry](…/%s#%s)", changelog.File, want[0])
	if len(links) == 0 {
		return []string{"PR description has no " + changelog.File + " link; " + fix}
	}
	return []string{fmt.Sprintf("PR description links %s#%s; %s", changelog.File, links[0], fix)}
}

// copyErrors fails a description that repeats a bullet the PR adds to file;
// the description links the entry instead.
func copyErrors(file string, bullets []string, body string) []string {
	text := strings.Join(strings.Fields(body), " ")
	var errs []string
	for _, bullet := range bullets {
		if line := strings.Join(strings.Fields(bullet), " "); strings.Contains(text, line) {
			errs = append(errs, fmt.Sprintf("PR description copies the %s line %q; link the entry instead", file, line))
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

// branchEntryFiles are the entry files the local tip changes against the
// PR's base commit, with the status and diff GitHub will show once pushed.
func branchEntryFiles(ctx context.Context, root, tip string, pr *branchPR) ([]aftereffect.FileChange, error) {
	span := pr.BaseSHA + "..." + tip
	out, err := exec.CommandContext(ctx, "git", "-C", root, "diff", "--name-status", "--no-renames", span, "--", changelog.EntryDir).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot list %s/ changes from base commit %s: %w", changelog.EntryDir, pr.BaseSHA, err)
	}
	var files []aftereffect.FileChange
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f, err := entryChange(ctx, root, span, line)
		if err != nil {
			return nil, err
		}
		if f != nil {
			files = append(files, *f)
		}
	}
	return files, nil
}

// entryChange is the entry file one --name-status line names, with its
// diff over span, or nil when the line names no entry file.
func entryChange(ctx context.Context, root, span, line string) (*aftereffect.FileChange, error) {
	code, name, ok := strings.Cut(line, "\t")
	if !ok || !changelog.IsEntryPath(name) {
		return nil, nil
	}
	patch, err := exec.CommandContext(ctx, "git", "-C", root, "diff", span, "--", name).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot diff %s over %s: %w", name, span, err)
	}
	return &aftereffect.FileChange{Filename: name, Status: gitStatus(code), Patch: string(patch)}, nil
}

// gitStatus is the GitHub file status for git's --name-status code.
func gitStatus(code string) string {
	if status, ok := map[string]string{"A": "added", "D": "removed"}[code]; ok {
		return status
	}
	return "modified"
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
