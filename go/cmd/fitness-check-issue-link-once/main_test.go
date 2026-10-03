package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const repo = "may-journal/fitness-runner"

func TestLinks(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"bare", "fix(x): y\n\nPlan #12\n", []string{repo + "#12"}},
		{"qualified", "see may-journal/may-calendar#94", []string{"may-journal/may-calendar#94"}},
		{"url", "see https://github.com/May-Journal/may-photos/pull/111", []string{"may-journal/may-photos#111"}},
		{"url with fragment", "https://github.com/may-journal/fitness-runner/issues/153#issuecomment-1", []string{repo + "#153"}},
		{"same issue three ways", "#7 and may-journal/fitness-runner#7 and https://github.com/may-journal/fitness-runner/issues/7", []string{repo + "#7"}},
		{"distinct in first-seen order", "#3 then #9 then #3", []string{repo + "#3", repo + "#9"}},
		{"no links", "feat(x): color #fff and item 5", nil},
		{"entity is not a link", "it&#39;s fine", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := links(tc.text, repo); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("links = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStripComments(t *testing.T) {
	msg := "fix(x): y\n\nPlan #4\n# Please enter the commit message #9\n# ------------------------ >8 ------------------------\ndiff mentions #10\n"
	if got := links(stripComments(msg), repo); !reflect.DeepEqual(got, []string{repo + "#4"}) {
		t.Fatalf("links after strip = %v, want only #4", got)
	}
}

func TestJudge(t *testing.T) {
	first := earlier{sha: "abc1234def", subject: "feat(x): start", message: "feat(x): start\n\nPlan #12\n"}
	cases := []struct {
		name  string
		msg   string
		prior []earlier
		pr    *openPR
		ok    bool
		hint  string
	}{
		{"first link passes", "feat(x): start\n\nPlan #12\n", nil, nil, true, ""},
		{"repeat link fails", "fix(x): more\n\nPlan #12\n", []earlier{first}, nil, false, "commit abc1234"},
		{"repeat as url fails", "fix(x): more https://github.com/may-journal/fitness-runner/issues/12", []earlier{first}, nil, false, "commit abc1234"},
		{"PR-linked issue fails", "fix(x): more\n\nFixes #20\n", nil, &openPR{number: 30, body: "Closes #20"}, false, "open PR #30"},
		{"other issue passes", "fix(x): more\n\nPlan #13\n", []earlier{first}, &openPR{number: 30, body: "Closes #20"}, true, ""},
		{"no PR passes", "fix(x): more\n\nFixes #20\n", nil, nil, true, ""},
		{"no link passes", "fix(x): more", []earlier{first}, &openPR{number: 30, body: "Plan #12"}, true, ""},
		{"same number in another repo passes", "fix(x): see may-journal/may-calendar#12", []earlier{first}, nil, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.msg, repo, tc.prior, tc.pr)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if !tc.ok && !strings.Contains(res.Errors[0], tc.hint) {
				t.Fatalf("error %q should name %q", res.Errors[0], tc.hint)
			}
		})
	}
}

// TestBranchCommits covers the git side: only commits since the merge base
// count, and an amend leaves HEAD out.
func TestBranchCommits(t *testing.T) {
	remote, dir := t.TempDir(), t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "--bare", remote)
	git("init", "-q", "-b", "main")
	git("remote", "add", "origin", remote)
	git("commit", "-q", "--allow-empty", "-m", "feat(x): landed\n\nPlan #12")
	git("push", "-q", "origin", "main")
	git("switch", "-q", "-c", "topic")
	git("commit", "-q", "--allow-empty", "-m", "feat(x): start\n\nPlan #13")

	prior := branchCommits(dir)
	if len(prior) != 1 || prior[0].subject != "feat(x): start" {
		t.Fatalf("branchCommits = %+v, want only the topic commit", prior)
	}
	expectJudgeVerdict(t, "fix(x): again\n\nPlan #12", prior, true, "a link already on main should not count")
	expectJudgeVerdict(t, "fix(x): again\n\nPlan #13", prior, false, "a link already on the branch should fail")

	t.Setenv(amendEnv, "1")
	if prior := branchCommits(dir); len(prior) != 0 {
		t.Fatalf("amend should leave HEAD out, got %+v", prior)
	}
}

// expectJudgeVerdict judges msg against prior with no open PR and requires want.
func expectJudgeVerdict(t *testing.T, msg string, prior []earlier, want bool, why string) {
	t.Helper()
	if res := judge(msg, repo, prior, nil); res.Ok != want {
		t.Fatalf("%s: %v", why, res.Errors)
	}
}

