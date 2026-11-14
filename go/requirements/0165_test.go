package requirements

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// completed0165 is what `fitness dependabot-pr` wrote to GitHub for a PR:
// its title, description and the changelog entry it committed.
type completed0165 struct {
	title, body, file, entry string
	issueMade                bool
}

// payload0165 is a recorded Dependabot pull_request_target payload, with its
// author replaced by login.
func payload0165(t *testing.T, login string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("testdata/dependabotPR.json")
	mustDo(t, err)
	var ev map[string]any
	mustDo(t, json.Unmarshal(raw, &ev))
	ev["pull_request"].(map[string]any)["user"] = map[string]any{"login": login}
	return ev
}

// complete0165 runs dependabot-pr on the payload, GitHub listing openIssue as the open issues.
func complete0165(t *testing.T, payload map[string]any, openIssue string) (completed0165, replays) {
	t.Helper()
	listed := response{Match: []string{"issues?state=open"}, Stdout: openIssue}
	saved := []response{
		listed,
		{Match: []string{"-X POST", "--jq .number", "issues"}, Stdout: "31\n"},
		{Match: []string{"pulls/7/files"}},
		{Match: []string{"-X PUT", "contents/"}, Stdout: "{}\n"},
		{Match: []string{"-X PATCH", "pulls/7"}, Stdout: "{}\n"},
	}
	rp := standIn(t, map[string][]response{"gh": saved})
	env := event0014(t, "pull_request_target", payload)
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "dependabot-pr")
	sees(t, out, code, 0)
	return readWrites0165(t, rp.calls("gh")), rp
}

// readWrites0165 reads what the saved gh calls wrote to GitHub.
func readWrites0165(t *testing.T, calls []string) completed0165 {
	t.Helper()
	var got completed0165
	for _, call := range calls {
		call = strings.ReplaceAll(call, `\n`, "\n")
		got.issueMade = got.issueMade || strings.Contains(call, "--jq .number")
		if strings.Contains(call, "-X PUT") {
			got.file, got.entry = entryWrite0165(t, call)
		}
		if strings.Contains(call, "-X PATCH") {
			got.title = between0165(call, "-f title=", " -f body=")
			got.body = between0165(call, "-f body=", " repos/"+repo0014+"/pulls/7")
		}
	}
	return got
}

// entryWrite0165 is the path and text a contents write committed.
func entryWrite0165(t *testing.T, call string) (string, string) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(between0165(call, "-f content=", " -f branch="))
	mustDo(t, err)
	return between0165(call, "contents/", ""), string(raw)
}

// between0165 is the text of s after from and before to; "" for to reads to the end.
func between0165(s, from, to string) string {
	_, rest, _ := strings.Cut(s, from)
	if to == "" {
		return strings.TrimSpace(rest)
	}
	head, _, _ := strings.Cut(rest, to)
	return head
}

// judge0165 runs pr-check on the completed PR, GitHub holding the bump issue
// and the PR's changelog entry file.
func judge0165(t *testing.T, got completed0165) (string, int) {
	t.Helper()
	patch := "@@ -0,0 +1 @@\n+" + strings.ReplaceAll(strings.TrimSpace(got.entry), "\n", "\n+")
	file, err := json.Marshal(map[string]string{"filename": got.file, "status": "added", "patch": patch})
	mustDo(t, err)
	rp := standIn(t, map[string][]response{"gh": {
		{Match: []string{"pulls/7/files", "startswith"}, Stdout: string(file) + "\n"},
		{Match: []string{"pulls/7/files"}},
		issue0014(t, 31, "Dependabot proposed a bump.\n"),
	}})
	pr := map[string]any{"number": 7, "title": got.title, "body": got.body}
	env := event0014(t, "pull_request", map[string]any{"action": "opened", "pull_request": pr})
	return fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
}

func Test0165_1(t *testing.T) {
	t.Parallel()
	got, _ := complete0165(t, payload0165(t, "dependabot[bot]"), "")
	if !got.issueMade || !strings.HasPrefix(got.title, "build(deps): bump github.com/spf13/cobra") {
		t.Fatalf("want a new issue and a semantic title, got %+v", got)
	}
	wants := map[string]string{"closes the issue": "Closes #31", "keeps the notes": "Fix a crash when a flag is missing"}
	for what, want := range wants {
		if !strings.Contains(got.body, want) {
			t.Errorf("want the description to hold %q (%s):\n%s", want, what, got.body)
		}
	}
	out, code := judge0165(t, got)
	sees(t, out, code, 0, "pr-check #7", "Passed.")
}

func Test0165_2(t *testing.T) {
	t.Parallel()
	got, _ := complete0165(t, payload0165(t, "dependabot[bot]"), `{"number": 31, "title": "Bump github.com/spf13/cobra"}`+"\n")
	if got.issueMade || !strings.Contains(got.body, "Closes #31") {
		t.Errorf("want the open bump issue reused, got %+v", got)
	}
}

func Test0165_3(t *testing.T) {
	t.Parallel()
	_, rp := complete0165(t, payload0165(t, "jason"), "")
	if rp.called("gh", "-X") {
		t.Errorf("want a person's PR left alone, got gh calls %q", rp.calls("gh"))
	}
}

func Test0165_4(t *testing.T) {
	t.Parallel()
	got, _ := complete0165(t, payload0165(t, "dependabot[bot]"), "")
	pr := payload0165(t, "dependabot[bot]")["pull_request"].(map[string]any)
	pr["title"], pr["body"] = got.title, got.body
	entry, err := json.Marshal(map[string]string{"filename": got.file, "status": "added", "patch": ""})
	mustDo(t, err)
	rp := standIn(t, map[string][]response{"gh": {{Match: []string{"pulls/7/files"}, Stdout: string(entry) + "\n"}}})
	env := event0014(t, "pull_request_target", map[string]any{"action": "synchronize", "pull_request": pr})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "dependabot-pr")
	sees(t, out, code, 0)
	if rp.called("gh", "-X") {
		t.Errorf("want a completed PR left alone, got gh calls %q", rp.calls("gh"))
	}
}
