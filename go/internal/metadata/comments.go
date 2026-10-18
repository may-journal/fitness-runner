package metadata

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
)

// CommentChecks judge one issue or PR comment: its own structure, then the
// prose and markdown checks every Plan and PR body gets.
var CommentChecks = append([]string{"comment-structure"}, docChecks...)

// defaultSweepDays is how far back a manual comment-check run looks, unless
// FITNESS_SWEEP_DAYS says otherwise.
const defaultSweepDays = 7

// eventComment is the comment an issue_comment or
// pull_request_review_comment event carries.
type eventComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	URL       string `json:"html_url"`
	InReplyTo int64  `json:"in_reply_to_id"`
	User      struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}

// commentTargets is the event's comment, or every recent comment on
// workflow_dispatch, minus the ones comment-check leaves alone.
func commentTargets(ev ghEvent, gh aftereffect.Client) ([]target, error) {
	all, err := eventComments(ev, gh)
	if err != nil {
		return nil, err
	}
	authors := listedAuthors()
	var judged []target
	for _, t := range all {
		if why := skipReason(t, authors); why != "" {
			fmt.Printf("Skipped %s: %s.\n", t.Comment.URL, why)
			continue
		}
		judged = append(judged, t)
	}
	return judged, nil
}

// eventComments reads the comments the event names.
func eventComments(ev ghEvent, gh aftereffect.Client) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.RecentComments(sweepDays())
	}
	parent := ev.Issue
	if ev.name == "pull_request_review_comment" {
		parent = ev.PullRequest
	}
	if ev.Comment == nil || parent == nil {
		return nil, fmt.Errorf("no comment in the %q event", ev.name)
	}
	c := ev.Comment
	return []target{aftereffect.CommentTarget(parent.Number, c.Body, aftereffect.Comment{
		ID: c.ID, Author: c.User.Login, URL: c.URL, Review: parent == ev.PullRequest, Root: c.InReplyTo,
	}, c.User.Type)}, nil
}

// skipReason says why comment-check leaves t alone, or "" to judge it:
// apps cannot fix a comment, fitness verdicts follow their own format, and
// a caller may judge only the authors it lists.
func skipReason(t target, authors []string) string {
	switch {
	case isApp(t):
		return "an app wrote it"
	case strings.Contains(t.Body, "<!-- fitness:"):
		return "it is a fitness verdict"
	case authors != nil && !slices.Contains(authors, t.Comment.Author):
		return "its author is not in FITNESS_COMMENT_AUTHORS"
	}
	return ""
}

// isApp reports whether an app, not a person or agent account, wrote t.
func isApp(t target) bool {
	return t.AuthorType == "Bot" || strings.HasSuffix(t.Comment.Author, "[bot]")
}

// listedAuthors is FITNESS_COMMENT_AUTHORS split on commas, or nil to judge
// every author.
func listedAuthors() []string {
	var out []string
	for _, a := range strings.Split(os.Getenv("FITNESS_COMMENT_AUTHORS"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

// sweepDays is FITNESS_SWEEP_DAYS when it is a positive number.
func sweepDays() int {
	if n, err := strconv.Atoi(os.Getenv("FITNESS_SWEEP_DAYS")); err == nil && n > 0 {
		return n
	}
	return defaultSweepDays
}
