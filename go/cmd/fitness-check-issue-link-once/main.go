// Command fitness-check-issue-link-once keeps each Issue's timeline readable:
// every commit that links an Issue adds an entry there, so a branch links an
// Issue from one commit at most.
//
// It runs in two places. The commit-msg hook passes the proposed message as
// the context-inline `--message` value. The check fails it when an earlier
// commit on the branch, since its merge base with origin's default branch,
// links the same Issue, or when the branch's open PR body links it. With no
// PR, or no network, it warns and skips the PR rule.
//
// On a pull request in CI, it judges every branch commit, oldest first, by
// the same two rules. The PR rule counts only commits authored after the PR
// opened, so the one link written before it stays allowed. A failure stays
// red until the history is rewritten without the repeat.
//
// Anywhere else, such as a local suite run or a push to main, it passes inert.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "issue-link-once", ContextInlineArg: "--message", TimeoutMs: 20000},
		Run:      run,
	})
}

// amendEnv is set by the commit-msg hook when the commit amends HEAD, so HEAD
// is the commit being replaced rather than an earlier one.
const amendEnv = "FITNESS_COMMIT_AMEND"

// rewriteHint tells a CI failure how to clear it.
const rewriteHint = "rewrite that commit without the link, then force-push with --force-with-lease"

var (
	// urlLinkRe matches an Issue or PR URL on github.com.
	urlLinkRe = regexp.MustCompile(`https?://github\.com/([\w.-]+/[\w.-]+)/(?:issues|pull)/(\d+)`)
	// qualifiedLinkRe matches `owner/repo#NN`.
	qualifiedLinkRe = regexp.MustCompile(`(?:^|[^\w/.-])([\w.-]+/[\w.-]+)#(\d+)\b`)
	// bareLinkRe matches `#NN` not glued to a word, path, or entity.
	bareLinkRe = regexp.MustCompile(`(?:^|[^\w&/#.-])#(\d+)\b`)
)

// earlier is one commit on the branch.
type earlier struct {
	sha      string
	subject  string
	message  string
	authored time.Time
}

// openPR is the branch's open pull request, when it has one.
type openPR struct {
	number int
	body   string
	opened time.Time
	head   string
}

func run(root string, _ []string) (checkkit.Result, error) {
	repo := gitx.OriginSlug(root)
	if msg, provided := checkkit.CtxMessage(); provided {
		if len(links(stripComments(msg), repo)) == 0 {
			return checkkit.Pass(1), nil
		}
		return judge(msg, repo, branchCommits(root), branchPR()), nil
	}
	if pr, base, ok := eventPR(); ok {
		return judgeBranch(repo, prCommits(root, pr.head, base), pr), nil
	}
	return checkkit.Pass(0), nil
}

// judge fails each Issue the proposed message links that an earlier branch
// commit or the open PR body already links.
func judge(msg, repo string, prior []earlier, pr *openPR) checkkit.Result {
	var errs []string
	for _, key := range links(stripComments(msg), repo) {
		if e := repeatOf(key, repo, prior, pr, time.Now()); e != "" {
			errs = append(errs, e+"; drop the reference from this message")
		}
	}
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}

// judgeBranch applies the same rules to every commit of a pull request,
// oldest first, each against the commits before it.
func judgeBranch(repo string, commits []earlier, pr *openPR) checkkit.Result {
	var errs []string
	for i, c := range commits {
		errs = append(errs, commitRepeats(repo, c, commits[:i], pr)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(commits), errs...)
	}
	return checkkit.Pass(len(commits))
}

// commitRepeats reports each repeat link in one pull request commit.
func commitRepeats(repo string, c earlier, prior []earlier, pr *openPR) []string {
	var errs []string
	for _, key := range links(stripComments(c.message), repo) {
		if e := repeatOf(key, repo, prior, pr, c.authored); e != "" {
			errs = append(errs, fmt.Sprintf("commit %s (%q): %s; %s", short(c.sha), c.subject, e, rewriteHint))
		}
	}
	return errs
}

// repeatOf names what already links key, or returns empty when nothing does.
// The PR body counts only against a commit authored after the PR opened.
func repeatOf(key, repo string, prior []earlier, pr *openPR, authored time.Time) string {
	if c, ok := linkedBy(key, repo, prior); ok {
		return fmt.Sprintf("%s is already linked by commit %s (%q)", key, short(c.sha), c.subject)
	}
	if pr != nil && authored.After(pr.opened) && contains(links(pr.body, repo), key) {
		return fmt.Sprintf("%s is already linked by open PR #%d", key, pr.number)
	}
	return ""
}

