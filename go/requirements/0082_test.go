package requirements

import (
	"encoding/json"
	"strings"
	"testing"
)

// kindIssue is the issue number every 0082 event and saved response names.
const kindIssue = "21"

// judgeKind runs issue-check on issue 21 with body, as an issue edit
// triggers it, in happyRepo with files written over it.
func judgeKind(t *testing.T, body string, files map[string]string, saved []response) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 21, "title": "t", "body": body}
	return ghRun(t, "issue-check", "issues", map[string]any{"action": "edited", "issue": issue}, files, saved)
}

// kindGitHub is the saved GitHub for issue 21 of kind, with no verdict yet,
// carrying labels.
func kindGitHub(kind string, labels ...string) []response {
	return issueGitHub(kindIssue, "fitness:issue-check:"+kind+":", "", labels...)
}

// epicGitHub is kindGitHub for an Epic whose sub-issues are subs, in this
// repository.
func epicGitHub(subs ...string) []response {
	var lines strings.Builder
	for _, n := range subs {
		line, _ := json.Marshal(map[string]any{"number": json.Number(n), "repo": "may-journal/example"})
		lines.Write(append(line, '\n'))
	}
	return append([]response{{Match: []string{"issues/" + kindIssue + "/sub_issues"}, Stdout: lines.String()}}, kindGitHub("epic", "Epic")...)
}

// epicBody is an Epic that follows the template, listing items.
func epicBody(items string) string {
	return "> Every fitness run finishes in a fraction of the time.\n\n## Background\n\nThis epic is a test fixture.\n\n## What needs to happen\n\n" + items + "\n"
}

