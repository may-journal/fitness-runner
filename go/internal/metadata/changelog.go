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
// description links instead of copying. A PR that adds none, such as a
// release PR, is not judged here, and CHANGELOG.md is no part of it.
func changelogErrors(t target, gh aftereffect.Client) []string {
	files, err := entryFiles(t, gh)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the %s/ files of PR #%d: %v; check gh authentication and network access, then retry", changelog.EntryDir, t.Number, err)}
	}
	return oneEntryErrors(addedEntries(files), t.Body)
}

// oneEntryErrors passes at most one new entry file, which the description
// links and does not copy, and names the extras. Editing or deleting an
// entry file already pending on the base amends that entry, so it needs no
// new one.
func oneEntryErrors(added []aftereffect.FileChange, body string) []string {
	switch {
	case len(added) > 1:
		return []string{extraEntriesError(added)}
	case len(added) == 1:
		return addedEntryErrors(added[0], body)
	}
	return nil
}

// extraEntriesError names the entry file to keep and the ones to fold into it.
func extraEntriesError(added []aftereffect.FileChange) string {
	var names []string
	for _, f := range added {
		names = append(names, f.Filename)
	}
	return fmt.Sprintf("this PR adds %d changelog entries; keep %s and move the bullets of %s into it",
		len(names), names[0], strings.Join(names[1:], ", "))
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
