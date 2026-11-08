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
	// Repo is the base repo's owner/name, as the pulls read names it.
	Repo string `json:"repo"`
}

// discoverBranchPR finds the open PR from the local source's branch in the
// base repo gh selects. With one remote and no gh choice, gh reads the
// remote's repo, so one pulls read finds both.
func discoverBranchPR(ctx context.Context, root, branch string) (*branchPR, string, error) {
	source := gitx.OriginSlug(root)
	if source == "" {
		return nil, "", fmt.Errorf("no GitHub origin remote; configure the branch repository first")
	}
	owner, _, _ := strings.Cut(source, "/")
	query := "/pulls?state=open&head=" + url.QueryEscape(owner+":"+branch) + "&per_page=100"
	repo, path := "", "repos/{owner}/{repo}"+query
	if !oneUnresolvedRemote(ctx, root) {
		var err error
		if repo, err = branchRepository(ctx, root); err != nil {
			return nil, "", err
		}
		path = "repos/" + repo + query
	}
	out, err := branchPulls(ctx, root, path)
	if err != nil {
		return nil, "", err
	}
	pr, err := matchingPR(out, source, branch)
	return pr, cmp.Or(repo, baseRepo(pr, source)), err
}

// baseRepo is the base repo the pull request names, else source, the
// remote's repo that gh read the pulls from. With no pull request it is empty.
func baseRepo(pr *branchPR, source string) string {
	if pr == nil {
		return ""
	}
	return cmp.Or(pr.Repo, source)
}

// branchRepository is the base repo gh selects.
func branchRepository(ctx context.Context, root string) (string, error) {
	out, err := branchOutput(ctx, root, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	return strings.TrimSpace(string(out)), err
}

// branchPulls reads the open PRs at path.
func branchPulls(ctx context.Context, root, path string) ([]byte, error) {
	filter := `.[] | {number, title, body: (.body // ""), head: .head.ref, source: .head.repo.full_name, base: .base.ref, baseSHA: .base.sha, url: .html_url, repo: .base.repo.full_name}`
	return branchOutput(ctx, root, "api", "--paginate", "--jq", filter, path)
}

// oneUnresolvedRemote reports whether the checkout has one remote and no
// `gh repo set-default`, so the pulls read needs no `gh repo view`.
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
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple open PRs match %s:%s; choose one PR per branch", source, branch)
	}
}
