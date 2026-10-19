package metadata

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

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

// discoverBranchPR finds the open PR from the local source's branch in the
// base repo gh selects. It asks gh for that repo while the pulls read runs.
func discoverBranchPR(ctx context.Context, root, branch string) (*branchPR, string, error) {
	source := gitx.OriginSlug(root)
	if source == "" {
		return nil, "", fmt.Errorf("no GitHub origin remote; configure the branch repository first")
	}
	owner, _, _ := strings.Cut(source, "/")
	query := "/pulls?state=open&head=" + url.QueryEscape(owner+":"+branch) + "&per_page=100"
	var repo string
	var out []byte
	var repoErr, pullsErr error
	if !oneUnresolvedRemote(ctx, root) {
		repo, repoErr = branchRepository(ctx, root)
		if repoErr == nil {
			out, pullsErr = branchPulls(ctx, root, "repos/"+repo+query)
		}
	} else {
		var wg sync.WaitGroup
		wg.Go(func() { repo, repoErr = branchRepository(ctx, root) })
		out, pullsErr = branchPulls(ctx, root, "repos/{owner}/{repo}"+query)
		wg.Wait()
	}
	if err := cmp.Or(repoErr, pullsErr); err != nil {
		return nil, "", err
	}
	pr, err := matchingPR(out, source, branch)
	return pr, repo, err
}

// branchRepository is the base repo gh selects.
func branchRepository(ctx context.Context, root string) (string, error) {
	out, err := branchOutput(ctx, root, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	return strings.TrimSpace(string(out)), err
}

// branchPulls reads the open PRs at path.
func branchPulls(ctx context.Context, root, path string) ([]byte, error) {
	filter := `.[] | {number, title, body: (.body // ""), head: .head.ref, source: .head.repo.full_name, base: .base.ref, baseSHA: .base.sha, url: .html_url}`
	return branchOutput(ctx, root, "api", "--paginate", "--jq", filter, path)
}

// oneUnresolvedRemote reports whether the checkout has exactly one remote and
// `gh repo set-default` never chose one. Only then do `gh repo view` and the
// {owner}/{repo} placeholders of `gh api` surely name the same repo, so the
// pulls read may start before gh repo view answers.
func oneUnresolvedRemote(ctx context.Context, root string) bool {
	out, err := exec.CommandContext(ctx, "git", "-C", root, "config", "--get-regexp", `^remote\.`).Output()
	return err == nil && os.Getenv("GH_REPO") == "" && remoteURLs(string(out)) == 1
}

// remoteURLs counts the remote URLs in `git config --get-regexp` output, or
// returns -1 when a remote carries a gh-resolved choice.
func remoteURLs(config string) int {
	urls := 0
	for _, line := range strings.Split(config, "\n") {
		key, _, _ := strings.Cut(line, " ")
		if strings.HasSuffix(key, ".gh-resolved") {
			return -1
		}
		if strings.HasSuffix(key, ".url") {
			urls++
		}
	}
	return urls
}

func currentBranch(ctx context.Context, root string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "symbolic-ref", "--quiet", "--short", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
		fmt.Fprintln(os.Stderr, "branch-check: detached HEAD; skipped.")
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
		fmt.Fprintf(os.Stderr, "branch-check: no open PR for %s:%s; skipped.\n", source, branch)
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple open PRs match %s:%s; choose one PR per branch", source, branch)
	}
}
