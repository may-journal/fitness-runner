package requirements

import (
	"os"
	"strings"
	"testing"
)

// pulls0095 is the REST path of pull requests in the event's repo.
const pulls0095 = "repos/" + repo0014 + "/pulls"

// view0095 is GitHub showing PR n with its review decision, mergeable and
// a Fitness suite check that ended in conclusion. Auto-merge's own check
// is always still running, as it is while the job runs.
func view0095(n int, review, conclusion string) response {
	checks := `[{"__typename":"CheckRun","workflowName":"Fitness suite","status":"COMPLETED","conclusion":"` + conclusion + `"},` +
		`{"__typename":"CheckRun","workflowName":"Auto-merge on ready","status":"IN_PROGRESS","conclusion":""},` +
		`{"__typename":"StatusContext","context":"ci/legacy","state":"SUCCESS"}]`
	return response{
		Match:  []string{"pr view " + itoa(n) + " --repo"},
		Stdout: `{"isDraft":false,"reviewDecision":"` + review + `","mergeable":"MERGEABLE","statusCheckRollup":` + checks + `}` + "\n",
	}
}

// stackPRs0095 lists stack 7 as GitHub does: PR 1 already merged, PR 2
// on its branch, and PR 3 on top of PR 2 at head bbb.
const stackPRs0095 = `[{"number":1,"state":"closed","head":{"ref":"s0","sha":"zzz"},"base":{"ref":"main"}},` +
	`{"number":2,"state":"open","head":{"ref":"s1","sha":"aaa"},"base":{"ref":"s0"}},` +
	`{"number":3,"state":"open","head":{"ref":"s2","sha":"bbb"},"base":{"ref":"s1"}}]`

// stack0095 runs the auto-merge step on PR 2, the bottom open PR of native
// stack 7 whose top is PR 3 at head bbb. GitHub shows PRs 2 and 3 as views
// does, then answers the top PR's merge with merge and its poll with poll.
func stack0095(t *testing.T, views []response, merge, poll response) (replays, string, int) {
	t.Helper()
	merge.Match = []string{"-X PUT " + pulls0095 + "/3/merge-async"}
	poll.Match = []string{"api " + pulls0095 + "/3/merge-async/u1"}
	saved := append([]response{
		{Match: []string{"api " + pulls0095 + "/2"}, Stdout: `{"number":2,"stack":{"number":7,"position":1,"size":2}}` + "\n"},
		{Match: []string{"api repos/" + repo0014 + "/stacks/7"}, Stdout: `{"number":7,"pull_requests":` + stackPRs0095 + `}` + "\n"},
		poll, merge,
	}, views...)
	return run0091(t, "main", saved)
}

// approved0095 shows both PRs of the stack approved, green and mergeable.
var approved0095 = []response{view0095(2, "APPROVED", "SUCCESS"), view0095(3, "APPROVED", "SUCCESS")}

// pending0095 is GitHub accepting a merge request it runs as u1.
var pending0095 = response{Stdout: `{"status":"pending","details":{"message":"Merge request enqueued.","uuid":"u1","merge_method":"squash","bypass_rules":false}}` + "\n"}

// failed0095 is GitHub's answer for a stack merge its rules stopped.
const failed0095 = `{"status":"failed","details":{"message":"Required workflows 'Fitness suite' failed"}}` + "\n"

// notedAll0095 checks that PRs 2 and 3 both got note.
func notedAll0095(t *testing.T, rp replays, note string) {
	t.Helper()
	for _, n := range []int{2, 3} {
		if !rp.called("gh", "api "+issues0091+"/"+itoa(n)+"/comments", logged0091(note)) {
			t.Fatalf("PR %d of the stack did not get the note %q: %q", n, note, rp.calls("gh"))
		}
	}
}

// waits0095 checks that the stack merged nothing and both PRs got the note
// saying which one waits and why.
func waits0095(t *testing.T, rp replays, out string, code int, why string) {
	t.Helper()
	sees(t, out, code, 0, "::notice::Stack #7 waits: "+why+".")
	if rp.count("gh", "merge-async")+rp.count("gh", "pr merge") > 0 {
		t.Fatalf("a stack that is not ready asked GitHub to merge: %q", rp.calls("gh"))
	}
	notedAll0095(t, rp, "Auto-merge waits: stack #7 (#2 #3) merges into `main` once every PR in it is approved, green and mergeable. Waiting: "+why+".")
}

// refused0095 checks that a stack GitHub would not merge failed the job
// naming GitHub's answer, and that only its top PR got a note on why.
func refused0095(t *testing.T, rp replays, out string, code int, why, note string) {
	t.Helper()
	sees(t, out, code, 1, "::error::GitHub "+why+" stack #7 through #3", "Required workflows 'Fitness suite' failed")
	if !rp.called("gh", "api "+issues0091+"/3/comments", logged0091(note)) {
		t.Fatalf("the top PR of a stack GitHub would not merge got no note saying why: %q", rp.calls("gh"))
	}
	if postedOn0091(rp, 2) != "" {
		t.Fatalf("a lower PR of a stack GitHub would not merge got a note: %q", rp.calls("gh"))
	}
}

