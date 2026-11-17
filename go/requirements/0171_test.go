package requirements

import (
	"strings"
	"testing"
	"time"
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

func Test0171_2(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	r.archives["semantic-commit"] = archive0173(t, "semantic-commit")
	r.publish(t, map[string]string{"semantic-commit": "1.0.0", "md-files": "1.0.0", "go-files": "1.0.0", "swift-files": "1.0.0"}, time.Now())
	repo := pinned0164(t, `{"semantic-commit": "v1", "md-files": "v1", "go-files": "v1", "swift-files": "v1"}`)
	rp := standIn(t, map[string][]response{"gh": {noChangelog0014}})
	pr := map[string]any{"number": 1, "title": "Say what the packages are for", "body": pr0014(11)}
	env := append(event0014(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr}), rp.env...)
	out, code := run0162(t, bundle0162(t), repo, append(env, "FITNESS_CACHE_DIR="+r.cache), "pr-check")
	sees(t, out, code, 1, "semantic-commit ran")
	fetchedOnly(t, r, out, "semantic-commit")
}
