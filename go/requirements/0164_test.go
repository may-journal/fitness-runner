package requirements

import "testing"

func Test0164_1(t *testing.T) {
	t.Parallel()
	const org = "https://github.com/may-journal/"
	items := "### Phase 1\n\n- [x] Repo created, " + org + "example/issues/22 " + org + "example/pull/5\n" +
		"- [ ] Move the sync, " + org + ".github/issues/243 " + org + "example/pull/6 " + org + "example/issues/22#comment-9\n\n" +
		"### Phase 2\n\n- [ ] Install the checks, " + org + "other/issues/7"
	out, code, rp := judgeKind(t, epicBody(items), nil, kindGitHub("epic", "Epic"))
	sees(t, out, code, 0, "Passed.")
	if !commentedOn(rp, kindIssue, "body=✅ Epic looks good.") || !labeledOn(rp, kindIssue, "fitness-valid") || rp.called("gh", "sub_issues") {
		t.Errorf("want a passing Epic verdict and no sub-issue read, got gh calls %q", rp.calls("gh"))
	}
}
