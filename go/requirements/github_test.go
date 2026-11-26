// cspell:ignore xlarge

package requirements

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// closedAtClosed is when the test's issue was closed.
const closedAtClosed = "2026-10-09T12:00:00Z"

// unfinished is an issue body with one unchecked item.
const unfinished = "- [x] Write the doc.\n- [ ] Write the tests.\n"

// closedClosed is the `issues: closed` event GitHub sends when closer-login
// closes issue 21 with body for reason.
func closedClosed(t *testing.T, body, reason string) []string {
	t.Helper()
	issue := map[string]any{"number": 21, "body": body, "state_reason": reason, "closed_at": closedAtClosed}
	return actionsEvent(t, "issues", map[string]any{"action": "closed", "issue": issue, "sender": map[string]any{"login": "closer-login"}})
}

// closeCheckClosed runs close-check on the event, GitHub answering from saved.
func closeCheckClosed(t *testing.T, event []string, saved []response) (replays, string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, event...), "close-check")
	return rp, out, code
}

// commentsClosed is the saved comment list on issue 21, as gh prints it
// through fitness's filter: one object per line.
func commentsClosed(lines string) response {
	return response{Match: []string{"repos/" + repoPR + "/issues/21/comments?per_page=100"}, Stdout: lines}
}

// quietClosed asserts close-check left issue 21 closed and uncommented.
func quietClosed(t *testing.T, rp replays) {
	t.Helper()
	if rp.called("gh", "-X PATCH") || rp.called("gh", "-X POST") {
		t.Errorf("want the issue left closed and quiet, got gh calls %q", rp.calls("gh"))
	}
}

// Issue texts for Test0007_5: an older issue, a new one that repeats
// it in other words, and one about something else.
const (
	offlineTitle = "Sync photos taken offline"
	offlineBody  = "> Photos taken without signal upload once the phone reconnects.\n\n## Background\n\nPhotos taken without signal stay on the phone, and their captions and album order are lost.\n\n## What needs to happen\n\n- [ ] Queue photos taken offline with their captions and album order.\n- [ ] Upload the queue once the phone reconnects.\n"
	repeatTitle  = "Upload photos taken offline after reconnecting"
	repeatBody   = "> Photos taken without signal upload once the phone reconnects.\n\n## Background\n\nPhotos taken without signal stay on the phone, and their captions and album order are lost.\n\n## What needs to happen\n\n- [ ] Queue offline photos with their captions and album order.\n- [ ] Upload the queue once the phone reconnects.\n"
	darkTitle    = "Dark theme for the settings screen"
	darkBody     = "> The settings screen follows the system dark theme.\n\n## Background\n\nSettings stay bright at night.\n\n## What needs to happen\n\n- [ ] Draw settings in dark colors when the system theme is dark.\n"
)

// listedDupe is one issue as the issue list prints it through fitness's
// filter: open unless closedAt is set, under parent unless 0.
func listedDupe(n int, title, body string, parent int, closedAt string) string {
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
// earlier note on issue n as the lookup prints it. Every write to it succeeds.
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

// workflowAutoMerge is the reusable Auto-merge workflow every org repo runs.
const workflowAutoMerge = "../../.github/workflows/auto-merge.yml"

// botAutoMerge is the login the org App comments as.
const botAutoMerge = "may-journal-automation[bot]"

// issuesAutoMerge is the REST path of PR 2's issue in the event's repo.
const issuesAutoMerge = "repos/" + repoPR + "/issues"

// workflowTextAutoMerge returns the workflow file's text.
func workflowTextAutoMerge(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(workflowAutoMerge)
	mustDo(t, err)
	return string(raw)
}

// stepAutoMerge returns the run script of the workflow step named name, its
// indentation dropped, as Actions hands it to bash.
func stepAutoMerge(t *testing.T, name string) string {
	t.Helper()
	_, step, ok := strings.Cut(workflowTextAutoMerge(t), "- name: "+name+"\n")
	if ok {
		step, ok = stepScript(t, step)
	}
	if !ok {
		t.Fatalf("%s has no run script for step %q", workflowAutoMerge, name)
	}
	return blockAutoMerge(step)
}

// blockAutoMerge returns text's leading lines indented at least as deep as its
// first, with that indentation dropped: a YAML block scalar's content.
func blockAutoMerge(text string) string {
	lines := strings.Split(strings.TrimLeft(text, "\n"), "\n")
	indent := lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " "))]
	var script []string
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, indent) {
			break
		}
		script = append(script, strings.TrimPrefix(line, indent))
	}
	return strings.Join(script, "\n")
}

// noteAutoMerge is GitHub listing the bot's auto-merge note on PR 2 as comment
// 7 with body, or listing none when body is empty.
func noteAutoMerge(body string) []response {
	if body == "" {
		return []response{{Match: []string{issuesAutoMerge + "/2/comments --paginate"}}}
	}
	return []response{
		{Match: []string{issuesAutoMerge + "/2/comments --paginate"}, Stdout: "7\n"},
		{Match: []string{issuesAutoMerge + "/comments/7 --jq .body"}, Stdout: body + "\n"},
	}
}

// autoMergeAutoMerge runs the auto-merge step as Actions does for PR 2 on base,
// GitHub answering merge with answer and listing note.
func autoMergeAutoMerge(t *testing.T, base string, answer response, note string, writes ...response) (replays, string, int) {
	t.Helper()
	answer.Match = []string{"pr merge --auto --squash"}
	return runAutoMerge(t, base, append(append([]response{answer}, noteAutoMerge(note)...), writes...))
}

