package aftereffect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// dupePrefix starts the marker of the one note dupe-check keeps on an issue;
// a hash of the note's text follows it.
const dupePrefix = MarkerPrefix + "dupe-check:"

// DupeComment keeps one note on an issue listing the older issues it
// resembles, edited in place as the issue changes, and closes the issue as
// a duplicate when the command asks. An issue that never resembled another
// gets nothing; one that stops resembling them gets its note cleared.
func DupeComment(gh Client, t Target, _ []string) error {
	prior, err := gh.markedComment("issues", t.Number, dupePrefix)
	if err != nil {
		return err
	}
	if err := writeDupeNote(gh, t, prior); err != nil {
		return err
	}
	if !t.CloseDupe {
		return nil
	}
	return gh.closeAsDuplicate(t.Number, t.Dupes[0].ID)
}

// writeDupeNote posts the note, or edits prior to it, unless prior already
// says the same.
func writeDupeNote(gh Client, t Target, prior *Notice) error {
	if len(t.Dupes) == 0 && prior == nil {
		return nil
	}
	text := dupeText(t)
	sum := sha256.Sum256([]byte(text))
	return putNote(gh, t.Number, prior, text, dupePrefix+hex.EncodeToString(sum[:])[:12]+" -->")
}

// putNote edits prior to text under mark, or posts it on issue n when there
// is no prior note, unless prior already carries mark.
func putNote(gh Client, n int, prior *Notice, text, mark string) error {
	switch {
	case prior == nil:
		return gh.Comment(n, text+footer(mark))
	case strings.Contains(prior.Body, mark):
		return nil
	}
	return gh.EditComment(prior.ID, text+footer(mark))
}

// dupeText is the note: the matches, best first, or that none remain.
func dupeText(t Target) string {
	if len(t.Dupes) == 0 {
		return passIcon + " This issue no longer resembles an earlier one."
	}
	var b strings.Builder
	if t.CloseDupe {
		fmt.Fprintf(&b, "🔁 Closed as a duplicate of #%d. Continue there, or reopen this issue if it differs.\n\n", t.Dupes[0].Number)
	} else {
		b.WriteString("🔁 This issue may repeat an earlier one. If it does, close it as a duplicate and continue there.\n\n")
	}
	for _, d := range t.Dupes {
		fmt.Fprintf(&b, "- #%d (score %.2f)\n", d.Number, d.Score)
	}
	b.WriteString("\nThis note follows edits to the issue.")
	return b.String()
}

// closeAsDuplicate closes issue n as a duplicate of the issue whose
// database id is of.
func (g Client) closeAsDuplicate(n int, of int64) error {
	_, err := g.api("-X", "PATCH", "--silent", "-f", "state=closed", "-f", "state_reason=duplicate",
		"-F", "duplicate_issue_id="+strconv.FormatInt(of, 10), "issues/"+strconv.Itoa(n))
	return err
}

// issuesQuery lists every issue, oldest first, with its parent, which only
// GraphQL reads in one list.
const issuesQuery = `query($owner: String!, $name: String!, $endCursor: String) {
  repository(owner: $owner, name: $name) {
    issues(first: 100, after: $endCursor, orderBy: {field: CREATED_AT, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes { databaseId number title body state stateReason closedAt parent { number } author { login __typename } }
    }
  }
}`

// issuesFilter shapes each listed issue as a Target.
const issuesFilter = `.data.repository.issues.nodes[] | {id: .databaseId, number, title, body: (.body // ""),
	state: (.state | ascii_downcase), state_reason: ((.stateReason // "") | ascii_downcase),
	closed_at: (.closedAt // ""), parentNumber: (.parent.number // 0),
	opener: (if .author.__typename == "Bot" then .author.login + "[bot]" else (.author.login // "") end)}`

// Issues lists every issue in the repository, open and closed, oldest
// first, each with its parent's number.
func (g Client) Issues() ([]Target, error) {
	owner, name, _ := strings.Cut(g.repo, "/")
	out, err := g.gh("api", "graphql", "--paginate",
		"-f", "query="+issuesQuery, "-F", "owner="+owner, "-F", "name="+name, "--jq", issuesFilter)
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}
