package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeGH is an in-memory githubAPI.
type fakeGH struct {
	issues   map[int]fakeIssue
	prs      []target
	plans    []target
	comments map[int][]verdict
	labels   map[string]bool
	nextID   int64
	closers  map[int]closer
	reopened []int
}

type fakeIssue struct {
	body   string
	labels []string
}

func (f *fakeGH) issue(n int) (string, []string, error) {
	i := f.issues[n]
	return i.body, i.labels, nil
}
func (f *fakeGH) verdicts(n int) ([]verdict, error) {
	return append([]verdict(nil), f.comments[n]...), nil
}
func (f *fakeGH) commentBodies(n int) ([]string, error) {
	var bodies []string
	for _, v := range f.comments[n] {
		bodies = append(bodies, v.Body)
	}
	return bodies, nil
}
func (f *fakeGH) openPRs() ([]target, error)   { return f.prs, nil }
func (f *fakeGH) openPlans() ([]target, error) { return f.plans, nil }
func (f *fakeGH) comment(n int, body string) error {
	if f.comments == nil {
		f.comments = map[int][]verdict{}
	}
	f.nextID++
	f.comments[n] = append(f.comments[n], verdict{ID: f.nextID, NodeID: "node" + strconv.FormatInt(f.nextID, 10), Body: body})
	return nil
}
func (f *fakeGH) editComment(id int64, body string) error {
	return f.eachComment(func(v *verdict) {
		if v.ID == id {
			v.Body = body
		}
	})
}
func (f *fakeGH) hideComment(nodeID string) error {
	return f.eachComment(func(v *verdict) {
		if v.NodeID == nodeID {
			v.IsHidden = true
		}
	})
}
func (f *fakeGH) eachComment(fn func(*verdict)) error {
	for _, vs := range f.comments {
		for i := range vs {
			fn(&vs[i])
		}
	}
	return nil
}
func (f *fakeGH) ensureLabel(name, _ string) error {
	if f.labels == nil {
		f.labels = map[string]bool{}
	}
	f.labels[name] = true
	return nil
}
func (f *fakeGH) addLabels(n int, names []string) error {
	i := f.issues[n]
	i.labels = append(i.labels, names...)
	f.setIssue(n, i)
	return nil
}
func (f *fakeGH) removeLabel(n int, name string) error {
	i := f.issues[n]
	var kept []string
	for _, l := range i.labels {
		if l != name {
			kept = append(kept, l)
		}
	}
	i.labels = kept
	f.setIssue(n, i)
	return nil
}
func (f *fakeGH) setIssue(n int, i fakeIssue) {
	if f.issues == nil {
		f.issues = map[int]fakeIssue{}
	}
	f.issues[n] = i
}

func (f *fakeGH) reopen(n int) error {
	f.reopened = append(f.reopened, n)
	return nil
}
func (f *fakeGH) closingPR(n int) (closer, error) { return f.closers[n], nil }

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
	t.Setenv("GITHUB_ACTIONS", "true")
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
	assertTitleThenBodyCalls(t, calls)
}

// assertTitleThenBodyCalls checks the PR title is checked first, then the body.
func assertTitleThenBodyCalls(t *testing.T, calls []call) {
	t.Helper()
	if calls[0].name != "semantic-commit" || calls[0].args[1] != "feat: x" {
		t.Errorf("first call = %+v, want semantic-commit on the title", calls[0])
	}
	if calls[1].name != "pr-structure" || calls[1].stdin != "body" {
		t.Errorf("second call = %+v, want pr-structure on stdin", calls[1])
	}
}

// assertContainsAll reports each wanted piece that text, named what, lacks.
func assertContainsAll(t *testing.T, what, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s missing %q: %q", what, want, text)
		}
	}
}

// assertContainsNone reports each unwanted piece that text, named what, holds.
func assertContainsNone(t *testing.T, what, text string, unwanted ...string) {
	t.Helper()
	for _, piece := range unwanted {
		if strings.Contains(text, piece) {
			t.Errorf("%s names %q: %q", what, piece, text)
		}
	}
}