// runAutoMerge runs the auto-merge step on PR 2 based on base, with GitHub
// answering from saved and accepting any other call.
func runAutoMerge(t *testing.T, base string, saved []response) (replays, string, int) {
	t.Helper()
	saved = append(saved, response{Match: []string{"api "}})
	rp := standIn(t, map[string][]response{"gh": saved})
	env := actionsEvent(t, "pull_request", map[string]any{
		"action":       "synchronize",
		"repository":   map[string]any{"default_branch": "main"},
		"pull_request": map[string]any{"number": 2, "html_url": "https://github.com/" + repoPR + "/pull/2", "base": map[string]any{"ref": base}},
	})
	env = append(env, "GH_TOKEN=app-token", "BOT="+botAutoMerge, "GITHUB_ENV="+filepath.Join(t.TempDir(), "env"))
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", stepAutoMerge(t, "Enable squash auto-merge"))
		cmd.Dir, cmd.Env = t.TempDir(), append(append(userEnv(), rp.env...), env...)
		return cmd
	})
	return rp, out, code
}

// loggedAutoMerge is a note as the stand-in logs the call that writes it.
func loggedAutoMerge(text string) string {
	return "body=" + strings.ReplaceAll(text+"\n\n<!-- fitness:auto-merge -->", "\n", `\n`)
}

// postedAutoMerge returns the note text the step posted on PR 2, or "".
func postedAutoMerge(rp replays) string {
	return postedOnAutoMerge(rp, 2)
}

// postedOnAutoMerge returns the note text the step posted on PR n, or "".
func postedOnAutoMerge(rp replays, n int) string {
	for _, c := range rp.calls("gh") {
		if _, body, ok := strings.Cut(c, "api "+issuesAutoMerge+"/"+itoa(n)+"/comments -f body="); ok {
			return body
		}
	}
	return ""
}

// stackedAutoMerge is the note on a PR based on photos-grid.
const stackedAutoMerge = "Auto-merge is off: this PR targets `photos-grid`, not `main`, in a branch chain. " +
	"The bot cannot merge a chain: merging this PR into `photos-grid` pushes to that PR and dismisses its approval. " +
	"Merge it by hand, or rebuild the chain with `gh stack`, which the bot merges whole. " +
	"Once GitHub moves this PR onto `main`, auto-merge turns on and this comment says so."

var (
	eventRefAutoMerge = regexp.MustCompile(`github\.event\.[\w.]+`)
	nullAutoMerge     = regexp.MustCompile(`\bnull\b`)
)

// runsAutoMerge evaluates the job's `if:` for a pull_request event holding every
// field it reads, with "" for null and "on" for auto_merge.
func runsAutoMerge(t *testing.T, event map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(workflowTextAutoMerge(t), "    if: >-\n")
	if !ok {
		t.Fatalf("%s has no job-level if", workflowAutoMerge)
	}
	return holdsAutoMerge(t, strings.ReplaceAll(blockAutoMerge(cond), "\n", " "), event)
}

// holdsAutoMerge evaluates an Actions condition cond for event, whose fields are
// keyed by path under github.event, "event_name" naming the event.
func holdsAutoMerge(t *testing.T, cond string, event map[string]string) bool {
	t.Helper()
	cond = strings.ReplaceAll(cond, "github.event_name", "github.event.event_name")
	cond = nullAutoMerge.ReplaceAllString(cond, `""`)
	cond = eventRefAutoMerge.ReplaceAllStringFunc(cond, func(ref string) string {
		value, given := event[strings.TrimPrefix(ref, "github.event.")]
		if !given {
			t.Fatalf("the job's if reads %s, which the test event does not give", ref)
		}
		return `"` + value + `"`
	})
	return exec.Command("bash", "-c", "[[ "+cond+" ]]").Run() == nil
}

// workflowReleaseApprove is the reusable workflow that approves green release PRs.
const workflowReleaseApprove = "../../.github/workflows/release-approve.yml"

// authorReleaseApprove is the login gh shows for the automation App's PRs.
const authorReleaseApprove = "app/may-journal-automation"

// sectionReleaseApprove returns the workflow text of the step named name, up to the
// next step.
func sectionReleaseApprove(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(workflowReleaseApprove)
	mustDo(t, err)
	_, rest, ok := strings.Cut(string(raw), "- name: "+name+"\n")
	if !ok {
		t.Fatalf("%s has no step %q", workflowReleaseApprove, name)
	}
	section, _, _ := strings.Cut(rest, "      - name: ")
	return section
}

// runReleaseApprove runs the named step's script as Actions does for release PR 23 at
// commit abc123 in run 99, with env overriding.
func runReleaseApprove(t *testing.T, name string, gh []response, env ...string) (replays, string, int, string) {
	t.Helper()
	script, ok := stepScript(t, sectionReleaseApprove(t, name))
	if !ok {
		t.Fatalf("step %q has no run script", name)
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	output := filepath.Join(t.TempDir(), "output")
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", blockAutoMerge(script))
		cmd.Dir = t.TempDir()
		cmd.Env = append(append(append(userEnv(), rp.env...), "PR=23", "SHA=abc123", "GH_TOKEN=app-token", "GH_REPO="+repoPR,
			"GITHUB_REPOSITORY="+repoPR, "GITHUB_OUTPUT="+output, "GITHUB_RUN_ID=99", "AUTHOR="+authorReleaseApprove,
			"TRIES=2", "WAIT_SECONDS=0", "KIND=release", "BOT=may-journal-approver"), env...)
		return cmd
	})
	written, _ := os.ReadFile(output)
	return rp, out, code, string(written)
}

