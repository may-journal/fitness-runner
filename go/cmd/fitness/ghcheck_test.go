package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeGH is an in-memory githubAPI.
type fakeGH struct {
	issues   map[int]fakeIssue
	prs      []target
	plans    []target
	comments map[int][]string
}

type fakeIssue struct {
	body   string
	labels []string
}

func (f *fakeGH) issue(n int) (string, []string, error) {
	i := f.issues[n]
	return i.body, i.labels, nil
}
func (f *fakeGH) openPRs() ([]target, error)   { return f.prs, nil }
func (f *fakeGH) openPlans() ([]target, error) { return f.plans, nil }
func (f *fakeGH) commentBodies(n int) ([]string, error) {
	return f.comments[n], nil
}
func (f *fakeGH) comment(n int, body string) error {
	if f.comments == nil {
		f.comments = map[int][]string{}
	}
	f.comments[n] = append(f.comments[n], body)
	return nil
}

// call records one fake check invocation.
type call struct {
	name  string
	args  []string
	stdin string
}

// fakeChecker passes every check except those in fail, answers
// pr-closes-issue --emit-closed from the body text, and records each call.
func fakeChecker(fail map[string]string, closes map[string]string, calls *[]call) bodyChecker {
	return bodyChecker{root: "/repo", exec: func(name string, args []string, stdin string) []byte {
		*calls = append(*calls, call{name, args, stdin})
		if name == "pr-closes-issue" && args[0] == "--emit-closed" {
			data, _ := os.ReadFile(args[2])
			if out, ok := closes[string(data)]; ok {
				return []byte(out)
			}
			return []byte("[]")
		}
		if msg, ok := fail[name]; ok {
			return []byte(`{"ok":false,"errors":["` + msg + `"],"filesChecked":1}`)
		}
		return []byte(`{"ok":true,"errors":[],"filesChecked":1}`)
	}}
}

func TestPRCheckPasses(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary.md"))
	var calls []call
	ev := ghEvent{name: "pull_request", PullRequest: &target{Number: 7, Title: "feat: x", Body: "body"}}
	if code := prCheck(ev, &fakeGH{}, fakeChecker(nil, nil, &calls)); code != 0 {
		t.Fatalf("prCheck = %d, want 0", code)
	}
	summary, _ := os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
	if !strings.Contains(string(summary), "✅ #7: PR description looks good.") {
		t.Errorf("summary = %q", summary)
	}
	if calls[0].name != "semantic-commit" || calls[0].args[1] != "feat: x" {
		t.Errorf("first call = %+v, want semantic-commit on the title", calls[0])
	}
	if calls[1].name != "pr-structure" || calls[1].stdin != "body" {
		t.Errorf("second call = %+v, want pr-structure on stdin", calls[1])
	}
}

func TestPRCheckFailsAndReports(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary.md"))
	var calls []call
	ev := ghEvent{name: "pull_request", PullRequest: &target{Number: 3, Title: "x", Body: "b"}}
	c := fakeChecker(map[string]string{"cspell": "unknown word"}, nil, &calls)
	if code := prCheck(ev, &fakeGH{}, c); code != 1 {
		t.Fatalf("prCheck = %d, want 1", code)
	}
	summary, _ := os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
	if !strings.Contains(string(summary), "❌ #3: PR description needs work:\n\n- unknown word") {
		t.Errorf("summary = %q", summary)
	}
}

func TestPRCheckRequiresAPlansClosures(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	var calls []call
	gh := &fakeGH{issues: map[int]fakeIssue{
		10: {body: "plan body", labels: []string{"Plan"}},
		11: {body: "not a plan", labels: []string{"bug"}},
	}}
	closes := map[string]string{"pr body": "[10,11]", "plan body": "[4,5]"}
	ev := ghEvent{name: "pull_request", PullRequest: &target{Number: 1, Title: "feat: x", Body: "pr body"}}
	prCheck(ev, gh, fakeChecker(nil, closes, &calls))
	for _, c := range calls {
		if c.name == "pr-closes-issue" && c.args[0] == "--body-file" {
			if got := c.args[len(c.args)-1]; got != "--require-close=4,5" {
				t.Fatalf("pr-closes-issue args end %q, want --require-close=4,5", got)
			}
			return
		}
	}
	t.Fatal("pr-closes-issue was not run in check mode")
}

func TestPRCheckDispatchChecksEveryOpenPR(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary.md"))
	var calls []call
	gh := &fakeGH{prs: []target{{Number: 1, Title: "feat: a"}, {Number: 2, Title: "fix: b"}}}
	prCheck(ghEvent{name: "workflow_dispatch"}, gh, fakeChecker(nil, nil, &calls))
	summary, _ := os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
	for _, want := range []string{"#1:", "#2:"} {
		if !strings.Contains(string(summary), want) {
			t.Errorf("summary missing %s: %q", want, summary)
		}
	}
}

func TestMissingResultFailsClosed(t *testing.T) {
	c := bodyChecker{exec: func(string, []string, string) []byte { return nil }}
	if got := c.errorsOf("cspell", nil, ""); len(got) != 1 || !strings.Contains(got[0], "cspell") {
		t.Fatalf("errorsOf = %v, want one error naming cspell", got)
	}
}

func TestPlanCheckCommentsOncePerBody(t *testing.T) {
	var calls []call
	gh := &fakeGH{}
	ev := ghEvent{name: "issues", Issue: &target{Number: 5, Body: "abc"}}
	c := fakeChecker(nil, nil, &calls)
	if code := planCheck(ev, gh, c); code != 0 {
		t.Fatalf("planCheck = %d, want 0", code)
	}
	planCheck(ev, gh, c)
	if len(gh.comments[5]) != 1 {
		t.Fatalf("comments = %d, want 1 (the re-run stays quiet)", len(gh.comments[5]))
	}
	want := "✅ Plan looks good.\n\n<!-- fitness:plan-structure:ba7816bf8f01 -->"
	if gh.comments[5][0] != want {
		t.Errorf("comment = %q, want %q", gh.comments[5][0], want)
	}
}

func TestPlanCheckFailureComment(t *testing.T) {
	var calls []call
	gh := &fakeGH{}
	ev := ghEvent{name: "issues", Issue: &target{Number: 2, Body: "abc"}}
	c := fakeChecker(map[string]string{"plan-structure": "missing Background"}, nil, &calls)
	if code := planCheck(ev, gh, c); code != 1 {
		t.Fatalf("planCheck = %d, want 1", code)
	}
	want := "❌ Plan needs work:\n- missing Background\n\n<!-- fitness:plan-structure:ba7816bf8f01 -->"
	if gh.comments[2][0] != want {
		t.Errorf("comment = %q, want %q", gh.comments[2][0], want)
	}
}

func TestPlanCheckDispatchChecksOpenPlans(t *testing.T) {
	var calls []call
	gh := &fakeGH{plans: []target{{Number: 1, Body: "a"}, {Number: 2, Body: "b"}}}
	planCheck(ghEvent{name: "workflow_dispatch"}, gh, fakeChecker(nil, nil, &calls))
	if len(gh.comments[1]) != 1 || len(gh.comments[2]) != 1 {
		t.Fatalf("comments = %v, want one on each open Plan", gh.comments)
	}
}

func TestDecodeLines(t *testing.T) {
	got, err := decodeLines[target]([]byte("{\"number\":1,\"body\":\"a\"}\n{\"number\":2,\"body\":\"b\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []target{{Number: 1, Body: "a"}, {Number: 2, Body: "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeLines = %+v, want %+v", got, want)
	}
}
