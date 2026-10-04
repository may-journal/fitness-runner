package release

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// EnsureTag publishes the changelog version at HEAD without moving existing tags.
// The caller must select a successful main commit. Generated pin updates are skipped.
func (c Config) EnsureTag(ctx context.Context) error {
	skip, err := c.PinOnlyCommit(ctx)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}
	return c.publishHeadTag(ctx)
}

func (c Config) publishHeadTag(ctx context.Context) error {
	version, err := c.Version()
	if err != nil {
		return err
	}
	head, err := c.releaseGit(ctx, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return err
	}
	return c.ensureRemoteTag(ctx, "refs/tags/go/v"+version, head)
}

func (c Config) ensureRemoteTag(ctx context.Context, ref, head string) error {
	remote, err := c.remoteTag(ctx, ref)
	if err != nil {
		return err
	}
	if remote != "" {
		return requireTagCommit(ref, remote, head)
	}
	_, err = c.releaseGit(ctx, "push", "origin", head+":"+ref)
	if err != nil {
		return c.checkTagRace(ctx, ref, head, err)
	}
	return nil
}

// A concurrent successful publisher is safe; a conflicting tag is never forced.
func (c Config) checkTagRace(ctx context.Context, ref, head string, pushErr error) error {
	remote, err := c.remoteTag(ctx, ref)
	if err != nil {
		return pushErr
	}
	if remote == head {
		return nil
	}
	return pushErr
}

func requireTagCommit(ref, remote, head string) error {
	if remote != head {
		return fmt.Errorf("release tag %s already points to %s, not %s; add a new changelog version", ref, remote, head)
	}
	return nil
}

func (c Config) remoteTag(ctx context.Context, ref string) (string, error) {
	output, err := c.releaseGit(ctx, "ls-remote", "--tags", "origin", ref, ref+"^{}")
	if err != nil {
		return "", err
	}
	refs := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			refs[parts[1]] = parts[0]
		}
	}
	return tagCommit(refs, ref), nil
}

func tagCommit(refs map[string]string, ref string) string {
	if commit := refs[ref+"^{}"]; commit != "" {
		return commit
	}
	return refs[ref]
}

func (c Config) releaseGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.Root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

var pinTrailer = regexp.MustCompile(`(?m)^Fitness-Pin-Release: go/v(0\.[0-9]{8}\.[0-9]+)$`)

// PinOnlyCommit recognizes generated pin updates by their content and trailer.
// A title or trailer alone never exempts a functional change from release.
func (c Config) PinOnlyCommit(ctx context.Context) (bool, error) {
	return c.pinOnlyRevision(ctx, "HEAD")
}

func (c Config) pinOnlyRevision(ctx context.Context, revision string) (bool, error) {
	message, err := c.releaseGit(ctx, "show", "-s", "--format=%B", revision)
	if err != nil {
		return false, err
	}
	match := pinTrailer.FindStringSubmatch(message)
	if match == nil {
		return false, nil
	}
	parents, err := c.releaseGit(ctx, "rev-list", "--parents", "-n", "1", revision)
	if err != nil {
		return false, err
	}
	if len(strings.Fields(parents)) < 2 {
		return false, nil
	}
	return c.pinCommitPaths(ctx, match[1], revision)
}

func (c Config) pinCommitPaths(ctx context.Context, version, revision string) (bool, error) {
	changed, err := c.releaseGit(ctx, "diff-tree", "-r", "--no-commit-id", "--name-only", revision+"^", revision)
	if err != nil {
		return false, err
	}
	paths := strings.Split(changed, "\n")
	if !slices.Contains(paths, "CHANGELOG.md") {
		return false, nil
	}
	if len(paths) < 2 {
		return false, nil
	}
	return c.onlyPinPaths(ctx, paths, version, revision)
}

func (c Config) onlyPinPaths(ctx context.Context, paths []string, version, revision string) (bool, error) {
	for _, path := range paths {
		valid, err := c.pinPath(ctx, path, version, revision)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, nil
		}
	}
	return true, nil
}

func (c Config) pinPath(ctx context.Context, path, version, revision string) (bool, error) {
	if path != "CHANGELOG.md" && !slices.Contains(PinFiles(), path) {
		return false, nil
	}
	before, err := c.releaseGit(ctx, "show", revision+"^:"+path)
	if err != nil {
		return false, err
	}
	after, err := c.releaseGit(ctx, "show", revision+":"+path)
	if err != nil {
		return false, err
	}
	return pinContent(path, before, after, version), nil
}

func pinContent(path, before, after, version string) bool {
	if path == "CHANGELOG.md" {
		return pinChangelog(before, after, version)
	}
	return before != after && NormalizePins(path, before) == NormalizePins(path, after)
}

func pinChangelog(before, after, version string) bool {
	const prefix = "\n## Changes\n\n"
	index := strings.Index(after, prefix)
	if index < 0 {
		return false
	}
	start := index + len(prefix)
	end := strings.Index(after[start:], "\n### ")
	if end < 0 {
		return false
	}
	section := after[start : start+end]
	lines := strings.SplitN(section, "\n", 2)
	if !headingPattern.MatchString(lines[0]) {
		return false
	}
	expected := "\n\n- Chore: update Fitness pins to verified release go/v" + version + ".\n- Chore: keep the installer and shared workflows on the same release.\n- Docs: refresh the pinned release used in setup examples.\n"
	if section != lines[0]+expected {
		return false
	}
	return after[:start]+after[start+end+1:] == before
}