// viewReleaseApprove is the CLI's answer to reading PR 23 by author, open as a ready PR
// at commit abc123.
func viewReleaseApprove(author string) response {
	return response{Match: []string{"pr view 23 --json author"},
		Stdout: `{"author":{"login":"` + author + `"},"state":"OPEN","isDraft":false,"headRefOid":"abc123"}` + "\n"}
}

// filesReleaseApprove is GitHub's paged answer listing PR 23's files, one page each.
func filesReleaseApprove(pages ...string) response {
	return response{Match: []string{"pulls/23/files --paginate"}, Stdout: strings.Join(pages, "\n") + "\n"}
}

// releaseFilesReleaseApprove is a release PR's files, over two pages.
var releaseFilesReleaseApprove = filesReleaseApprove(`[{"filename":"CHANGELOG.md"},{"filename":"version.txt"}]`, `[{"filename":".release-please-manifest.json"}]`)

// guardReleaseApprove runs the release PR check with gh answering from gh.
func guardReleaseApprove(t *testing.T, gh ...response) (replays, string, int, string) {
	t.Helper()
	return runReleaseApprove(t, "Check the release PR", gh)
}

// checksReleaseApprove is the CLI's answer listing PR 23's checks as JSON, exiting as gh
// does: 8 while any is pending.
func checksReleaseApprove(json string, exit int) response {
	return response{Match: []string{"pr checks 23 --json bucket,link"}, Stdout: json + "\n", Exit: exit}
}

// headReleaseApprove is the CLI's answer reading PR 23's head commit.
func headReleaseApprove(sha string) response {
	return response{Match: []string{"pr view 23 --json headRefOid"}, Stdout: sha + "\n"}
}

// waitReleaseApprove runs the green-checks wait with gh answering from gh.
func waitReleaseApprove(t *testing.T, gh ...response) (replays, string, int, string) {
	t.Helper()
	return runReleaseApprove(t, "Wait for green checks", gh)
}

// reviewsReleaseApprove is GitHub listing PR 23's reviews as reviews, and accepting
// a new one.
func reviewsReleaseApprove(reviews string) []response {
	return []response{{Match: []string{"pulls/23/reviews --paginate"}, Stdout: reviews + "\n"}, {Match: []string{"-X POST"}, Stdout: "{}\n"}}
}

// approvedReleaseApprove is a page of reviews holding the approver's approval of
// commit abc123.
const approvedReleaseApprove = `[{"user":{"login":"jasonvanderslice"},"commit_id":"abc000","state":"APPROVED"},` +
	`{"user":{"login":"may-journal-approver"},"commit_id":"abc123","state":"APPROVED"}]`

// tokensReleaseApprove asserts that only the approval uses the approver's token, and
// every read of the PR and its checks uses the run's own.
func tokensReleaseApprove(t *testing.T) {
	t.Helper()
	for step, token := range map[string]string{
		"Check the release PR":      "github.token",
		"Check the dependency bump": "github.token",
		"Wait for green checks":     "github.token",
		"Approve the PR":            "secrets.APPROVER_TOKEN",
	} {
		if want := "GH_TOKEN: ${{ " + token + " }}\n"; !strings.Contains(sectionReleaseApprove(t, step), want) {
			t.Errorf("step %q does not use %s", step, token)
		}
	}
}

var outputReleaseApprove = regexp.MustCompile(`(?:steps\.(\w+)\.outputs|inputs)\.(\w+)`)

// runsReleaseApprove evaluates the step's `if:` as Actions does, given earlier steps'
// outputs keyed step.output and inputs keyed inputs.name, missing ones empty.
func runsReleaseApprove(t *testing.T, name string, outputs map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(sectionReleaseApprove(t, name), "if: ${{ ")
	if !ok {
		t.Fatalf("step %q has no if", name)
	}
	cond, _, _ = strings.Cut(cond, " }}\n")
	if cond == "always()" {
		return true
	}
	cond = outputReleaseApprove.ReplaceAllStringFunc(cond, func(ref string) string {
		m := outputReleaseApprove.FindStringSubmatch(ref)
		key := m[1] + "." + m[2]
		if m[1] == "" {
			key = "inputs." + m[2]
		}
		return `"` + outputs[key] + `"`
	})
	return exec.Command("bash", "-c", "[[ "+strings.ReplaceAll(cond, "'", `"`)+" ]]").Run() == nil
}

// skipsAllButReportReleaseApprove asserts that every step after the token check skips
// but the report when the token is missing.
func skipsAllButReportReleaseApprove(t *testing.T) {
	t.Helper()
	for _, step := range []string{"Check the release PR", "Check the dependency bump", "Wait for green checks", "Approve the PR"} {
		if runsReleaseApprove(t, step, map[string]string{}) {
			t.Errorf("without the approver token, step %q runs, want it skipped", step)
		}
	}
	if !runsReleaseApprove(t, "Report job outcome", map[string]string{}) {
		t.Error("without the approver token, the report step skips, want it run")
	}
}