func Test0095_1(t *testing.T) {
	t.Parallel()
	merged := response{Stdout: `{"status":"merged","details":{"sha":"ccc"}}` + "\n"}
	rp, out, code := stack0095(t, approved0095, pending0095, merged)
	sees(t, out, code, 0, "::notice::GitHub merged stack #7 through #3.")
	if !rp.called("gh", "-X PUT "+pulls0095+"/3/merge-async", "merge_method=squash", "sha=bbb", "bypass_rules=false") {
		t.Fatalf("an approved stack did not ask GitHub to merge its top PR at its head without bypassing rules: %q", rp.calls("gh"))
	}
	if rp.count("gh", "pr merge")+rp.count("gh", pulls0095+"/2/merge-async") > 0 {
		t.Fatalf("an approved stack asked GitHub to merge a PR other than its top: %q", rp.calls("gh"))
	}
	notedAll0095(t, rp, "Auto-merge merged stack #7 (#2 #3) into `main` through #3.")
}

func Test0095_2(t *testing.T) {
	t.Parallel()
	views := []response{view0095(2, "APPROVED", "SUCCESS"), view0095(3, "REVIEW_REQUIRED", "SUCCESS")}
	rp, out, code := stack0095(t, views, response{}, response{})
	waits0095(t, rp, out, code, "#3 needs approval")
}

func Test0095_3(t *testing.T) {
	t.Parallel()
	views := []response{view0095(2, "APPROVED", "FAILURE"), view0095(3, "APPROVED", "SUCCESS")}
	rp, out, code := stack0095(t, views, response{}, response{})
	waits0095(t, rp, out, code, "#2 is not green")
}

func Test0095_4(t *testing.T) {
	t.Parallel()
	refusal := response{Stdout: failed0095, Stderr: "gh: Required workflows 'Fitness suite' failed (HTTP 400)\n", Exit: 1}
	rp, out, code := stack0095(t, approved0095, refusal, response{})
	refused0095(t, rp, out, code, "refused to merge",
		"Auto-merge could not merge stack #7 (#2 #3): GitHub refused, Required workflows 'Fitness suite' failed.")
	if rp.called("gh", pulls0095+"/3/merge-async/") {
		t.Fatalf("a refused merge was polled: %q", rp.calls("gh"))
	}
}

func Test0095_5(t *testing.T) {
	t.Parallel()
	rp, out, code := stack0095(t, approved0095, pending0095, response{Stdout: failed0095})
	refused0095(t, rp, out, code, "failed to merge",
		"Auto-merge could not merge stack #7 (#2 #3): GitHub failed, Required workflows 'Fitness suite' failed.")
}

// caller0095 is the caller that runs Auto-merge on ready and on reviews.
const caller0095 = "../../.github/workflows/auto-merge-on-ready.yml"

// callerIf0095 returns the caller's job-level condition, failing unless
// it runs on submitted reviews.
func callerIf0095(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(caller0095)
	mustDo(t, err)
	text := string(raw)
	_, cond, ok := strings.Cut(text, "    if: ")
	if !ok || !strings.Contains(text, "  pull_request_review:\n    types: [submitted]\n") {
		t.Fatalf("%s does not run on submitted reviews behind a job-level if", caller0095)
	}
	cond, _, _ = strings.Cut(cond, "\n")
	// holds0091 reads only event fields, so the repository is filled in here.
	cond = strings.ReplaceAll(cond, "github.repository", `"`+repo0014+`"`)
	return cond
}

// review0095 is a submitted review event in state on a branch of repo.
func review0095(state, repo string) map[string]string {
	return map[string]string{"event_name": "pull_request_review", "review.state": state, "pull_request.head.repo.full_name": repo}
}

func Test0095_6(t *testing.T) {
	t.Parallel()
	cond := callerIf0095(t)
	cases := []struct {
		name  string
		event map[string]string
		want  bool
	}{
		{"an approval", review0095("approved", repo0014), true},
		{"an approval of a fork's PR", review0095("approved", "someone/example"), false},
		{"a comment review", review0095("commented", repo0014), false},
		{"a review requesting changes", review0095("changes_requested", repo0014), false},
		{"a draft marked ready", map[string]string{"event_name": "pull_request_target", "review.state": "", "pull_request.head.repo.full_name": repo0014}, true},
	}
	for _, c := range cases {
		if holds0091(t, cond, c.event) != c.want {
			t.Errorf("%s: caller runs = %v, want %v", c.name, !c.want, c.want)
		}
	}
	approval := map[string]string{"action": "submitted", "pull_request.state": "open", "pull_request.draft": "false", "pull_request.auto_merge": ""}
	if !runs0091(t, approval) {
		t.Error("an approval of a stacked PR skips the Auto-merge job, want it run")
	}
}