func TestPRCheckFailsAndReports(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
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
	t.Setenv("GITHUB_ACTIONS", "true")
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

func TestPRCheckFailsOnAClosedIssuesUncheckedItems(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary.md"))
	var calls []call
	gh := &fakeGH{issues: map[int]fakeIssue{
		20: {body: "- [x] done\n- [ ] write docs\n```\n- [ ] fenced example\n```\n- [ ] ship it"},
		21: {body: "- [x] all done"},
	}}
	closes := map[string]string{"pr body": "[20,21]"}
	ev := ghEvent{name: "pull_request", PullRequest: &target{Number: 9, Title: "feat: x", Body: "pr body"}}
	if code := prCheck(ev, gh, fakeChecker(nil, closes, &calls)); code != 1 {
		t.Fatalf("prCheck = %d, want 1", code)
	}
	summary, _ := os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
	assertContainsAll(t, "summary", string(summary),
		`#20, which has an unchecked item: "write docs"`, `#20, which has an unchecked item: "ship it"`)
	assertContainsNone(t, "summary", string(summary), "fenced example", "#21")
}

func TestPRCheckPassesWhenClosedIssuesAreTicked(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	var calls []call
	gh := &fakeGH{issues: map[int]fakeIssue{20: {body: "- [x] done\n```\n- [ ] fenced example\n```"}}}
	ev := ghEvent{name: "pull_request", PullRequest: &target{Number: 9, Title: "feat: x", Body: "pr body"}}
	if code := prCheck(ev, gh, fakeChecker(nil, map[string]string{"pr body": "[20]"}, &calls)); code != 0 {
		t.Fatalf("prCheck = %d, want 0", code)
	}
}

func TestPRCheckDispatchChecksEveryOpenPR(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
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

// pinAttribution fixes the build and run the verdict footer names, and
// returns that footer.
func pinAttribution(t *testing.T) string {
	t.Helper()
	buildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef1234567890"}}}, true
	}
	t.Cleanup(func() { buildInfo = debug.ReadBuildInfo })
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "o/r")
	t.Setenv("GITHUB_RUN_ID", "42")
	return "<sub>Checked by fitness-runner [abcdef1](https://github.com/may-journal/fitness-runner/commit/abcdef1234567890) · [workflow run](https://github.com/o/r/actions/runs/42)</sub>"
}

func TestAttributionFromAnInstalledVersion(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "")
	cases := map[string]string{
		"v0.0.0-20260930220000-0123456789ab": "<sub>Checked by fitness-runner [0123456](https://github.com/may-journal/fitness-runner/commit/0123456789ab)</sub>",
		"v1.2.3":                             "<sub>Checked by fitness-runner [v1.2.3](https://github.com/may-journal/fitness-runner/releases/tag/v1.2.3)</sub>",
		"(devel)":                            "<sub>Checked by fitness-runner (unknown build)</sub>",
	}
	defer func() { buildInfo = debug.ReadBuildInfo }()
	for v, want := range cases {
		buildInfo = func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: v}}, true }
		if got := attribution(); got != want {
			t.Errorf("attribution(%s) = %q, want %q", v, got, want)
		}
	}
}

func TestPlanCheckCommentsOncePerBody(t *testing.T) {
	foot := pinAttribution(t)
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
	want := "✅ Plan looks good.\n\n" + foot + "\n\n<!-- fitness:plan-structure:ba7816bf8f01 -->"
	if gh.comments[5][0].Body != want {
		t.Errorf("comment = %q, want %q", gh.comments[5][0].Body, want)
	}
}