// manifestDepApprove is a package.json with one runtime dependency, zod at zod,
// and the dev dependency parser at parser.
func manifestDepApprove(zod, parser string) map[string]any {
	return map[string]any{
		"name": "example", "version": "1.0.0",
		"dependencies":    map[string]any{"zod": zod},
		"devDependencies": map[string]any{"@typescript-eslint/parser": parser, "vitest": "^3.2.0"},
	}
}

// lockDepApprove is the lockfile for manifest, with packages its installed
// packages beside the root entry.
func lockDepApprove(manifest map[string]any, packages map[string]any) map[string]any {
	all := map[string]any{"": manifest}
	for name, entry := range packages {
		all[name] = entry
	}
	return map[string]any{"name": "example", "version": "1.0.0", "lockfileVersion": 3, "requires": true, "packages": all}
}

// installedDepApprove is a lockfile package entry at version, dev-only when dev.
func installedDepApprove(version string, dev bool) map[string]any {
	if dev {
		return map[string]any{"version": version, "dev": true}
	}
	return map[string]any{"version": version}
}

// sideDepApprove is one commit's package.json and lockfile.
type sideDepApprove struct{ manifest, lock map[string]any }

// atDepApprove is a commit with zod and parser in package.json, installed as
// zodLock and parserLock, plus dev-only tiny 0.3.5 and extra packages.
func atDepApprove(zod, parser, zodLock, parserLock string, extra map[string]any) sideDepApprove {
	m := manifestDepApprove(zod, parser)
	packages := map[string]any{
		"node_modules/zod":                       installedDepApprove(zodLock, false),
		"node_modules/@typescript-eslint/parser": installedDepApprove(parserLock, true),
		"node_modules/tiny":                      installedDepApprove("0.3.5", true),
	}
	for name, entry := range extra {
		packages[name] = entry
	}
	return sideDepApprove{m, lockDepApprove(m, packages)}
}

// baseDepApprove is the base commit every bump starts from.
var baseDepApprove = atDepApprove("^3.23.0", "^8.71.2", "3.23.8", "8.71.2", nil)

// contentsDepApprove is GitHub's answer reading file at commit ref as raw JSON.
func contentsDepApprove(t *testing.T, file, ref string, body map[string]any) response {
	t.Helper()
	raw, err := json.Marshal(body)
	mustDo(t, err)
	return response{Match: []string{"contents/" + file + "?ref=" + ref}, Stdout: string(raw)}
}

// bumpDepApprove runs the dependency check on PR 23, titled title, at head abc123
// on base tip000 with merge base base000, listing files.
func bumpDepApprove(t *testing.T, title string, files response, head sideDepApprove) (replays, string, int, string) {
	t.Helper()
	view := response{Match: []string{"pr view 23 --json title"},
		Stdout: `{"title":"` + title + `","state":"OPEN","isDraft":false,"headRefOid":"abc123","baseRefOid":"tip000"}` + "\n"}
	mergeBase := response{Match: []string{"compare/tip000...abc123 --jq .merge_base_commit.sha"}, Stdout: "base000\n"}
	return runReleaseApprove(t, "Check the dependency bump", []response{
		view, files, mergeBase,
		contentsDepApprove(t, "package.json", "base000", baseDepApprove.manifest), contentsDepApprove(t, "package-lock.json", "base000", baseDepApprove.lock),
		contentsDepApprove(t, "package.json", "abc123", head.manifest), contentsDepApprove(t, "package-lock.json", "abc123", head.lock),
	})
}

// bothDepApprove lists package.json and its lockfile as PR 23's files.
var bothDepApprove = filesReleaseApprove(`[{"filename":"package.json"},{"filename":"package-lock.json"}]`)

// titleDepApprove is a dependency bump's title.
const titleDepApprove = "build(deps): update @typescript-eslint/parser to 8.72.0"

// defaultDepApprove returns the default of the workflow's kind input.
func defaultDepApprove(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(workflowReleaseApprove)
	mustDo(t, err)
	_, kind, ok := strings.Cut(string(raw), "      kind:\n")
	if ok {
		_, kind, ok = strings.Cut(kind, "default: ")
	}
	if !ok {
		t.Fatalf("%s has no default for its kind input", workflowReleaseApprove)
	}
	value, _, _ := strings.Cut(kind, "\n")
	return value
}

// pullsStack is the REST path of pull requests in the event's repo.
const pullsStack = "repos/" + repoPR + "/pulls"

// viewStack is GitHub showing PR n with its review decision, mergeable and
// a Fitness suite check that ended in conclusion. Auto-merge's own check
// is always still running, as it is while the job runs.
func viewStack(n int, review, conclusion string) response {
	checks := `[{"__typename":"CheckRun","workflowName":"Fitness suite","status":"COMPLETED","conclusion":"` + conclusion + `"},` +
		`{"__typename":"CheckRun","workflowName":"Auto-merge on ready","status":"IN_PROGRESS","conclusion":""},` +
		`{"__typename":"StatusContext","context":"ci/legacy","state":"SUCCESS"}]`
	return response{
		Match:  []string{"pr view " + itoa(n) + " --repo"},
		Stdout: `{"isDraft":false,"reviewDecision":"` + review + `","mergeable":"MERGEABLE","statusCheckRollup":` + checks + `}` + "\n",
	}
}

