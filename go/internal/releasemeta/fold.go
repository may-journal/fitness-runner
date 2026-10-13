// cspell:ignore releasemeta
package releasemeta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
)

// foldMessage is the subject of the commit that folds entry files.
const foldMessage = "chore(release): fold changelog entries"

// commitMutation commits file changes to a branch through GitHub, which
// signs the commit.
const commitMutation = `mutation($input: CreateCommitOnBranchInput!) { createCommitOnBranch(input: $input) { commit { oid } } }`

// foldEntries moves the release branch's entry files into its CHANGELOG.md,
// below the version section Release Please wrote, and deletes them in one
// commit. It returns the branch's new head, or the old one when no entry
// files are pending.
func (c Client) foldEntries(ctx context.Context, pr pullRequest) (string, error) {
	if !c.pendingEntries() {
		return pr.Head.SHA, nil
	}
	entries, err := c.branchEntries(ctx, pr.Head.SHA)
	if err != nil || len(entries) == 0 {
		return pr.Head.SHA, err
	}
	text, err := c.raw(ctx, changelog.File, pr.Head.SHA)
	if err != nil {
		return "", err
	}
	return c.commitFold(ctx, pr, changelog.Fold(text, entries), entries)
}

// pendingEntries reports whether the checkout holds entry files, so a repo
// without them makes no extra GitHub reads.
func (c Client) pendingEntries() bool {
	pending, err := changelog.ReadEntries(c.Root)
	return err == nil && len(pending) > 0
}

// branchEntries reads the entry files on the release branch at sha, newest
// first.
func (c Client) branchEntries(ctx context.Context, sha string) ([]changelog.Entry, error) {
	paths, err := c.lines(ctx, "--jq", `.[] | select(.type == "file") | .path`, c.contents(changelog.EntryDir, sha))
	if err != nil {
		return nil, err
	}
	var entries []changelog.Entry
	for _, p := range paths {
		if !changelog.IsEntryPath(p) {
			continue
		}
		text, err := c.raw(ctx, p, sha)
		if err != nil {
			return nil, err
		}
		entries = append(entries, changelog.Entry{Path: p, Text: text})
	}
	changelog.SortEntries(entries)
	return entries, nil
}

// commitFold commits the folded CHANGELOG.md and the entry deletions on the
// release branch, expecting its current head, and returns the new head.
func (c Client) commitFold(ctx context.Context, pr pullRequest, text string, entries []changelog.Entry) (string, error) {
	type path struct {
		Path     string `json:"path"`
		Contents string `json:"contents,omitempty"`
	}
	deletions := make([]path, len(entries))
	for i, e := range entries {
		deletions[i] = path{Path: e.Path}
	}
	input := map[string]any{
		"branch":          map[string]string{"repositoryNameWithOwner": c.Repo, "branchName": pr.Head.Ref},
		"expectedHeadOid": pr.Head.SHA,
		"message":         map[string]string{"headline": foldMessage},
		"fileChanges": map[string]any{
			"additions": []path{{Path: changelog.File, Contents: base64.StdEncoding.EncodeToString([]byte(text))}},
			"deletions": deletions,
		},
	}
	data, _ := json.Marshal(map[string]any{"query": commitMutation, "variables": map[string]any{"input": input}})
	out, err := c.api(ctx, "graphql", "--jq", ".data.createCommitOnBranch.commit.oid", "--input", "-", string(data))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// contents is the REST path of a file or folder at ref.
func (c Client) contents(p, ref string) string {
	return "repos/" + c.Repo + "/contents/" + p + "?ref=" + ref
}

// raw returns a file's text at ref.
func (c Client) raw(ctx context.Context, p, ref string) (string, error) {
	out, err := c.api(ctx, "-H", "Accept: application/vnd.github.raw+json", c.contents(p, ref))
	return string(out), err
}

// lines runs a gh api call and splits its output into non-empty lines.
func (c Client) lines(ctx context.Context, args ...string) ([]string, error) {
	out, err := c.api(ctx, args...)
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
