package aftereffect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Comment is one issue or pull request review comment that comment-check
// judges.
type Comment struct {
	ID     int64
	Author string
	URL    string
	// Review marks a pull request review comment, whose replies go in its
	// thread; Root is that thread's first comment.
	Review bool
	Root   int64
}

// kind is the REST segment the comment lives under: issues or pulls.
func (c Comment) kind() string {
	if c.Review {
		return "pulls"
	}
	return "issues"
}

// path is the REST path of the comment, or of one under it.
func (c Comment) path(rest ...string) string {
	return strings.Join(append([]string{c.kind(), "comments", strconv.FormatInt(c.ID, 10)}, rest...), "/")
}

// noticePrefix starts the marker that ties a reply to the comment it judges.
const noticePrefix = "<!-- fitness:comment-check:"

// Notice is one comment-check reply already on GitHub.
type Notice struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// botLogin is the account whose replies count as comment-check notices;
// FITNESS_BOT_LOGIN overrides the Actions token's login.
func botLogin() string {
	if login := os.Getenv("FITNESS_BOT_LOGIN"); login != "" {
		return login
	}
	return "github-actions[bot]"
}

// CommentReply keeps one reply per failing comment, naming its author and
// violations, with a thumbs-down reaction. When the comment is fixed, the
// reply becomes a passing note and the reaction goes; a comment that always
// passed gets nothing.
func CommentReply(gh Client, t Target, errs []string) error {
	c := *t.Comment
	prior, err := gh.notice(t.Number, c)
	if err != nil {
		return err
	}
	if len(errs) == 0 {
		return passed(gh, c, prior)
	}
	return failed(gh, t.Number, c, prior, errs)
}

// failed writes or refreshes the reply, then reacts.
func failed(gh Client, n int, c Comment, prior *Notice, errs []string) error {
	mark := noticeMarker(c.ID, strings.Join(errs, "\n"))
	if err := gh.writeNotice(n, c, prior, failNotice(c, errs, mark), mark); err != nil {
		return err
	}
	_, err := gh.api("-X", "POST", "--silent", "-f", "content=-1", c.path("reactions"))
	return err
}

// passed turns an earlier failing reply into a passing note and removes the
// reaction.
func passed(gh Client, c Comment, prior *Notice) error {
	if prior == nil {
		return nil
	}
	mark := noticeMarker(c.ID, "pass")
	if err := gh.writeNotice(0, c, prior, passNotice(c, mark), mark); err != nil {
		return err
	}
	return gh.dropReaction(c)
}

// writeNotice edits prior to body, or posts body as a new reply, unless
// prior already carries mark.
func (g Client) writeNotice(n int, c Comment, prior *Notice, body, mark string) error {
	switch {
	case prior != nil && strings.Contains(prior.Body, mark):
		return nil
	case prior != nil:
		_, err := g.api("-X", "PATCH", "-f", "body="+body, noticePath(c, prior.ID))
		return err
	case c.Review:
		_, err := g.api("-X", "POST", "-f", "body="+body, fmt.Sprintf("pulls/%d/comments/%d/replies", n, c.Root))
		return err
	}
	return g.Comment(n, body)
}

// noticePath is the REST path of reply id, which lives where c does.
func noticePath(c Comment, id int64) string {
	return Comment{ID: id, Review: c.Review}.path()
}

// noticeMarker ties a reply to comment id and the verdict it states.
func noticeMarker(id int64, verdict string) string {
	sum := sha256.Sum256([]byte(verdict))
	return noticePrefix + strconv.FormatInt(id, 10) + ":" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// failNotice lists the violations and mentions the author.
func failNotice(c Comment, errs []string, mark string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "❌ %s [This comment](%s) breaks the comment rules:\n\n", mentions(c.Author), c.URL)
	b.WriteString(BulletList(errs))
	b.WriteString("\nRun `fitness comment-check --file <draft>` before posting, then edit the comment. This reply follows the edit.")
	return b.String() + footer(mark)
}

// passNotice replaces a failing reply once the comment is fixed.
func passNotice(c Comment, mark string) string {
	return fmt.Sprintf("%s [This comment](%s) now passes the comment rules (checked %s).", passIcon, c.URL,
		time.Now().UTC().Format("2006-01-02 15:04 UTC")) + footer(mark)
}

// notice is the bot's reply already judging c, or nil.
func (g Client) notice(n int, c Comment) (*Notice, error) {
	jq := fmt.Sprintf(`.[] | select(.user.login == %q and (.body // "" | contains(%q))) | {id, body}`,
		botLogin(), noticePrefix+strconv.FormatInt(c.ID, 10)+":")
	out, err := g.api("--paginate", "--jq", jq, fmt.Sprintf("%s/%d/comments?per_page=100", c.kind(), n))
	if err != nil {
		return nil, err
	}
	found, err := decodeLines[Notice](out)
	if err != nil || len(found) == 0 {
		return nil, err
	}
	return &found[0], nil
}

// dropReaction removes the bot's thumbs-down from c.
func (g Client) dropReaction(c Comment) error {
	jq := fmt.Sprintf(`.[] | select(.user.login == %q) | .id`, botLogin())
	out, err := g.api("--paginate", "--jq", jq, c.path("reactions")+"?content=-1&per_page=100")
	if err != nil {
		return err
	}
	for _, id := range strings.Fields(string(out)) {
		if _, err := g.api("-X", "DELETE", "--silent", c.path("reactions", id)); err != nil {
			return err
		}
	}
	return nil
}

// sweptComment is a comment as the repository-wide comment lists return it.
type sweptComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Type      string `json:"type"`
	Parent    string `json:"parent"`
	InReplyTo int64  `json:"in_reply_to_id"`
}

// sweepJQ shapes each listed comment as a sweptComment.
const sweepJQ = `.[] | {id, body: (.body // ""), url: .html_url, author: .user.login, type: .user.type,
	parent: (.issue_url // .pull_request_url), in_reply_to_id: (.in_reply_to_id // 0)}`

// RecentComments lists the issue and review comments created or edited in
// the last days days, oldest first within each kind.
func (g Client) RecentComments(days int) ([]Target, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	var all []Target
	for _, kind := range []string{"issues", "pulls"} {
		out, err := g.api("--paginate", "--jq", sweepJQ, kind+"/comments?per_page=100&since="+since)
		if err != nil {
			return nil, err
		}
		found, err := decodeLines[sweptComment](out)
		if err != nil {
			return nil, err
		}
		all = append(all, sweptTargets(found, kind == "pulls")...)
	}
	return all, nil
}

// sweptTargets turns listed comments into targets.
func sweptTargets(found []sweptComment, review bool) []Target {
	targets := make([]Target, len(found))
	for i, s := range found {
		targets[i] = CommentTarget(parentNumber(s.Parent), s.Body, Comment{
			ID: s.ID, Author: s.Author, URL: s.URL, Review: review, Root: s.InReplyTo,
		}, s.Type)
	}
	return targets
}

// parentNumber is the issue or PR number at the end of an API URL.
func parentNumber(url string) int {
	n, _ := strconv.Atoi(url[strings.LastIndex(url, "/")+1:])
	return n
}

// CommentTarget builds the target for comment c on issue or PR n. A review
// reply's thread root defaults to the comment itself.
func CommentTarget(n int, body string, c Comment, authorType string) Target {
	if c.Root == 0 {
		c.Root = c.ID
	}
	return Target{Number: n, Body: body, Comment: &c, AuthorType: authorType}
}
