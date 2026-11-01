package requirements

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Issue texts every 0087 test files: an older issue, a new one that repeats
// it in other words, and one about something else.
const (
	offlineTitle = "Sync photos taken offline"
	offlineBody  = "> Photos taken without signal upload once the phone reconnects.\n\n## Background\n\nPhotos taken without signal stay on the phone, and their captions and album order are lost.\n\n## What needs to happen\n\n- [ ] Queue photos taken offline with their captions and album order.\n- [ ] Upload the queue once the phone reconnects.\n"
	repeatTitle  = "Upload photos taken offline after reconnecting"
	repeatBody   = "> Photos taken without signal upload once the phone reconnects.\n\n## Background\n\nPhotos taken without signal stay on the phone, and their captions and album order are lost.\n\n## What needs to happen\n\n- [ ] Queue offline photos with their captions and album order.\n- [ ] Upload the queue once the phone reconnects.\n"
	darkTitle    = "Dark theme for the settings screen"
	darkBody     = "> The settings screen follows the system dark theme.\n\n## Background\n\nSettings stay bright at night.\n\n## What needs to happen\n\n- [ ] Draw settings in dark colors when the system theme is dark.\n"
)

// listed0087 is one issue as the issue list prints it through fitness's
// filter: open unless closedAt is set, under parent unless 0.
func listed0087(n int, title, body string, parent int, closedAt string) string {
	state := "open"
	if closedAt != "" {
		state = "closed"
	}
	line, _ := json.Marshal(map[string]any{
		"id": 1000 + n, "number": n, "title": title, "body": body, "state": state,
		"state_reason": "", "closed_at": closedAt, "parentNumber": parent, "opener": "dev",
	})
	return string(line) + "\n"
}

// dupeGitHub is the saved GitHub that lists issues and holds note, the bot's
// earlier note on issue n as the lookup prints it. Every write succeeds.
func dupeGitHub(n int, note string, issues ...string) []response {
	var open, closed string
	for _, line := range issues {
		if strings.Contains(line, `"state":"closed"`) {
			closed += line
		} else {
			open += line
		}
	}
	return []response{
		{Match: []string{"graphql", "states: [OPEN]"}, Stdout: open},
		{Match: []string{"graphql", "states: [CLOSED]", "since="}, Stdout: closed},
		{Match: []string{"issues/" + itoa(n) + "/comments?per_page=100", "fitness:dupe-check:"}, Stdout: note},
		{Match: []string{"-X POST", "issues/" + itoa(n) + "/comments"}, Stdout: `{"id":90}`},
		{Match: []string{"-X PATCH"}, Stdout: `{}`},
	}
}

// dupeEvent is the `issues` event for action on issue n, by dev.
func dupeEvent(action string, n int, title, body string) map[string]any {
	issue := map[string]any{"id": 1000 + n, "number": n, "title": title, "body": body, "state": "open"}
	return map[string]any{"action": action, "issue": issue, "sender": map[string]any{"login": "dev"}}
}

// wrote reports whether dupe-check wrote anything to GitHub.
func wrote(rp replays) bool {
	return rp.called("gh", "-X POST") || rp.called("gh", "-X PATCH")
}

func Test0087_1(t *testing.T) {
	t.Parallel()
	issues := []string{listed0087(12, offlineTitle, offlineBody, 0, ""), listed0087(21, repeatTitle, repeatBody, 0, ""), listed0087(22, darkTitle, darkBody, 0, "")}
	out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("opened", 21, repeatTitle, repeatBody), nil, dupeGitHub(21, "", issues...))
	sees(t, out, code, 0, `\#21 may repeat \#12`)
	if rp.count("gh", "-X POST", "issues/21/comments", "#12 (score") != 1 || rp.called("gh", "-X PATCH") {
		t.Errorf("want one note on #21 naming #12, got gh calls %q", rp.calls("gh"))
	}
	out, code, rp = ghRun(t, "dupe-check", "issues", dupeEvent("opened", 22, darkTitle, darkBody), nil, dupeGitHub(22, "", issues...))
	sees(t, out, code, 0, `\#22 resembles no earlier issue.`)
	if wrote(rp) {
		t.Errorf("want nothing written on #22, got gh calls %q", rp.calls("gh"))
	}
}

