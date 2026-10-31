package requirements

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// finished0134 is the body of the child just closed, with every item ticked.
const finished0134 = "- [x] Write the doc.\n"

// list0134 is a parent body whose list names the issues.
func list0134(nums ...int) string {
	body := "> A parent.\n\n## What needs to happen\n\n"
	for _, n := range nums {
		body += fmt.Sprintf("- #%d\n", n)
	}
	return body
}

// issuePath0134 is the REST path of issue n.
func issuePath0134(n int) string {
	return fmt.Sprintf("repos/%s/issues/%d", repo0014, n)
}

// noParent0134 is GitHub's answer that issue n has no parent issue.
func noParent0134(n int) []response {
	return []response{
		{Match: []string{issuePath0134(n) + "/parent"}, Stderr: "gh: Not Found (HTTP 404)", Exit: 1},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: ""},
	}
}

// parentOf0134 is GitHub's answer that issue n has parent p as a sub-issue.
func parentOf0134(n, p int) []response {
	return []response{
		{Match: []string{issuePath0134(n) + "/parent"}, Stdout: fmt.Sprintf("{\"number\":%d,\"repo\":%q}\n", p, repo0014)},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: ""},
	}
}

// mentionedBy0134 is GitHub's answer that issue n has no parent but is named in
// the body of issue p.
func mentionedBy0134(n, p int) []response {
	return []response{
		{Match: []string{issuePath0134(n) + "/parent"}, Stderr: "gh: Not Found (HTTP 404)", Exit: 1},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: fmt.Sprintf("{\"number\":%d}\n", p)},
	}
}

// issue0134 is GitHub's answer for reading issue n: its state, close reason
// and body.
func issue0134(t *testing.T, n int, state, reason, body string) response {
	t.Helper()
	out, err := json.Marshal(map[string]string{"state": state, "reason": reason, "body": body})
	mustDo(t, err)
	return response{Match: []string{"{state, reason", issuePath0134(n)}, Stdout: string(out) + "\n"}
}

// subIssues0134 is GitHub's answer for the sub-issues of issue n.
func subIssues0134(n int, kids ...int) response {
	var out string
	for _, k := range kids {
		out += fmt.Sprintf("{\"number\":%d,\"repo\":%q}\n", k, repo0014)
	}
	return response{Match: []string{issuePath0134(n) + "/sub_issues"}, Stdout: out}
}

// writes0134 are the answers to commenting on and closing issue n.
func writes0134(n int) []response {
	return []response{
		{Match: []string{"-X POST", issuePath0134(n) + "/comments"}, Stdout: "{}"},
		{Match: []string{"-X PATCH", "state=closed", issuePath0134(n)}, Stdout: "{}"},
	}
}

// closeParents0134 runs close-check on the closing of issue 21 as completed.
func closeParents0134(t *testing.T, saved ...[]response) (replays, string, int) {
	t.Helper()
	var all []response
	for _, s := range saved {
		all = append(all, s...)
	}
	return closeCheck0015(t, closed0015(t, finished0134, "completed"), all)
}

// closedParent0134 asserts issue n was commented once and closed as completed.
func closedParent0134(t *testing.T, rp replays, n int) {
	t.Helper()
	if !rp.called("gh", "-X PATCH", "state=closed", "state_reason=completed", issuePath0134(n)) {
		t.Errorf("want issue %d closed as completed, got gh calls %q", n, rp.calls("gh"))
	}
	if c := rp.count("gh", "-X POST", issuePath0134(n)+"/comments"); c != 1 {
		t.Errorf("want one comment on issue %d, got %d in gh calls %q", n, c, rp.calls("gh"))
	}
}

func Test0134_1(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", "> A parent.\n"), subIssues0134(30, 21),
	}, writes0134(30), noParent0134(30))
	sees(t, out, code, 0)
	closedParent0134(t, rp, 30)
}

func Test0134_2(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, mentionedBy0134(21, 30), []response{
		issue0134(t, 30, "open", "", list0134(20, 21)), subIssues0134(30),
		issue0134(t, 20, "closed", "completed", ""),
	}, writes0134(30), noParent0134(30))
	sees(t, out, code, 0)
	closedParent0134(t, rp, 30)
}

// Test0134_3 expects one comment on the parent that names the child once.
func Test0134_3(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", list0134(21)), subIssues0134(30, 21),
	}, writes0134(30), noParent0134(30))
	sees(t, out, code, 0)
	closedParent0134(t, rp, 30)
	if c := rp.count("gh", "-X POST", issuePath0134(30)+"/comments", repo0014+"#21 closed as completed"); c != 1 {
		t.Errorf("want the comment to name child %s#21 and its close reason once, got gh calls %q", repo0014, rp.calls("gh"))
	}
	if n := strings.Count(strings.Join(rp.calls("gh"), "\n"), repo0014+"#21 closed"); n != 1 {
		t.Errorf("want child 21 named once in the comment, got %d in gh calls %q", n, rp.calls("gh"))
	}
}

func Test0134_4(t *testing.T) {
	t.Parallel()
	rp, out, code := closeParents0134(t, parentOf0134(21, 30), []response{
		issue0134(t, 30, "open", "", "> A parent.\n"), subIssues0134(30, 21),
		issue0134(t, 40, "open", "", "> A grandparent.\n"), subIssues0134(40, 30),
	}, writes0134(30), parentOf0134(30, 40), writes0134(40), noParent0134(40))
	sees(t, out, code, 0)
	closedParent0134(t, rp, 30)
	closedParent0134(t, rp, 40)
}