// stackPRsStack lists stack 7 as GitHub does: PR 1 merged, PR 2 open, PR 3 on
// PR 2 at head bbb.
const stackPRsStack = `[{"number":1,"state":"closed","head":{"ref":"s0","sha":"zzz"},"base":{"ref":"main"}},` +
	`{"number":2,"state":"open","head":{"ref":"s1","sha":"aaa"},"base":{"ref":"s0"}},` +
	`{"number":3,"state":"open","head":{"ref":"s2","sha":"bbb"},"base":{"ref":"s1"}}]`

// stackStack runs the auto-merge step on PR 2, bottom of stack 7 topped by PR
// 3, answering with views, merge and poll.
func stackStack(t *testing.T, views []response, merge, poll response) (replays, string, int) {
	t.Helper()
	merge.Match = []string{"-X PUT " + pullsStack + "/3/merge-async"}
	poll.Match = []string{"api " + pullsStack + "/3/merge-async/u1"}
	saved := append([]response{
		{Match: []string{"api " + pullsStack + "/2"}, Stdout: `{"number":2,"stack":{"number":7,"position":1,"size":2}}` + "\n"},
		{Match: []string{"api repos/" + repoPR + "/stacks/7"}, Stdout: `{"number":7,"pull_requests":` + stackPRsStack + `}` + "\n"},
		poll, merge,
	}, views...)
	return runAutoMerge(t, "main", saved)
}

// approvedStack shows both PRs of the stack approved, green and mergeable.
var approvedStack = []response{viewStack(2, "APPROVED", "SUCCESS"), viewStack(3, "APPROVED", "SUCCESS")}

// pendingStack is GitHub accepting a merge request it runs as u1.
var pendingStack = response{Stdout: `{"status":"pending","details":{"message":"Merge request enqueued.","uuid":"u1","merge_method":"squash","bypass_rules":false}}` + "\n"}

// failedStack is GitHub's answer for a stack merge its rules stopped.
const failedStack = `{"status":"failed","details":{"message":"Required workflows 'Fitness suite' failed"}}` + "\n"

// notedAllStack checks that PRs 2 and 3 both got note.
func notedAllStack(t *testing.T, rp replays, note string) {
	t.Helper()
	for _, n := range []int{2, 3} {
		if !rp.called("gh", "api "+issuesAutoMerge+"/"+itoa(n)+"/comments", loggedAutoMerge(note)) {
			t.Fatalf("PR %d of the stack did not get the note %q: %q", n, note, rp.calls("gh"))
		}
	}
}

// waitsStack checks that the stack merged nothing and both PRs got the note
// saying which one waits and why.
func waitsStack(t *testing.T, rp replays, out string, code int, why string) {
	t.Helper()
	sees(t, out, code, 0, "::notice::Stack #7 waits: "+why+".")
	if rp.count("gh", "merge-async")+rp.count("gh", "pr merge") > 0 {
		t.Fatalf("a stack that is not ready asked GitHub to merge: %q", rp.calls("gh"))
	}
	notedAllStack(t, rp, "Auto-merge waits: stack #7 (#2 #3) merges into `main` once every PR in it is approved, green and mergeable. Waiting: "+why+".")
}

// refusedStack checks that an unmergeable stack failed the job naming GitHub's
// answer, and only its top PR got a note on why.
func refusedStack(t *testing.T, rp replays, out string, code int, why, note string) {
	t.Helper()
	sees(t, out, code, 1, "::error::GitHub "+why+" stack #7 through #3", "Required workflows 'Fitness suite' failed")
	if !rp.called("gh", "api "+issuesAutoMerge+"/3/comments", loggedAutoMerge(note)) {
		t.Fatalf("the top PR of a stack GitHub would not merge got no note saying why: %q", rp.calls("gh"))
	}
	if postedOnAutoMerge(rp, 2) != "" {
		t.Fatalf("a lower PR of a stack GitHub would not merge got a note: %q", rp.calls("gh"))
	}
}

// callerStack is the caller that runs Auto-merge on ready and on reviews.
const callerStack = "../../.github/workflows/auto-merge-on-ready.yml"

// callerIfStack returns the caller's job-level condition, failing unless
// it runs on submitted reviews.
func callerIfStack(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(callerStack)
	mustDo(t, err)
	text := string(raw)
	_, cond, ok := strings.Cut(text, "    if: ")
	if !ok || !strings.Contains(text, "  pull_request_review:\n    types: [submitted]\n") {
		t.Fatalf("%s does not run on submitted reviews behind a job-level if", callerStack)
	}
	cond, _, _ = strings.Cut(cond, "\n")
	// holdsAutoMerge reads only event fields, so the repository is filled in here.
	cond = strings.ReplaceAll(cond, "github.repository", `"`+repoPR+`"`)
	return cond
}

// reviewStack is a submitted review event in state on a branch of repo.
func reviewStack(state, repo string) map[string]string {
	return map[string]string{"event_name": "pull_request_review", "review.state": state, "pull_request.head.repo.full_name": repo}
}

// actionsJob is one job as the run's job list prints it: conclusion, start
// and end, tab-separated.
func actionsJob(conclusion, start, end string) string {
	return conclusion + "\t" + start + "\t" + end + "\n"
}

// minutesRun runs `fitness-release minutes` with args, gh answering saved.
func minutesRun(t *testing.T, saved []response, args ...string) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), rp.env, append([]string{"minutes"}, args...)...)
	return out, code, rp
}

