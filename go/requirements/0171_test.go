package requirements

import (
	"strings"
	"testing"
)

func Test0171_1(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"gh": {issue0014(t, 11, plan0014("- [x] Write the tests."), "Plan"), noChangelog0014}})
	pr := map[string]any{"number": 1, "title": "Say what the packages are for", "body": pr0014(11)}
	env := append(event0014(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr}), rp.env...)
	// An empty FITNESS_CHECKS_DIR hands down no checks, as on a contributor's machine.
	out, code := fitness(t, pinned0164(t, `{"*": "v1"}`), append(env, "FITNESS_CHECKS_DIR="), "pr-check")
	sees(t, out, code, 1, "type(scope): description")
	if strings.Contains(out, "no result from the check") {
		t.Errorf("pr-check judged the PR body without fetching its checks:\n%s", out)
	}
}
