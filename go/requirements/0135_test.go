package requirements

import "testing"

// stayedOpen0135 asserts close-check closed and commented on nothing.
func stayedOpen0135(t *testing.T, rp replays) {
	t.Helper()
	if rp.called("gh", "-X PATCH") || rp.called("gh", "-X POST") {
		t.Errorf("want the parent left open and quiet, got gh calls %q", rp.calls("gh"))
	}
}

func Test0135_1(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", "> A parent.\n"), subIssues0134(30, 21, 22),
		issue0134(t, 22, "open", "", ""),
	})
	sees(t, out, code, 0)
	stayedOpen0135(t, rp)
}

func Test0135_2(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, mentionedBy0134(21, 30), []response{
		issue0134(t, 30, "open", "", "> Mentions #21 but lists no child.\n"), subIssues0134(30),
	}, noParent0134(30))
	sees(t, out, code, 0)
	stayedOpen0135(t, rp)
}

func Test0135_3(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", "> A parent.\n"), subIssues0134(30, 21, 22),
		issue0134(t, 22, "closed", "not_planned", ""),
	})
	sees(t, out, code, 0)
	stayedOpen0135(t, rp)
}

func Test0135_4(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", list0134(21)+"- [ ] Announce it.\n"), subIssues0134(30, 21),
	})
	sees(t, out, code, 0)
	stayedOpen0135(t, rp)
}

func Test0135_5(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "closed", "completed", "> A parent.\n"),
	})
	sees(t, out, code, 0)
	stayedOpen0135(t, rp)
}
