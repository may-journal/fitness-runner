// Command fitness-check-issue-link-once keeps each Issue's timeline readable:
// every commit that links an Issue adds an entry there, so a commit may link
// an Issue only once per branch.
//
// It fails a proposed commit message that links an Issue when an earlier
// commit on the branch, since its merge base with origin's default branch,
// already links it. It also fails when the branch's open PR body links it,
// since the PR already shows on the Issue's timeline.
//
// It reads the message from the context-inline `--message` value and passes
// inert without one, so full-suite runs leave it alone. With no PR, or no
// network, it warns and skips the PR rule.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

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

var (
	// urlLinkRe matches an Issue or PR URL on github.com.
	urlLinkRe = regexp.MustCompile(`https?://github\.com/([\w.-]+/[\w.-]+)/(?:issues|pull)/(\d+)`)
	// qualifiedLinkRe matches `owner/repo#NN`.
	qualifiedLinkRe = regexp.MustCompile(`(?:^|[^\w/.-])([\w.-]+/[\w.-]+)#(\d+)\b`)
	// bareLinkRe matches `#NN` not glued to a word, path, or entity.
	bareLinkRe = regexp.MustCompile(`(?:^|[^\w&/#.-])#(\d+)\b`)
)

// earlier is one commit already on the branch.
type earlier struct {
	sha     string
	subject string
	message string
}

// openPR is the branch's open pull request, when it has one.
type openPR struct {
	number int
	body   string
}

func run(root string, _ []string) (checkkit.Result, error) {
	msg, provided := checkkit.CtxMessage()
	if !provided {
		return checkkit.Pass(0), nil
	}
	repo := gitx.OriginSlug(root)
	if len(links(stripComments(msg), repo)) == 0 {
		return checkkit.Pass(1), nil
	}
	return judge(msg, repo, branchCommits(root), branchPR()), nil
}

// judge fails each Issue the message links that an earlier branch commit or
// the open PR body already links.
func judge(msg, repo string, prior []earlier, pr *openPR) checkkit.Result {
	var errs []string
	for _, key := range links(stripComments(msg), repo) {
		if e := repeatOf(key, repo, prior, pr); e != "" {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}

// repeatOf names what already links key, or returns empty when nothing does.
func repeatOf(key, repo string, prior []earlier, pr *openPR) string {
	if c, ok := linkedBy(key, repo, prior); ok {
		return fmt.Sprintf("%s is already linked by commit %s (%q); drop the reference from this message", key, c.sha[:min(7, len(c.sha))], c.subject)
	}
	if pr != nil && contains(links(pr.body, repo), key) {
		return fmt.Sprintf("%s is already linked by open PR #%d; drop the reference from this message", key, pr.number)
	}
	return ""
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
	out := make([]earlier, 0, len(shas))
	for _, sha := range shas {
		m := gitx.Message(root, sha)
		out = append(out, earlier{sha: sha, subject: strings.SplitN(strings.TrimSpace(m), "\n", 2)[0], message: m})
	}
	return out
}

// branchPR returns the current branch's open PR through the gh CLI, or nil
// with a warning when there is none or gh cannot answer.
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