func Test0087_2(t *testing.T) {
	t.Parallel()
	note := `{"id":77,"body":"🔁 This issue may repeat an earlier one.\n\n- #12 (score 0.50)\n\n<!-- fitness:dupe-check:000000000000 -->"}` + "\n"
	issues := []string{listed0087(12, offlineTitle, offlineBody, 0, ""), listed0087(21, repeatTitle, repeatBody, 0, "")}
	out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("edited", 21, repeatTitle, repeatBody), nil, dupeGitHub(21, note, issues...))
	sees(t, out, code, 0, `\#21 may repeat \#12`)
	if !rp.called("gh", "-X PATCH", "issues/comments/77", "#12 (score") || rp.called("gh", "-X POST") {
		t.Errorf("want note 77 edited in place and no second note, got gh calls %q", rp.calls("gh"))
	}
}

func Test0087_3(t *testing.T) {
	t.Parallel()
	named := repeatBody + "\nFollows #9.\n"
	issues := []string{
		listed0087(9, offlineTitle, offlineBody, 0, ""), listed0087(12, offlineTitle, offlineBody, 0, ""),
		listed0087(20, repeatTitle, repeatBody, 12, ""), listed0087(21, repeatTitle, named, 12, ""),
	}
	out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("opened", 21, repeatTitle, named), nil, dupeGitHub(21, "", issues...))
	sees(t, out, code, 0, `\#21 resembles no earlier issue.`)
	if wrote(rp) {
		t.Errorf("want nothing written on #21, got gh calls %q", rp.calls("gh"))
	}
}

func Test0087_4(t *testing.T) {
	t.Parallel()
	config := map[string]string{".fitnessrc.json": `{"dupeCheck": {"closeScore": 0.75}}`}
	issues := []string{listed0087(12, offlineTitle, offlineBody, 0, ""), listed0087(21, offlineTitle, offlineBody, 0, "")}
	out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("opened", 21, offlineTitle, offlineBody), config, dupeGitHub(21, "", issues...))
	sees(t, out, code, 0, `\#21 may repeat \#12 (score 1.00). Closed it as a duplicate of \#12.`)
	if !rp.called("gh", "-X POST", "issues/21/comments", "Closed as a duplicate of #12") {
		t.Errorf("want a note naming #12, got gh calls %q", rp.calls("gh"))
	}
	if !rp.called("gh", "-X PATCH", "state=closed", "state_reason=duplicate", "duplicate_issue_id=1012", "issues/21") {
		t.Errorf("want #21 closed as a duplicate of #12, got gh calls %q", rp.calls("gh"))
	}
	closed := map[string]any{"number": 21, "body": offlineBody, "state_reason": "duplicate", "closed_at": time.Now().UTC().Format(time.RFC3339)}
	out, code, rp = ghRun(t, "close-check", "issues", map[string]any{"action": "closed", "issue": closed, "sender": map[string]any{"login": "github-actions[bot]"}}, nil, nil)
	sees(t, out, code, 0, "#21 closed as duplicate")
	if wrote(rp) {
		t.Errorf("want close-check to leave #21 closed, got gh calls %q", rp.calls("gh"))
	}
}

func Test0087_5(t *testing.T) {
	t.Parallel()
	old := time.Now().UTC().AddDate(0, 0, -200).Format(time.RFC3339)
	issues := []string{
		listed0087(5, offlineTitle, offlineBody, 0, old), listed0087(12, offlineTitle, offlineBody, 0, ""),
		listed0087(21, repeatTitle, repeatBody, 0, ""), listed0087(22, darkTitle, darkBody, 0, ""),
	}
	out, code, rp := ghRun(t, "dupe-check", "workflow_dispatch", map[string]any{}, nil, dupeGitHub(21, "", issues...))
	sees(t, out, code, 0, "3 issues;", "| #21 | #12 |", "| warn |")
	if strings.Contains(out, "#5 |") || strings.Contains(out, "| #22 |") {
		t.Errorf("want only the pair in the window scoring near the warn score, got:\n%s", out)
	}
	if wrote(rp) {
		t.Errorf("want nothing written, got gh calls %q", rp.calls("gh"))
	}
}
