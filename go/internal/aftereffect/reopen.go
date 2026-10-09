package aftereffect

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ReopenUnfinished reopens an issue closed with unfinished items, once per
// close, with one comment listing them and mentioning the people behind the
// close. A re-run on the same close stays quiet.
func ReopenUnfinished(gh Client, t Target, errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	mark := closeMarker(t)
	seen, err := hasMarker(gh, t.Number, mark)
	if err != nil || seen {
		return err
	}
	return reopen(gh, t, errs, mark)
}

// reopen reopens the issue and comments, naming the people behind the close.
func reopen(gh Client, t Target, errs []string, mark string) error {
	pr, err := gh.ClosingPR(t.Number)
	if err != nil {
		return err
	}
	if err := gh.Reopen(t.Number); err != nil {
		return err
	}
	return gh.Comment(t.Number, closeComment(errs, mentions(t.ClosedBy, pr.Author, pr.Merger), mark))
}

// closeMarker tags the comment with a hash of this close, its time and body.
func closeMarker(t Target) string {
	sum := sha256.Sum256([]byte(t.ClosedAt + "\n" + t.Body))
	return "<!-- fitness:close-check:" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// hasMarker reports whether a comment on issue n already carries mark.
func hasMarker(gh Client, n int, mark string) (bool, error) {
	existing, err := gh.CommentBodies(n)
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

// closeComment explains the reopen, lists the unfinished items, and
// mentions the people behind the close.
func closeComment(errs []string, who, mark string) string {
	var b strings.Builder
	b.WriteString("🔁 Reopened: this issue was closed with unchecked checklist items. ")
	b.WriteString("Finish or tick each one, or close it as not planned if the work was dropped.\n\n")
	b.WriteString(BulletList(errs))
	if who != "" {
		b.WriteString("\ncc " + who + "\n")
	}
	b.WriteString("\n" + mark)
	return b.String()
}