// finishedParents is the body of the child just closed, with every item ticked.
const finishedParents = "- [x] Write the doc.\n"

// listParents is a parent body whose list names the issues.
func listParents(nums ...int) string {
	body := "> A parent.\n\n## What needs to happen\n\n"
	for _, n := range nums {
		body += fmt.Sprintf("- #%d\n", n)
	}
	return body
}

// issuePathParents is the REST path of issue n.
func issuePathParents(n int) string {
	return fmt.Sprintf("repos/%s/issues/%d", repoPR, n)
}

// noParentParents is GitHub's answer that issue n has no parent issue.
func noParentParents(n int) []response {
	return []response{
		{Match: []string{issuePathParents(n) + "/parent"}, Stderr: "gh: Not Found (HTTP 404)", Exit: 1},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: ""},
	}
}

// parentOfParents is GitHub's answer that issue n has parent p as a sub-issue.
func parentOfParents(n, p int) []response {
	return []response{
		{Match: []string{issuePathParents(n) + "/parent"}, Stdout: fmt.Sprintf("{\"number\":%d,\"repo\":%q}\n", p, repoPR)},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: ""},
	}
}

// mentionedByParents is GitHub's answer that issue n has no parent but is named in
// the body of issue p.
func mentionedByParents(n, p int) []response {
	return []response{
		{Match: []string{issuePathParents(n) + "/parent"}, Stderr: "gh: Not Found (HTTP 404)", Exit: 1},
		{Match: []string{"search/issues", fmt.Sprintf("%%23%d%%22", n)}, Stdout: fmt.Sprintf("{\"number\":%d}\n", p)},
	}
}

// issueParents is GitHub's answer for reading issue n: its state, close reason
// and body.
func issueParents(t *testing.T, n int, state, reason, body string) response {
	t.Helper()
	out, err := json.Marshal(map[string]string{"state": state, "reason": reason, "body": body})
	mustDo(t, err)
	return response{Match: []string{"{state, reason", issuePathParents(n)}, Stdout: string(out) + "\n"}
}

// subIssuesParents is GitHub's answer for the sub-issues of issue n.
func subIssuesParents(n int, kids ...int) response {
	var out string
	for _, k := range kids {
		out += fmt.Sprintf("{\"number\":%d,\"repo\":%q}\n", k, repoPR)
	}
	return response{Match: []string{issuePathParents(n) + "/sub_issues"}, Stdout: out}
}

// writesParents are the answers to commenting on and closing issue n.
func writesParents(n int) []response {
	return []response{
		{Match: []string{"-X POST", issuePathParents(n) + "/comments"}, Stdout: "{}"},
		{Match: []string{"-X PATCH", "state=closed", issuePathParents(n)}, Stdout: "{}"},
	}
}

// closeParentsParents runs close-check on the closing of issue 21 as completed.
func closeParentsParents(t *testing.T, saved ...[]response) (replays, string, int) {
	t.Helper()
	var all []response
	for _, s := range saved {
		all = append(all, s...)
	}
	return closeCheckClosed(t, closedClosed(t, finishedParents, "completed"), all)
}

// closedParentParents asserts issue n was commented once and closed as completed.
func closedParentParents(t *testing.T, rp replays, n int) {
	t.Helper()
	if !rp.called("gh", "-X PATCH", "state=closed", "state_reason=completed", issuePathParents(n)) {
		t.Errorf("want issue %d closed as completed, got gh calls %q", n, rp.calls("gh"))
	}
	if c := rp.count("gh", "-X POST", issuePathParents(n)+"/comments"); c != 1 {
		t.Errorf("want one comment on issue %d, got %d in gh calls %q", n, c, rp.calls("gh"))
	}
}

// stayedOpenReopen asserts close-check closed and commented on nothing.
func stayedOpenReopen(t *testing.T, rp replays) {
	t.Helper()
	if rp.called("gh", "-X PATCH") || rp.called("gh", "-X POST") {
		t.Errorf("want the parent left open and quiet, got gh calls %q", rp.calls("gh"))
	}
}

// boldReadme is a readme markdown-no-bold-italic fails.
const boldReadme = readme + "\nSome **bold** text.\n"

// prBase is happyRepo with every check passing and files written over it,
// committed with origin/main at that commit, as a pull request's base.
func prBase(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", with(architectedRepo(), files))
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	return repo
}

// prChanges commits files on the branch and runs the suite as a pull
// request's GitHub Actions job does. It returns the output, exit code and job
// summary.
func prChanges(t *testing.T, repo string, files map[string]string) (out string, code int, summary string) {
	t.Helper()
	commit(t, repo, files, "docs(changelog): note the guide")
	path := filepath.Join(t.TempDir(), "summary.md")
	out, code = fitness(t, repo, append([]string{"GITHUB_STEP_SUMMARY=" + path}, prEnv...), "--policy=external")
	written, _ := os.ReadFile(path)
	return out, code, string(written)
}

// noteGuide is a change to happyRepo's changelog that passes every check:
// one more bullet.
func noteGuide(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "tests", "examples", "happyRepo", "CHANGELOG.md"))
	mustDo(t, err)
	return map[string]string{"CHANGELOG.md": string(raw) + "- Docs: add the guide.\n"}
}

// boldFinding is the error markdown-no-bold-italic reports for boldReadme.
const boldFinding = "README.md: disallowed **bold**"