func TestJudgeBranch(t *testing.T) {
	opened := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	before, after := opened.Add(-time.Hour), opened.Add(time.Hour)
	commit := func(sha, msg string, at time.Time) earlier {
		return earlier{sha: sha, subject: strings.SplitN(msg, "\n", 2)[0], message: msg, authored: at}
	}
	pr := &openPR{number: 30, body: "Closes #12", opened: opened}
	cases := []struct {
		name    string
		commits []earlier
		ok      bool
		hint    string
	}{
		{"pre-PR link passes", []earlier{commit("aaaaaaa1", "feat(x): start\n\nPlan #12", before)}, true, ""},
		{"branch repeat fails", []earlier{
			commit("aaaaaaa1", "feat(x): start\n\nPlan #12", before),
			commit("bbbbbbb2", "fix(x): more\n\nPlan #12", before),
		}, false, "commit bbbbbbb"},
		{"post-PR link fails", []earlier{commit("ccccccc3", "fix(x): more\n\nFixes #12", after)}, false, "open PR #30"},
		{"post-PR link to another issue passes", []earlier{commit("ccccccc3", "fix(x): more\n\nSee #13", after)}, true, ""},
		{"no links passes", []earlier{commit("ddddddd4", "fix(x): more", after)}, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judgeBranch(repo, tc.commits, pr)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if !tc.ok && (!strings.Contains(res.Errors[0], tc.hint) || !strings.Contains(res.Errors[0], "--force-with-lease")) {
				t.Fatalf("error %q should name %q and the fix", res.Errors[0], tc.hint)
			}
		})
	}
}

func TestParseEvent(t *testing.T) {
	payload := `{"pull_request":{"number":7,"body":"Closes #3","created_at":"2026-09-30T12:00:00Z","head":{"sha":"abc"}}}`
	pr, base, ok := parseEvent([]byte(payload), "origin/main")
	if !ok {
		t.Fatalf("parseEvent = %+v, %q, %v", pr, base, ok)
	}
	got := []any{pr.number, pr.body, pr.head, base, pr.opened.IsZero()}
	want := []any{7, "Closes #3", "abc", "origin/main", false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseEvent = %+v, %q, %v", pr, base, ok)
	}
	if _, _, ok := parseEvent([]byte(`{"ref":"refs/heads/main"}`), "origin/main"); ok {
		t.Fatal("a push payload has no pull request")
	}
}

// TestRunInert covers no event payload: a local suite run or a push to main.
func TestRunInert(t *testing.T) {
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_EVENT_PATH", "")
	os.Unsetenv("FITNESS_CTX_MESSAGE")
	res, err := run(t.TempDir(), nil)
	if err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("run = %+v, %v; want an inert pass", res, err)
	}
}

// TestPullRequestRun drives run end to end from an event payload, with main
// merged into the branch: main's commits and the merge never count.
func TestPullRequestRun(t *testing.T) {
	remote, dir := t.TempDir(), t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_AUTHOR_DATE=2026-09-30T10:00:00Z")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "--bare", remote)
	git("init", "-q", "-b", "main")
	git("remote", "add", "origin", remote)
	git("commit", "-q", "--allow-empty", "-m", "feat(x): landed\n\nPlan #12")
	git("push", "-q", "origin", "main")
	git("switch", "-q", "-c", "topic")
	git("commit", "-q", "--allow-empty", "-m", "feat(x): start\n\nPlan #12")
	git("switch", "-q", "main")
	git("commit", "-q", "--allow-empty", "-m", "feat(x): more on main\n\nPlan #12")
	git("push", "-q", "origin", "main")
	git("switch", "-q", "topic")
	git("merge", "-q", "--no-edit", "main")
	head := git("rev-parse", "HEAD")

	event := filepath.Join(t.TempDir(), "event.json")
	payload := `{"pull_request":{"number":9,"body":"Closes #12","created_at":"2026-09-30T12:00:00Z","head":{"sha":"` + head + `"}}}`
	if err := os.WriteFile(event, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_EVENT_PATH", event)
	os.Unsetenv("FITNESS_CTX_MESSAGE")
	res, err := run(dir, nil)
	if err != nil || !res.Ok {
		t.Fatalf("run = %+v, %v; the one pre-PR link should pass", res, err)
	}
	if res.FilesChecked != 2 {
		t.Fatalf("judged %d commits, want 2 (start + merge)", res.FilesChecked)
	}
}
