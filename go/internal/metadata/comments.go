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
	} `json:"user"`
}

// commentTargets is the event's comment, or every recent comment on
// workflow_dispatch, minus the ones comment-check leaves alone.
func commentTargets(ev ghEvent, gh aftereffect.Client, _ blobChecker) ([]target, error) {
	all, err := eventComments(ev, gh)
	if err != nil {
		return nil, err
	}
	ignored := ignoredAuthors()
	var judged []target
	for _, t := range all {
		if why := skipReason(t, ignored); why != "" {
			fmt.Printf("Skipped %s: %s.\n", t.Comment.URL, why)
			continue
		}
		judged = append(judged, t)
	}
	return judged, nil
}

// eventComments reads the comments the event names.
func eventComments(ev ghEvent, gh aftereffect.Client) ([]target, error) {
	switch ev.name {
	case "workflow_dispatch":
		return gh.RecentComments(sweepDays())
	case "pull_request_review":
		return reviewComments(ev, gh)
	}
	return oneComment(ev)
}

// reviewComments reads every inline comment of the submitted review, so
// one run judges the whole review.
func reviewComments(ev ghEvent, gh aftereffect.Client) ([]target, error) {
	if ev.Review == nil || ev.PullRequest == nil {
		return nil, fmt.Errorf("no review in the %q event", ev.name)
	}
	return gh.ReviewComments(ev.PullRequest.Number, ev.Review.ID)
}

// oneComment is the single comment an issue_comment or
// pull_request_review_comment event carries.
func oneComment(ev ghEvent) ([]target, error) {
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
	})}, nil
}

// KnownBots are the org's automation accounts, whose comments no one can fix;
// the comment-check workflows repeat this list, so change both.
var KnownBots = []string{"github-actions[bot]", "may-journal-automation[bot]", "release-please[bot]", "dependabot[bot]"}

// skipReason says why comment-check leaves t alone, or "" to judge it. Fitness
// posts as aftereffect.BotLogin, which is always ignored; a marker from anyone else is judged.
func skipReason(t target, ignored []string) string {
	if slices.Contains(ignored, t.Comment.Author) {
		return "its author is a known bot"
	}
	return ""
}

// ignoredAuthors is FITNESS_COMMENT_IGNORE_AUTHORS split on commas, or
// KnownBots when it names none, plus the login fitness itself posts as.
func ignoredAuthors() []string {
	out := []string{aftereffect.BotLogin()}
	for _, a := range strings.Split(os.Getenv("FITNESS_COMMENT_IGNORE_AUTHORS"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	if len(out) == 1 {
		out = append(out, KnownBots...)
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