func short(sha string) string {
	return sha[:min(7, len(sha))]
}

// linkedBy returns the first earlier commit whose message links key.
func linkedBy(key, repo string, prior []earlier) (earlier, bool) {
	for _, c := range prior {
		if contains(links(stripComments(c.message), repo), key) {
			return c, true
		}
	}
	return earlier{}, false
}

// links returns the distinct Issue links in text as `owner/repo#NN` keys, in
// first-seen order. A bare `#NN` belongs to repo.
func links(text, repo string) []string {
	var keys []string
	add := func(slug, n string) {
		if k := strings.ToLower(slug) + "#" + n; !contains(keys, k) {
			keys = append(keys, k)
		}
	}
	for _, m := range urlLinkRe.FindAllStringSubmatch(text, -1) {
		add(m[1], m[2])
	}
	for _, m := range qualifiedLinkRe.FindAllStringSubmatch(urlLinkRe.ReplaceAllString(text, " "), -1) {
		add(m[1], m[2])
	}
	for _, m := range bareLinkRe.FindAllStringSubmatch(urlLinkRe.ReplaceAllString(text, " "), -1) {
		add(repo, m[1])
	}
	return keys
}

// stripComments drops the lines git removes from a commit message: comment
// lines and everything below the scissors line.
func stripComments(msg string) string {
	var kept []string
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(line, "# ------------------------ >8") {
			break
		}
		if !strings.HasPrefix(line, "#") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// branchCommits lists the commits already on the branch, leaving out HEAD
// when the new commit amends it.
func branchCommits(root string) []earlier {
	shas := gitx.BranchCommits(root, "HEAD", gitx.DefaultBase(root))
	if os.Getenv(amendEnv) == "1" && len(shas) > 0 {
		shas = shas[1:]
	}
	return describeCommits(root, shas)
}

// prCommits lists a pull request's commits, oldest first.
func prCommits(root, head, base string) []earlier {
	shas := gitx.BranchCommits(root, head, base)
	for i, j := 0, len(shas)-1; i < j; i, j = i+1, j-1 {
		shas[i], shas[j] = shas[j], shas[i]
	}
	return describeCommits(root, shas)
}

// describeCommits reads each commit's message, subject, and author date.
// The author date survives a rebase, so rewriting history keeps it.
func describeCommits(root string, shas []string) []earlier {
	out := make([]earlier, 0, len(shas))
	for _, sha := range shas {
		m := gitx.Message(root, sha)
		authored, _ := time.Parse(time.RFC3339, strings.TrimSpace(gitx.Format(root, sha, "%aI")))
		out = append(out, earlier{sha: sha, subject: strings.SplitN(strings.TrimSpace(m), "\n", 2)[0], message: m, authored: authored})
	}
	return out
}

// branchPR returns the current branch's open PR through the gh CLI, or nil
// with a warning when there is none or gh cannot answer. It returns only
// open PRs, and the proposed commit is always newer, so opened stays zero.
func branchPR() *openPR {
	out, err := exec.Command("gh", "pr", "view", "--json", "number,state,body").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "issue-link-once: no open PR found for this branch, so the PR rule was skipped")
		return nil
	}
	var pr struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Body   string `json:"body"`
	}
	if json.Unmarshal(out, &pr) != nil || pr.State != "OPEN" {
		return nil
	}
	return &openPR{number: pr.Number, body: pr.Body}
}

// eventPR reads the pull request from the GitHub Actions event payload, with
// the base ref its commits are counted from. ok is false outside a pull
// request run.
func eventPR() (pr *openPR, base string, ok bool) {
	ref := os.Getenv("GITHUB_BASE_REF")
	path := os.Getenv("GITHUB_EVENT_PATH")
	if ref == "" || path == "" {
		return nil, "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false
	}
	return parseEvent(data, "origin/"+ref)
}

// parseEvent is the pure payload decoder behind eventPR.
func parseEvent(data []byte, base string) (*openPR, string, bool) {
	var ev struct {
		PullRequest *struct {
			Number    int       `json:"number"`
			Body      string    `json:"body"`
			CreatedAt time.Time `json:"created_at"`
			Head      struct {
				Sha string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
	}
	if json.Unmarshal(data, &ev) != nil || ev.PullRequest == nil || ev.PullRequest.Head.Sha == "" {
		return nil, "", false
	}
	p := ev.PullRequest
	return &openPR{number: p.Number, body: p.Body, opened: p.CreatedAt, head: p.Head.Sha}, base, true
}
