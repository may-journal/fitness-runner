package aftereffect

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// dupePrefix starts the marker of the one note dupe-check keeps on an issue;
// a hash of the note's text follows it.
const dupePrefix = MarkerPrefix + "dupe-check:"

// DupeComment keeps one note on an issue listing the older issues it
// resembles, edited in place as the issue changes, and closes the issue as
// a duplicate when the command asks. It closes first, so a failed close
// never leaves a note saying it closed. An issue that never resembled
// another gets nothing; one that stops resembling them gets its note
// cleared.
func DupeComment(gh Client, t Target, _ []string) error {
	prior, err := gh.markedComment("issues", t.Number, dupePrefix)
	if err != nil {
		return err
	}
	if t.CloseDupe {
		if err := gh.closeAsDuplicate(t.Number, t.Dupes[0].ID); err != nil {
			return err
		}
	}
	return writeDupeNote(gh, t, prior)
}

// writeDupeNote posts the note, or edits prior to it, unless prior already
// says the same.
func writeDupeNote(gh Client, t Target, prior *Notice) error {
	if len(t.Dupes) == 0 && prior == nil {
		return nil
	}
	text := dupeText(t)
	return putNote(gh, t.Number, prior, text, dupePrefix+shortHash(text)+" -->")
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

// issuesQuery lists the issues in states, updated since $since, with each
// one's parent, which only GraphQL reads in one list. A null $since lists
// them all.
const issuesQuery = `query($owner: String!, $name: String!, $since: DateTime, $endCursor: String) {
  repository(owner: $owner, name: $name) {
    issues(first: 100, after: $endCursor, states: [%s], filterBy: {since: $since}) {
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

// Issues lists every open issue and every closed one updated since since,
// an RFC 3339 time, oldest first, each with its parent's number. Older
// closed issues are never read.
func (g Client) Issues(since string) ([]Target, error) {
	open, err := g.issuesIn("OPEN")
	if err != nil {
		return nil, err
	}
	closed, err := g.issuesIn("CLOSED", "-f", "since="+since)
	all := append(open, closed...)
	slices.SortFunc(all, func(a, b Target) int { return a.Number - b.Number })
	return all, err
}

// issuesIn lists the issues in state, with extra variables for the query.
func (g Client) issuesIn(state string, vars ...string) ([]Target, error) {
	owner, name, _ := strings.Cut(g.repo, "/")
	args := append([]string{"api", "graphql", "--paginate", "-f", "query=" + fmt.Sprintf(issuesQuery, state),
		"-f", "owner=" + owner, "-f", "name=" + name, "--jq", issuesFilter}, vars...)
	out, err := g.gh(args...)
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}