// suiteSuite is the workflow the org ruleset runs on every pull request, and
// reusableSuite the build it calls.
const (
	suiteSuite    = "../../.github/workflows/fitness-suite.yml"
	reusableSuite = "../../.github/workflows/ci-reusable.yml"
)

// pickSuite is the tail of the suite's runs-on input that hands the variable
// on when the condition before it holds, and nothing otherwise.
const pickSuite = " && vars.FITNESS_RUNS_ON || ''"

// callSuite matches startsWith or endsWith on the variable with a literal.
var callSuite = regexp.MustCompile(`(startsWith|endsWith)\(vars\.FITNESS_RUNS_ON, '([^']*)'\)`)

// runsOnSuite returns the expression on the file's first runs-on line.
func runsOnSuite(t *testing.T, file string) string {
	t.Helper()
	raw, err := os.ReadFile(file)
	mustDo(t, err)
	_, line, ok := strings.Cut(string(raw), "runs-on: ${{ ")
	expr, _, closed := strings.Cut(line, " }}\n")
	if !ok || !closed {
		t.Fatalf("%s has no runs-on expression", file)
	}
	return expr
}

// labelSuite is the runner label the suite passes to ci-reusable when the
// repo's FITNESS_RUNS_ON holds value. Bash evaluates the condition as
// Actions does, ignoring case; an empty label keeps ci-reusable's choice.
func labelSuite(t *testing.T, value string) string {
	t.Helper()
	expr := runsOnSuite(t, suiteSuite)
	cond, ok := strings.CutSuffix(expr, pickSuite)
	if !ok {
		t.Fatalf("the suite's runs-on %q does not end with %q", expr, pickSuite)
	}
	cond = callSuite.ReplaceAllStringFunc(cond, func(call string) string {
		m := callSuite.FindStringSubmatch(call)
		if m[1] == "startsWith" {
			return ` "$V" == '` + m[2] + `'*`
		}
		return ` "$V" == *'` + m[2] + `'`
	})
	if strings.Contains(cond, "vars.") {
		t.Fatalf("the suite's runs-on condition %q reads the variable some other way", cond)
	}
	cmd := exec.Command("bash", "-c", "shopt -s nocasematch; [[ "+cond+" ]]")
	cmd.Env = append(os.Environ(), "V="+value)
	if cmd.Run() == nil {
		return value
	}
	return ""
}

// completedDependabot is what `fitness dependabot-pr` wrote to GitHub for a PR:
// its title, description and the changelog entry it committed.
type completedDependabot struct {
	title, body, file, entry string
	issueMade                bool
}

// payloadDependabot is a recorded Dependabot pull_request_target payload, with its
// author replaced by login.
func payloadDependabot(t *testing.T, login string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("testdata/dependabotPR.json")
	mustDo(t, err)
	var ev map[string]any
	mustDo(t, json.Unmarshal(raw, &ev))
	ev["pull_request"].(map[string]any)["user"] = map[string]any{"login": login}
	return ev
}

// completeDependabot runs dependabot-pr on the payload, GitHub listing openIssue as the open issues.
func completeDependabot(t *testing.T, payload map[string]any, openIssue string) (completedDependabot, replays) {
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
	env := actionsEvent(t, "pull_request_target", payload)
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "dependabot-pr")
	sees(t, out, code, 0)
	return readWritesDependabot(t, rp.calls("gh")), rp
}

// readWritesDependabot reads what the saved gh calls wrote to GitHub.
func readWritesDependabot(t *testing.T, calls []string) completedDependabot {
	t.Helper()
	var got completedDependabot
	for _, call := range calls {
		call = strings.ReplaceAll(call, `\n`, "\n")
		got.issueMade = got.issueMade || strings.Contains(call, "--jq .number")
		if strings.Contains(call, "-X PUT") {
			got.file, got.entry = entryWriteDependabot(t, call)
		}
		if strings.Contains(call, "-X PATCH") {
			got.title = betweenDependabot(call, "-f title=", " -f body=")
			got.body = betweenDependabot(call, "-f body=", " repos/"+repoPR+"/pulls/7")
		}
	}
	return got
}

// entryWriteDependabot is the path and text a contents write committed.
func entryWriteDependabot(t *testing.T, call string) (string, string) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(betweenDependabot(call, "-f content=", " -f branch="))
	mustDo(t, err)
	return betweenDependabot(call, "contents/", ""), string(raw)
}

// betweenDependabot is the text of s after from and before to; "" for to reads to the end.
func betweenDependabot(s, from, to string) string {
	_, rest, _ := strings.Cut(s, from)
	if to == "" {
		return strings.TrimSpace(rest)
	}
	head, _, _ := strings.Cut(rest, to)
	return head
}

// judgeDependabot runs pr-check on the completed PR, GitHub holding the bump issue
// and the PR's changelog entry file.
func judgeDependabot(t *testing.T, got completedDependabot) (string, int) {
	t.Helper()
	patch := "@@ -0,0 +1 @@\n+" + strings.ReplaceAll(strings.TrimSpace(got.entry), "\n", "\n+")
	file, err := json.Marshal(map[string]string{"filename": got.file, "status": "added", "patch": patch})
	mustDo(t, err)
	rp := standIn(t, map[string][]response{"gh": {
		{Match: []string{"pulls/7/files", "startswith"}, Stdout: string(file) + "\n"},
		{Match: []string{"pulls/7/files"}},
		{Match: []string{"pulls/7/commits"}},
		{Match: []string{"sha: .head.sha", "pulls?state=open&per_page=100"}},
		issuePR(t, 31, "Dependabot proposed a bump.\n"),
	}})
	pr := map[string]any{"number": 7, "title": got.title, "body": got.body}
	env := actionsEvent(t, "pull_request", map[string]any{"action": "opened", "pull_request": pr})
	return fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
}

