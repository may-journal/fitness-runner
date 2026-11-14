package gitx

import (
	"regexp"
	"slices"
	"strings"
)

var (
	// urlLinkRe matches an Issue or PR URL on github.com.
	urlLinkRe = regexp.MustCompile(`https?://github\.com/([\w.-]+/[\w.-]+)/(?:issues|pull)/(\d+)`)
	// qualifiedLinkRe matches `owner/repo#NN`.
	qualifiedLinkRe = regexp.MustCompile(`(?:^|[^\w/.-])([\w.-]+/[\w.-]+)#(\d+)\b`)
	// bareLinkRe matches `#NN` not glued to a word, path, or entity.
	bareLinkRe = regexp.MustCompile(`(?:^|[^\w&/#.-])#(\d+)\b`)
)

// MessageLinks returns the distinct Issue links a commit message makes, minus
// lines git drops, as IssueLinks keys them. The runner asks it whether a
// message reads the network, and the issue-link-once check judges each link.
func MessageLinks(msg, repo string) []string {
	return IssueLinks(stripMessageComments(msg), repo)
}

// IssueLinks returns the distinct Issue links in text as `owner/repo#NN` keys,
// in first-seen order. A bare `#NN` belongs to repo.
func IssueLinks(text, repo string) []string {
	var keys []string
	add := func(slug, n string) {
		if k := strings.ToLower(slug) + "#" + n; !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	for _, m := range urlLinkRe.FindAllStringSubmatch(text, -1) {
		add(m[1], m[2])
	}
	rest := urlLinkRe.ReplaceAllString(text, " ")
	for _, m := range qualifiedLinkRe.FindAllStringSubmatch(rest, -1) {
		add(m[1], m[2])
	}
	for _, m := range bareLinkRe.FindAllStringSubmatch(rest, -1) {
		add(repo, m[1])
	}
	return keys
}

// stripMessageComments drops the lines git removes from a commit message:
// comment lines and everything below the scissors line.
func stripMessageComments(msg string) string {
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
