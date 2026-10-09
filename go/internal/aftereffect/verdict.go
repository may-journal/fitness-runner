package aftereffect

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// PlanVerdict comments a Plan's verdict once per body version and labels the
// Issue fitness-valid or fitness-invalid.
func PlanVerdict(gh Client, t Target, errs []string) error {
	if err := postVerdict(gh, t, errs); err != nil {
		return err
	}
	return labelVerdict(gh, t.Number, len(errs) == 0)
}

// postVerdict keeps one live verdict per body version. A pass that follows a
// pass edits that comment in place with a timestamp; anything else posts a
// new one. Every older verdict is then hidden as outdated.
func postVerdict(gh Client, t Target, errs []string) error {
	mark := planMarker(t.Body)
	prior, err := gh.Verdicts(t.Number)
	if err != nil {
		return err
	}
	keep, err := liveVerdict(gh, t.Number, errs, mark, prior)
	if err != nil {
		return err
	}
	return hideOlder(gh, prior, keep)
}

// liveVerdict returns the index of the verdict judging this body version,
// writing it first when none does; -1 means a new comment was posted.
func liveVerdict(gh Client, n int, errs []string, mark string, prior []Verdict) (int, error) {
	if i := indexOfMarker(prior, mark); i >= 0 {
		return i, nil
	}
	last := len(prior) - 1
	if len(errs) == 0 && last >= 0 && strings.HasPrefix(prior[last].Body, passIcon) {
		return last, gh.EditComment(prior[last].ID, validatedComment(mark))
	}
	return -1, gh.Comment(n, planComment(errs, mark))
}

// indexOfMarker is the verdict already judging this body version, or -1.
func indexOfMarker(prior []Verdict, mark string) int {
	for i, v := range prior {
		if strings.Contains(v.Body, mark) {
			return i
		}
	}
	return -1
}

// hideOlder hides every verdict but the live one, skipping those already
// hidden.
func hideOlder(gh Client, prior []Verdict, keep int) error {
	for i, v := range prior {
		if i == keep || v.IsHidden {
			continue
		}
		if err := gh.HideComment(v.NodeID); err != nil {
			return err
		}
	}
	return nil
}

// Labels plan-check keeps on each checked Issue.
const (
	fitnessLabel = "fitness"
	validLabel   = "fitness-valid"
	invalidLabel = "fitness-invalid"
)

// labelColors are the colors a missing label is created with.
var labelColors = map[string]string{fitnessLabel: "5319e7", validLabel: "0e8a16", invalidLabel: "d93f0b"}

// labelVerdict adds the fitness label and the label matching the verdict,
// and removes the opposite verdict label.
func labelVerdict(gh Client, n int, valid bool) error {
	want, drop := validLabel, invalidLabel
	if !valid {
		want, drop = invalidLabel, validLabel
	}
	_, labels, err := gh.Issue(n)
	if err != nil {
		return err
	}
	if err := addMissing(gh, n, labels, []string{fitnessLabel, want}); err != nil {
		return err
	}
	if HasLabel(labels, drop) {
		return gh.RemoveLabel(n, drop)
	}
	return nil
}

// addMissing adds the wanted labels the Issue lacks, creating each in the
// repo first when it does not exist.
func addMissing(gh Client, n int, labels, want []string) error {
	var add []string
	for _, l := range want {
		if HasLabel(labels, l) {
			continue
		}
		if err := gh.EnsureLabel(l, labelColors[l]); err != nil {
			return err
		}
		add = append(add, l)
	}
	if len(add) == 0 {
		return nil
	}
	return gh.AddLabels(n, add)
}

// HasLabel reports whether labels contains name exactly.
func HasLabel(labels []string, name string) bool {
	for _, l := range labels {
		if l == name {
			return true
		}
	}
	return false
}

// planMarker tags a comment with a hash of the body it judged; the format
// matches the earlier JavaScript, so existing comments still count.
func planMarker(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// passIcon starts every passing verdict.
const passIcon = "✅"

// validatedComment replaces a passing verdict when a later body passes too.
func validatedComment(mark string) string {
	return passIcon + " Validated (updated " + time.Now().UTC().Format("2006-01-02 15:04 UTC") + ")" + footer(mark)
}

// planComment is the verdict comment for a Plan body.
func planComment(errs []string, mark string) string {
	if len(errs) == 0 {
		return passIcon + " Plan looks good." + footer(mark)
	}
	return "❌ Plan needs work:\n" + strings.TrimSuffix(BulletList(errs), "\n") + footer(mark)
}

// footer ends a verdict with its attribution and the body-version marker.
func footer(mark string) string {
	return "\n\n" + attribution() + "\n\n" + mark
}

// BulletList renders errs as a markdown list.
func BulletList(errs []string) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString("- " + report.EscapeMarkdown(e) + "\n")
	}
	return b.String()
}