// jobLineCheckout starts a job under a workflow's jobs key.
var jobLineCheckout = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)

// jobsCheckout splits a workflow's text into its jobs' texts by name.
func jobsCheckout(text string) map[string]string {
	jobs := map[string]string{}
	_, body, _ := strings.Cut(text, "\njobs:\n")
	name := ""
	for _, line := range strings.Split(body, "\n") {
		if m := jobLineCheckout.FindStringSubmatch(line); m != nil {
			name = m[1]
		}
		jobs[name] += line + "\n"
	}
	return jobs
}

// checksOutFirst reports whether a job's text checks out the repo before it
// runs the fitness-command action, or runs no fitness command at all.
func checksOutFirst(job string) bool {
	command := strings.Index(job, "actions/fitness-command")
	checkout := strings.Index(job, "actions/checkout@")
	return command < 0 || (checkout >= 0 && checkout < command)
}

// installDeleted runs the fitness-install action's source build on ref, as an
// Actions run of event would, with go and gh answering as saved. It returns
// the output, the exit code, the stand-ins and the PATH file the script wrote.
func installDeleted(t *testing.T, repo, ref, event string, goAnswers, ghAnswers []response) (string, int, replays, string) {
	t.Helper()
	rp := standIn(t, map[string][]response{"go": goAnswers, "gh": ghAnswers})
	dir := t.TempDir()
	write(t, dir, map[string]string{"event.json": event})
	pathFile := filepath.Join(dir, "path")
	mustDo(t, os.WriteFile(pathFile, nil, 0o644))
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(fitnessCI(t), "install-source "+ref)
		cmd.Env = append(userEnv(), rp.env...)
		cmd.Env = append(cmd.Env, "GITHUB_EVENT_PATH="+filepath.Join(dir, "event.json"), "GITHUB_REPOSITORY="+repo, "GITHUB_PATH="+pathFile)
		return cmd
	})
	return out, code, rp, pathFile
}

// goAnswersDeleted is go failing to fetch gone and building anything else.
func goAnswersDeleted(gone string) []response {
	return []response{
		{Match: []string{"install", "@" + gone}, Stderr: "unknown revision " + gone + "\n", Exit: 1},
		{Match: []string{"install"}},
		{Match: []string{"env", "GOPATH"}, Stdout: "/gopath\n"},
	}
}

// pullDeleted is GitHub's answer for pull request 7 with head deadbeef.
func pullDeleted(merged string) []response {
	return []response{{Match: []string{"pulls/7"}, Stdout: `{"merged":` + merged + `,"head":{"sha":"deadbeef"},"merge_commit_sha":"cafe1234"}` + "\n"}}
}

// actionCIFetch is the text of the shared action name.
func actionCIFetch(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "actions", name, "action.yml"))
	mustDo(t, err)
	return string(data)
}

// stepCIFetch is the text of the step named name in text, up to the next step.
func stepCIFetch(t *testing.T, text, name string) string {
	t.Helper()
	_, step, found := strings.Cut(text, "- name: "+name+"\n")
	if !found {
		t.Fatalf("no step named %q in:\n%s", name, text)
	}
	step, _, _ = strings.Cut(step, "\n    - ")
	return step
}

// hasLine reports whether a line of text, trimmed, is line.
func hasLine(text, line string) bool {
	return slices.ContainsFunc(strings.Split(text, "\n"), func(l string) bool { return strings.TrimSpace(l) == line })
}

// scriptCIFetch runs the fitness-ci subcommand a shared action's step names,
// as the fitness-ci action passes it, and returns the outputs it wrote.
func scriptCIFetch(t *testing.T, sub string, env ...string) map[string]string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "output")
	cmd := exec.Command(fitnessCI(t), sub)
	cmd.Env = append(append(os.Environ(), "GITHUB_OUTPUT="+output), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fitness-ci %s: %v\n%s", sub, err, out)
	}
	data, _ := os.ReadFile(output)
	outputs := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			outputs[key] = value
		}
	}
	return outputs
}

// indexHashCIFetch is the SHA-256 of the release index GitHub serves now.
func indexHashCIFetch(t *testing.T) string {
	t.Helper()
	response, err := http.Get("https://github.com/may-journal/fitness-checks/releases/download/fitness-check-index/index.json")
	mustDo(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	mustDo(t, err)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// cacheCIFetch is a home whose checks cache holds programs a and b, with a
// check set linking only a.
func cacheCIFetch(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cache := filepath.Join(home, ".cache", "fitness")
	write(t, cache, map[string]string{"checks/a-1/fitness-check-a": "a", "checks/b-1/fitness-check-b": "b"})
	mustDo(t, os.MkdirAll(filepath.Join(cache, "check-sets", "run"), 0o755))
	mustDo(t, os.Symlink(filepath.Join(cache, "checks", "a-1", "fitness-check-a"), filepath.Join(cache, "check-sets", "run", "fitness-check-a")))
	return home
}
