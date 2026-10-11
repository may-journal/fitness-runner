package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

type branchPR struct {
	aftereffect.Target
	Head    string `json:"head"`
	Source  string `json:"source"`
	Base    string `json:"base"`
	BaseSHA string `json:"baseSHA"`
	URL     string `json:"url"`
}

func discoverBranchPR(ctx context.Context, root string) (*branchPR, string, error) {
	branch, err := currentBranch(ctx, root)
	if err != nil || branch == "" {
		return nil, "", err
	}
	source, repo, err := branchRepository(ctx, root)
	if err != nil {
		return nil, "", err
	}
	owner, _, _ := strings.Cut(source, "/")
	path := "repos/" + repo + "/pulls?state=open&head=" + url.QueryEscape(owner+":"+branch) + "&per_page=100"
	filter := `.[] | {number, title, body: (.body // ""), head: .head.ref, source: .head.repo.full_name, base: .base.ref, baseSHA: .base.sha, url: .html_url}`
	out, err := branchOutput(ctx, root, "api", "--paginate", "--jq", filter, path)
	if err != nil {
		return nil, "", err
	}
	pr, err := matchingPR(out, source, branch)
	return pr, repo, err
}

// branchRepository resolves the local source and the base repo selected by gh.
func branchRepository(ctx context.Context, root string) (string, string, error) {
	source := gitx.OriginSlug(root)
	if source == "" {
		return "", "", fmt.Errorf("no GitHub origin remote; configure the branch repository first")
	}
	out, err := branchOutput(ctx, root, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	return source, strings.TrimSpace(string(out)), err
}

func currentBranch(ctx context.Context, root string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "symbolic-ref", "--quiet", "--short", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
		fmt.Println("branch-check: detached HEAD; skipped.")
		return "", nil
	}
	return "", fmt.Errorf("cannot read the current git branch: %w", err)
}

func matchingPR(out []byte, source, branch string) (*branchPR, error) {
	dec := json.NewDecoder(bytes.NewReader(out))
	var matches []*branchPR
	for dec.More() {
		var pr branchPR
		if err := dec.Decode(&pr); err != nil {
			return nil, fmt.Errorf("read GitHub PR response: %w", err)
		}
		if pr.Head != branch || !strings.EqualFold(pr.Source, source) {
			continue
		}
		matches = append(matches, &pr)
	}
	return selectBranchPR(matches, source, branch)
}

func selectBranchPR(matches []*branchPR, source, branch string) (*branchPR, error) {
	switch len(matches) {
	case 0:
		fmt.Printf("branch-check: no open PR for %s:%s; skipped.\n", source, branch)
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple open PRs match %s:%s; choose one PR per branch", source, branch)
	}
}