func Test0082_1(t *testing.T) {
	t.Parallel()
	out, code, rp := judgeKind(t, epicBody("1. #22\n2. #23, after #22"), nil, epicGitHub("22", "23"))
	sees(t, out, code, 0, "Passed.")
	if !commentedOn(rp, kindIssue, "body=✅ Epic looks good.") || !labeledOn(rp, kindIssue, "fitness-valid") {
		t.Errorf("want a passing Epic verdict and fitness-valid, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_2(t *testing.T) {
	t.Parallel()
	table := "\n\n| Plan | State |\n| --- | --- |\n| 22 | open |"
	items := "1. [ ] #22\n2. #23 ✅\n3. #24, after #25\n4. #25\n5. may-journal/example#25\n6. may-journal/example#26, after #27\n7. #27"
	body := strings.Replace(epicBody(items), "fixture.", "fixture."+table, 1)
	files := map[string]string{".fitnessrc.json": `{"proseBudget": {"maxListItems": 4}}` + "\n"}
	out, code, rp := judgeKind(t, body, files, epicGitHub("22", "23", "24", "25", "26", "27"))
	sees(t, out, code, 1, "item 1 uses a checkbox", "item 2 has an emoji", "item 3 (\\#24) is listed before \\#25",
		"example\\#25 is listed more than once", "item 6 (may-journal/example\\#26) is listed before \\#27",
		"tables are not allowed", "a list has more than 4 items at level 1")
	if strings.Contains(out, "is not listed under") || strings.Count(out, "more than 4 items") != 1 {
		t.Errorf("want the checkbox item counted as listed and one list-length finding:\n%s", out)
	}
	if !commentedOn(rp, kindIssue, `body=❌ Epic needs work:\n- `) || !labeledOn(rp, kindIssue, "fitness-invalid") {
		t.Errorf("want a failing Epic verdict and fitness-invalid, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_3(t *testing.T) {
	t.Parallel()
	saved := append([]response{
		{Match: []string{"labels: [.labels[].name]", "issues/24"}, Stdout: `{"body": "", "labels": []}` + "\n"},
		{Match: []string{"issues/26"}, Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1},
	}, epicGitHub("22", "23")...)
	out, code, _ := judgeKind(t, epicBody("1. #22\n2. #24\n3. #26"), nil, saved)
	sees(t, out, code, 1, "24 is listed but is not a sub-issue of this epic", "26 is listed but cannot be read",
		"sub-issue \\#23 is not listed under")
}

func Test0082_4(t *testing.T) {
	t.Parallel()
	template := "---\nname: Research\nlabels: Research\n---\n\n<!-- Pitch. -->\n> REPLACE-ME\n\n## Background\n<!-- Context. -->\n\n## What to find out\n<!-- Questions. -->\n- REPLACE-ME\n"
	files := map[string]string{".github/ISSUE_TEMPLATE/research.md": template}
	pitch := "> We know which sync wins before we plan the fix.\n\n## Background\n\nTwo devices edit one note offline.\n"
	out, code, _ := judgeKind(t, pitch, files, kindGitHub("research", "Research"))
	sees(t, out, code, 1, "What to find out")
	out, code, rp := judgeKind(t, pitch+"\n## What to find out\n\n- Which edit wins.\n", files, kindGitHub("research", "Research"))
	sees(t, out, code, 0, "Passed.")
	if !commentedOn(rp, kindIssue, "body=✅ Research looks good.") {
		t.Errorf("want a passing Research verdict, got gh calls %q", rp.calls("gh"))
	}

	plan := "---\nname: Plan\nlabels: Plan\n---\n\n<!-- Pitch. -->\n> REPLACE-ME\n\n## Background\n<!-- Context. -->\n\n## Acceptance\n<!-- Checks. -->\n\n## What needs to happen\n<!-- Steps. -->\n- [ ] REPLACE-ME\n"
	files = map[string]string{".github/ISSUE_TEMPLATE/plan.md": plan}
	out, code, _ = judgeKind(t, planBody("- [ ] Write the tests."), files, kindGitHub("plan", "Plan"))
	sees(t, out, code, 1, "Acceptance")
	if strings.Contains(out, "Unexpected") {
		t.Errorf("want only the template's sections asked for:\n%s", out)
	}
	accepted := strings.Replace(planBody("- [ ] Write the tests."), "## What", "## Acceptance\n\nThe tests pass.\n\n## What", 1)
	out, code, _ = judgeKind(t, accepted, files, kindGitHub("plan", "Plan"))
	sees(t, out, code, 0, "Passed.")
}

func Test0082_5(t *testing.T) {
	t.Parallel()
	out, code, rp := judgeKind(t, "Something odd happened.", nil, kindGitHub("plan", "question"))
	sees(t, out, code, 0, "has no issue kind label, so it is not checked.")
	if rp.called("gh", "-X POST") {
		t.Errorf("want nothing written to an issue with no kind, got gh calls %q", rp.calls("gh"))
	}

	issue := map[string]any{"number": 21, "title": "t", "body": epicBody("1. #22"), "labels": []map[string]string{{"name": "Epic"}}}
	gone := response{Match: []string{"fitness:issue-check:epic:"}, Stderr: "gh: Could not resolve to an Issue with the number of 21.\n", Exit: 1}
	out, code, rp = ghRun(t, "issue-check", "issues", map[string]any{"action": "edited", "issue": issue}, nil, append([]response{gone}, epicGitHub("22")...))
	sees(t, out, code, 0, "21 no longer exists")
	if rp.called("gh", "-X POST") || rp.called("gh", "labels: [.labels[].name]") {
		t.Errorf("want the event's labels used and nothing written to a moved issue, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_6(t *testing.T) {
	t.Parallel()
	out, code, rp := judgeKind(t, planBody("- [ ] Write the tests."), nil, kindGitHub("plan", "Plan", "Epic"))
	sees(t, out, code, 1, "the issue has 2 kind labels (Plan, Epic); keep the one label that names its kind")
	if !labeledOn(rp, kindIssue, "fitness-invalid") {
		t.Errorf("want fitness-invalid, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_7(t *testing.T) {
	t.Parallel()
	body := planBody("- [ ] Write the tests.")
	issue := map[string]any{"number": 7, "title": "t", "body": body}
	saved := planGitHub(planVerdict("✅ Plan looks good.", body), "fitness", "fitness-valid")
	out, code, rp := ghRun(t, "issue-check", "issues", map[string]any{"action": "edited", "issue": issue}, nil, saved)
	sees(t, out, code, 0, "Passed.")
	if rp.called("gh", "body=") {
		t.Errorf("want no comment written, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_8(t *testing.T) {
	t.Parallel()
	plan, _ := json.Marshal(map[string]any{"number": 7, "body": planBody("- [ ] Write the tests."), "labelNames": []string{"Plan"}})
	epic, _ := json.Marshal(map[string]any{"number": 21, "body": epicBody("1. #22"), "labelNames": []string{"Epic"}})
	saved := append([]response{
		{Match: []string{"issues?labels=Plan&state=open"}, Stdout: string(plan) + "\n"},
		{Match: []string{"issues?labels=Epic&state=open"}, Stdout: string(epic) + "\n"},
	}, epicGitHub("22")...)
	saved = append(saved, issueGitHub(planIssue, "fitness:issue-check:plan:", "", "Plan")...)
	out, code, rp := ghRun(t, "issue-check", "workflow_dispatch", map[string]any{}, nil, saved)
	sees(t, out, code, 0, "issue-check #7", "issue-check #21")
	if !labeledOn(rp, planIssue, "fitness-valid") || !labeledOn(rp, kindIssue, "fitness-valid") {
		t.Errorf("want both open kind issues judged, got gh calls %q", rp.calls("gh"))
	}
}
