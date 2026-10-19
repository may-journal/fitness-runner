package aftereffect

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// IssueVerdict comments an issue's verdict once per body version under its
// kind's marker, and labels the Issue fitness-valid or fitness-invalid. A
// target with no kind is a Plan.
func IssueVerdict(gh Client, t Target, errs []string) error {
	if err := postVerdict(gh, t, errs); err != nil {
		return err
	}
	return labelVerdict(gh, t.Number, len(errs) == 0)
}

// kindOf is the target's kind label, Plan when none is set.
func kindOf(t Target) string {
	if t.Kind == "" {
		return planKind
	}
	return t.Kind
}

// planKind is the kind plan-check judges, and whose earlier verdicts carry
// the legacy plan-structure marker.
const planKind = "Plan"

// postVerdict keeps one live verdict per body version. A pass that follows a
// pass edits that comment in place with a timestamp; anything else posts a
// new one. Every older verdict is then hidden as outdated.
func postVerdict(gh Client, t Target, errs []string) error {
	kind := kindOf(t)
	prior, err := gh.Verdicts(t.Number, kind)
	if err != nil {
		return err
	}
	keep, err := liveVerdict(gh, t.Number, kind, errs, markers(kind, t.Body), prior)
	if err != nil {
		return err
	}
	return hideOlder(gh, prior, keep)
}

// liveVerdict returns the index of the verdict judging this body version,
// writing it first when none does; -1 means a new comment was posted. The
// first marker is the one a new verdict carries; the rest are earlier names
// for it.
func liveVerdict(gh Client, n int, kind string, errs, marks []string, prior []Verdict) (int, error) {
	if i := indexOfMarker(prior, marks); i >= 0 {
		return i, nil
	}
	last := len(prior) - 1
	if len(errs) == 0 && last >= 0 && strings.HasPrefix(prior[last].Body, passIcon) {
		return last, gh.EditComment(prior[last].ID, validatedComment(marks[0]))
	}
	return -1, gh.Comment(n, verdictComment(kind, errs, marks[0]))
}

// indexOfMarker is the verdict already judging this body version, or -1.
func indexOfMarker(prior []Verdict, marks []string) int {
	for i, v := range prior {
		for _, mark := range marks {
			if strings.Contains(v.Body, mark) {
				return i
			}
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

// Labels issue-check keeps on each checked Issue.
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

// MarkerPrefixes are the comment-marker prefixes that name a kind's
// verdicts: `fitness:issue-check:<kind>:`, and for Plan also the earlier
// `fitness:plan-structure:`, so verdicts posted before issue-check still
// count and are hidden in turn.
func MarkerPrefixes(kind string) []string {
	prefixes := []string{"fitness:issue-check:" + KindID(kind) + ":"}
	if kind == planKind {
		prefixes = append(prefixes, "fitness:plan-structure:")
	}
	return prefixes
}

// KindID is a kind label as it appears in markers: lower case, with each
// run of spaces a hyphen.
func KindID(kind string) string {
	return strings.Join(strings.Fields(strings.ToLower(kind)), "-")
}

// markers tag a comment with a hash of the body it judged, one per marker
// prefix; the first is the one a new verdict carries.
func markers(kind, body string) []string {
	hash := shortHash(body)
	var marks []string
	for _, p := range MarkerPrefixes(kind) {
		marks = append(marks, "<!-- "+p+hash+" -->")
	}
	return marks
}

// shortHash is the first 12 hex digits of text's SHA-256, which every
// marker uses to name the text it judged.
func shortHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:12]
}

// passIcon starts every passing verdict.
const passIcon = "✅"

// validatedComment replaces a passing verdict when a later body passes too.
func validatedComment(mark string) string {
	return passIcon + " Validated (updated " + time.Now().UTC().Format("2006-01-02 15:04 UTC") + ")" + footer(mark)
}

// verdictComment is the verdict comment for an issue body of kind.
func verdictComment(kind string, errs []string, mark string) string {
	if len(errs) == 0 {
		return passIcon + " " + kind + " looks good." + footer(mark)
	}
	return "❌ " + kind + " needs work:\n" + strings.TrimSuffix(BulletList(errs), "\n") + footer(mark)
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
