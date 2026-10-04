package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checklist"
)

// runCloseCheck handles an `issues: closed` event: an issue closed as
// completed while its checklist still has unchecked items is reopened, with
// one comment naming the items and the people behind the close.
func runCloseCheck() int {
	ev, err := readEvent()
	if err != nil {
		report.Error("fitness close-check", err)
		return 1
	}
	gh, err := newGHClient()
	if err != nil {
		report.Error("fitness close-check", err)
		return 1
	}
	return closeCheck(ev, gh)
}

// closeCheck is runCloseCheck past the environment.
func closeCheck(ev ghEvent, gh githubAPI) int {
	issue, err := closedIssue(ev)
	if err != nil {
		report.Error("fitness close-check", err)
		return 1
	}
	if skippedReason(issue.StateReason) {
		closeOutcome(issue, fmt.Sprintf("Closed as %s; unfinished items are allowed.", issue.StateReason))
		return 0
	}
	items := checklist.Unchecked(issue.Body)
	if len(items) == 0 {
		closeOutcome(issue, "Closed with every checklist item done.")
		return 0
	}
	if err := reopenOnce(issue, items, ev.Sender.Login, gh); err != nil {
		report.Error(fmt.Sprintf("fitness close-check #%d", issue.Number), err)
		report.WriteSummary(targetLink("issues", issue.Number))
		return 1
	}
	closeOutcome(issue, fmt.Sprintf("Reopened: %d unchecked item(s).", len(items)))
	return 0
}

// closedIssue is the issue an `issues: closed` event carries.
func closedIssue(ev ghEvent) (target, error) {
	if ev.Issue == nil || ev.Action != "closed" {
		return target{}, fmt.Errorf("no closed issue in the %q event", ev.name)
	}
	return *ev.Issue, nil
}

// skippedReason reports whether a close reason means the work was dropped,
// so unchecked items are expected.
func skippedReason(reason string) bool {
	return reason == "not_planned" || reason == "duplicate"
}

// reopenOnce reopens the issue and comments, unless this close already has
// a comment, so a re-run stays quiet.
func reopenOnce(issue target, items []string, closedBy string, gh githubAPI) error {
	mark := closeMarker(issue)
	seen, err := hasMarker(issue.Number, mark, gh)
	if err != nil || seen {
		return err
	}
	pr, err := gh.closingPR(issue.Number)
	if err != nil {
		return err
	}
	if err := gh.reopen(issue.Number); err != nil {
		return err
	}
	return gh.comment(issue.Number, closeComment(items, mentions(closedBy, pr.Author, pr.Merger), mark))
}

// closeMarker tags the comment with a hash of this close, its time and body.
func closeMarker(issue target) string {
	sum := sha256.Sum256([]byte(issue.ClosedAt + "\n" + issue.Body))
	return "<!-- fitness:close-check:" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// hasMarker reports whether a comment on issue n already carries mark.
func hasMarker(n int, mark string, gh githubAPI) (bool, error) {
	existing, err := gh.commentBodies(n)
	if err != nil {
		return false, err
	}
	for _, body := range existing {
		if strings.Contains(body, mark) {
			return true, nil
		}
	}
	return false, nil
}

// mentions is the @-mention line for the closer, and for a merge close the
// PR's author and merger, each once. Apps cannot be notified, so they are
// left out.
func mentions(logins ...string) string {
	var out []string
	seen := map[string]bool{}
	for _, l := range logins {
		if l == "" || strings.HasSuffix(l, "[bot]") || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, "@"+l)
	}
	return strings.Join(out, " ")
}

// closeComment explains the reopen, lists the unchecked items, and mentions
// the people behind the close.
func closeComment(items []string, who, mark string) string {
	var b strings.Builder
	b.WriteString("🔁 Reopened: this issue was closed with unchecked checklist items. ")
	b.WriteString("Finish or tick each one, or close it as not planned if the work was dropped.\n\n")
	b.WriteString(bulletList(items))
	if who != "" {
		b.WriteString("\ncc " + who + "\n")
	}
	b.WriteString("\n" + mark)
	return b.String()
}
