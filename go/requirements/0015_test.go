package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// closedAt0015 is when the test's issue was closed.
const closedAt0015 = "2026-10-09T12:00:00Z"

// unfinished is an issue body with one unchecked item.
const unfinished = "- [x] Write the doc.\n- [ ] Write the tests.\n"

// closed0015 is the `issues: closed` event GitHub sends when closer-login
// closes issue 21 with body for reason.
func closed0015(t *testing.T, body, reason string) []string {
	t.Helper()
	issue := map[string]any{"number": 21, "body": body, "state_reason": reason, "closed_at": closedAt0015}
	return event0014(t, "issues", map[string]any{"action": "closed", "issue": issue, "sender": map[string]any{"login": "closer-login"}})
}

// closeCheck0015 runs close-check on the event, GitHub answering from saved.
func closeCheck0015(t *testing.T, event []string, saved []response) (replays, string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, event...), "close-check")
	return rp, out, code
}

// comments0015 is the saved comment list on issue 21, as gh prints it
// through fitness's filter: one object per line.
func comments0015(lines string) response {
	return response{Match: []string{"repos/" + repo0014 + "/issues/21/comments?per_page=100"}, Stdout: lines}
}

// quiet0015 asserts close-check left issue 21 closed and uncommented.
func quiet0015(t *testing.T, rp replays) {
	t.Helper()
	if rp.called("gh", "-X PATCH") || rp.called("gh", "-X POST") {
		t.Errorf("want the issue left closed and quiet, got gh calls %q", rp.calls("gh"))
	}
}

func Test0015_1(t *testing.T) {
	t.Parallel()
	rp, out, code := closeCheck0015(t, closed0015(t, unfinished, "completed"), []response{
		{Match: []string{"-X POST", "issues/21/comments"}, Stdout: `{"id":901,"body":"🔁 Reopened"}`},
		comments0015(""),
		{Match: []string{"graphql", "ClosedEvent"}, Stdout: `{"author":"","merger":""}` + "\n"},
		{Match: []string{"-X PATCH", "state=open", "repos/" + repo0014 + "/issues/21"}, Stdout: `{"number":21,"state":"open"}`},
	})
	sees(t, out, code, 0, `unchecked item: "Write the tests."`)
	if !rp.called("gh", "-X PATCH", "state=open", "issues/21") {
		t.Errorf("want issue 21 reopened, got gh calls %q", rp.calls("gh"))
	}
	if !rp.called("gh", "-X POST", "issues/21/comments", `"Write the tests."`, "@closer-login") {
		t.Errorf("want one comment naming the item and @closer-login, got gh calls %q", rp.calls("gh"))
	}
}

func Test0015_2(t *testing.T) {
	t.Parallel()
	sum := sha256.Sum256([]byte(closedAt0015 + "\n" + unfinished))
	mark := "<!-- fitness:close-check:" + hex.EncodeToString(sum[:])[:12] + " -->"
	rp, out, code := closeCheck0015(t, closed0015(t, unfinished, "completed"), []response{
		comments0015(`{"body":"🔁 Reopened: this issue was closed with unchecked checklist items.\n\n` + mark + `"}` + "\n"),
	})
	sees(t, out, code, 0)
	if rp.called("gh", "-X POST") {
		t.Errorf("want no second comment, got gh calls %q", rp.calls("gh"))
	}
}

func Test0015_3(t *testing.T) {
	t.Parallel()
	rp, out, code := closeCheck0015(t, closed0015(t, unfinished, "not_planned"), nil)
	sees(t, out, code, 0, "closed as not_planned")
	quiet0015(t, rp)
}

func Test0015_4(t *testing.T) {
	t.Parallel()
	done := "- [x] Write the doc.\n- [x] Write the tests.\n"
	rp, out, code := closeCheck0015(t, closed0015(t, done, "completed"), nil)
	sees(t, out, code, 0, "Passed.")
	quiet0015(t, rp)
}