func TestPlanCheckFailureComment(t *testing.T) {
	foot := pinAttribution(t)
	var calls []call
	gh := &fakeGH{}
	ev := ghEvent{name: "issues", Issue: &target{Number: 2, Body: "abc"}}
	c := fakeChecker(map[string]string{"plan-structure": "missing Background"}, nil, &calls)
	if code := planCheck(ev, gh, c); code != 1 {
		t.Fatalf("planCheck = %d, want 1", code)
	}
	want := "❌ Plan needs work:\n- missing Background\n\n" + foot + "\n\n<!-- fitness:plan-structure:ba7816bf8f01 -->"
	if gh.comments[2][0].Body != want {
		t.Errorf("comment = %q, want %q", gh.comments[2][0].Body, want)
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

// runPlan checks body on issue n with the named checks failing.
func runPlan(gh *fakeGH, n int, body string, fail map[string]string) int {
	var calls []call
	ev := ghEvent{name: "issues", Issue: &target{Number: n, Body: body}}
	return planCheck(ev, gh, fakeChecker(fail, nil, &calls))
}

func TestPlanCheckEditsAPassAfterAPass(t *testing.T) {
	foot := pinAttribution(t)
	now = func() time.Time { return time.Date(2026, 9, 30, 22, 15, 0, 0, time.UTC) }
	defer func() { now = time.Now }()
	gh := &fakeGH{}
	runPlan(gh, 4, "one", nil)
	runPlan(gh, 4, "two", nil)
	if len(gh.comments[4]) != 1 {
		t.Fatalf("comments = %d, want the first one edited", len(gh.comments[4]))
	}
	want := "✅ Validated (updated 2026-09-30 22:15 UTC)\n\n" + foot + "\n\n" + planMarker("two")
	if got := gh.comments[4][0].Body; got != want {
		t.Errorf("comment = %q, want %q", got, want)
	}
	runPlan(gh, 4, "two", nil)
	if len(gh.comments[4]) != 1 || gh.comments[4][0].IsHidden {
		t.Errorf("same body re-run changed comments: %+v", gh.comments[4])
	}
}

func TestPlanCheckPostsAPassAfterAFail(t *testing.T) {
	gh := &fakeGH{}
	runPlan(gh, 4, "one", map[string]string{"cspell": "unknown word"})
	runPlan(gh, 4, "two", nil)
	c := gh.comments[4]
	if len(c) != 2 || !c[0].IsHidden || c[1].IsHidden || !strings.HasPrefix(c[1].Body, "✅ Plan looks good.") {
		t.Errorf("comments = %+v, want the fail hidden and a new pass", c)
	}
}

func TestPlanCheckPostsAFailAfterAPass(t *testing.T) {
	gh := &fakeGH{}
	runPlan(gh, 4, "one", nil)
	runPlan(gh, 4, "two", map[string]string{"cspell": "unknown word"})
	c := gh.comments[4]
	if len(c) != 2 || !c[0].IsHidden || !strings.HasPrefix(c[1].Body, "❌") {
		t.Errorf("comments = %+v, want the pass hidden and a new fail", c)
	}
}

func TestPlanCheckHidesEveryOlderVerdict(t *testing.T) {
	gh := &fakeGH{}
	runPlan(gh, 4, "one", map[string]string{"cspell": "a"})
	runPlan(gh, 4, "two", map[string]string{"cspell": "b"})
	runPlan(gh, 4, "three", nil)
	runPlan(gh, 4, "four", nil)
	c := gh.comments[4]
	if len(c) != 3 || !c[0].IsHidden || !c[1].IsHidden || c[2].IsHidden {
		t.Errorf("comments = %+v, want two hidden fails and one live pass", c)
	}
}

// assertLabels checks issue n carries exactly want after the named run.
func assertLabels(t *testing.T, gh *fakeGH, n int, after string, want ...string) {
	t.Helper()
	if got := gh.issues[n].labels; !reflect.DeepEqual(got, want) {
		t.Errorf("labels after %s = %v", after, got)
	}
}

func TestPlanCheckLabelsTheVerdict(t *testing.T) {
	gh := &fakeGH{issues: map[int]fakeIssue{4: {labels: []string{"Plan"}}}}
	runPlan(gh, 4, "one", map[string]string{"cspell": "a"})
	assertLabels(t, gh, 4, "a fail", "Plan", "fitness", "fitness-invalid")
	runPlan(gh, 4, "two", nil)
	assertLabels(t, gh, 4, "a pass", "Plan", "fitness", "fitness-valid")
	if !gh.labels["fitness"] || !gh.labels["fitness-valid"] || !gh.labels["fitness-invalid"] {
		t.Errorf("created labels = %v, want all three", gh.labels)
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

// TestGHClientLive runs the real gh against a public repo, so the query
// output formats are checked end to end. Opt in with FITNESS_LIVE_GH=1.
func TestGHClientLive(t *testing.T) {
	if os.Getenv("FITNESS_LIVE_GH") != "1" {
		t.Skip("set FITNESS_LIVE_GH=1 to run against the real gh")
	}
	gh := ghClient{repo: "may-journal/fitness-runner"}
	checkLiveVerdicts(t, gh)
	checkLiveListings(t, gh)
	checkLiveMergedClose(t, gh)
	checkLiveManualClose(t, gh)
}

// checkLiveVerdicts reads the verdicts on Plan #108.
func checkLiveVerdicts(t *testing.T, gh ghClient) {
	t.Helper()
	vs, err := gh.verdicts(108)
	if err != nil || len(vs) == 0 {
		t.Fatalf("verdicts(108) = %d verdicts, %v", len(vs), err)
	}
	if vs[0].ID == 0 || vs[0].NodeID == "" {
		t.Errorf("verdict ids missing: %+v", vs[0])
	}
}

// checkLiveListings reads one issue's labels and the open PR and Plan lists.
func checkLiveListings(t *testing.T, gh ghClient) {
	t.Helper()
	if _, labels, err := gh.issue(108); err != nil || !hasLabel(labels, "Plan") {
		t.Errorf("issue(108) labels = %v, %v; want Plan", labels, err)
	}
	if _, err := gh.openPRs(); err != nil {
		t.Errorf("openPRs: %v", err)
	}
	if _, err := gh.openPlans(); err != nil {
		t.Errorf("openPlans: %v", err)
	}
}

// checkLiveMergedClose looks up the PR that closed a merged issue.
func checkLiveMergedClose(t *testing.T, gh ghClient) {
	t.Helper()
	// #137 closed when PR #139 was squash-merged, so its closer is a commit.
	if c, err := gh.closingPR(137); err != nil || c.Author == "" || c.Merger == "" {
		t.Errorf("closingPR(137) = %+v, %v; want the PR's author and merger", c, err)
	}
}

// checkLiveManualClose confirms a hand-closed issue has no closing PR.
func checkLiveManualClose(t *testing.T, gh ghClient) {
	t.Helper()
	// #138 was closed by hand as not planned, so there is no closing PR.
	if c, err := gh.closingPR(138); err != nil || c != (closer{}) {
		t.Errorf("closingPR(138) = %+v, %v; want no PR", c, err)
	}
}
