// cspell:ignore xlarge sdkmanrc

package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func Test0007_1(t *testing.T) {
	cases{
		{"a pull request whose changed files pass while a file it does not change fails", func(t *testing.T) {
			t.Parallel()
			out, code, summary := prChanges(t, prBase(t, map[string]string{"README.md": boldReadme}), noteGuide(t))
			sees(t, out, code, 1, "Stage 1 of 2: checking the files this pull request changes: 1", allPassed(), "Stage 2 of 2: the changes pass",
				boldFinding, "Stage 2 failed", "Files this pull request does not change:", "- README.md (markdown-no-bold-italic)")
			sees(t, summary, code, 1, "## Stage 1 of 2", "## Stage 2 failed", "- README.md (markdown-no-bold-italic)")
		}},
		{"a pull request in a repo whose every file passes: its changes stage and then its full scan stage pass", func(t *testing.T) {
			t.Parallel()
			out, code, _ := prChanges(t, prBase(t, nil), noteGuide(t))
			sees(t, out, code, 0, "Stage 1 of 2:", "Stage 2 of 2:")
			if strings.Count(out, allPassed()) != 2 {
				t.Errorf("both stages must pass every check:\n%s", out)
			}
		}},
		{"a pull request in a repo whose config sets scopedPullRequests, and a file it does not change fails", func(t *testing.T) {
			t.Parallel()
			repo := prBase(t, map[string]string{"README.md": boldReadme, ".fitnessrc.json": `{"scopedPullRequests": true}` + "\n"})
			out, code, _ := prChanges(t, repo, noteGuide(t))
			sees(t, out, code, 0, "Stage 1 of 1:", allPassed(),
				"Stage 2 skipped: .fitnessrc.json sets scopedPullRequests, so only a push to main scans every file")
			if strings.Contains(out, boldFinding) || strings.Contains(out, "Stage 2 of") {
				t.Errorf("the run judged a file the pull request does not change:\n%s", out)
			}
		}},
		{"a pull request whose changed file fails", func(t *testing.T) {
			t.Parallel()
			out, code, _ := prChanges(t, prBase(t, nil), map[string]string{"README.md": boldReadme})
			sees(t, out, code, 1, "Stage 1 of 2:", boldFinding, "Stage 2 skipped: stage 1 failed")
			if strings.Contains(out, "Stage 2 of 2") {
				t.Errorf("stage 2 ran after stage 1 failed:\n%s", out)
			}
		}},
	}.run(t)
}

func Test0007_2(t *testing.T) {
	cases{
		{"sets the repo's `FITNESS_RUNS_ON` variable to a standard hosted macOS label such as `xcode-27` or `macos-26`", func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"xcode-27", "macos-26", "macos-15-intel"} {
				if got := labelSuite(t, value); got != value {
					t.Errorf("FITNESS_RUNS_ON=%s runs the suite on %q, want %q", value, got, value)
				}
			}
		}},
		{"leaves `FITNESS_RUNS_ON` unset, or sets it to a self-hosted, Linux or `-large` label", func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"", "fitness", "self-hosted", "ubuntu-24.04-arm", "xcode-27-xlarge", "macos-latest-large", "MACOS-26-LARGE", "my-macos-26"} {
				if got := labelSuite(t, value); got != "" {
					t.Errorf("FITNESS_RUNS_ON=%q runs the suite on %q, want the usual runner", value, got)
				}
			}
			if expr := runsOnSuite(t, reusableSuite); !strings.HasPrefix(expr, "inputs.runs-on || github.event.repository.private && ") {
				t.Errorf("ci-reusable does not fall back to its usual choice for an empty label: %q", expr)
			}
		}},
		{"the reusable workflows", func(t *testing.T) {
			t.Parallel()
			paths, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*-reusable.yml"))
			mustDo(t, err)
			commands := 0
			for _, path := range paths {
				text, err := os.ReadFile(path)
				mustDo(t, err)
				for name, job := range jobsCheckout(string(text)) {
					commands += strings.Count(job, "actions/fitness-command")
					if !checksOutFirst(job) {
						t.Errorf("%s job %s runs a fitness command without a checkout first, so it installs no pinned check", filepath.Base(path), name)
					}
				}
			}
			if commands == 0 {
				t.Fatal("no reusable workflow runs the fitness-command action, so the test proves nothing")
			}
		}},
		{"a merged fitness-runner pull request, from a review or comment, whose head `go install` cannot fetch", func(t *testing.T) {
			t.Parallel()
			for _, event := range []string{`{"pull_request":{"number":7}}`, `{"issue":{"number":7}}`} {
				out, code, rp, pathFile := installDeleted(t, "may-journal/fitness-runner", "deadbeef", event, goAnswersDeleted("deadbeef"), pullDeleted("true"))
				sees(t, out, code, 0, "Pull request 7 merged", "its head deadbeef is gone", "merge commit cafe1234")
				if !rp.called("go", "install", "@cafe1234") {
					t.Errorf("go never built the merge commit: %q", rp.calls("go"))
				}
				if path, _ := os.ReadFile(pathFile); !strings.Contains(string(path), "/gopath/bin") {
					t.Errorf("the build is not on PATH: %q", path)
				}
			}
		}},
		{"an open pull request, another ref, another repo or a failed lookup, and `go install` cannot fetch the ref", func(t *testing.T) {
			t.Parallel()
			cases := []struct {
				repo, ref, event string
				answers          []response
			}{
				{"may-journal/fitness-runner", "deadbeef", `{"pull_request":{"number":7}}`, pullDeleted("false")},
				{"may-journal/fitness-runner", "deadbeef", `{}`, nil},
				{"may-journal/fitness-runner", "abcdef12", `{"pull_request":{"number":7}}`, pullDeleted("true")},
				{"may-journal/other", "deadbeef", `{"pull_request":{"number":7}}`, pullDeleted("true")},
				{"may-journal/fitness-runner", "deadbeef", `{"pull_request":{"number":7}}`, []response{{Match: []string{"pulls/7"}, Exit: 1}}},
			}
			for _, c := range cases {
				out, code, rp, _ := installDeleted(t, c.repo, c.ref, c.event, goAnswersDeleted(c.ref), c.answers)
				sees(t, out, code, 1, "go install of "+c.ref+" failed", "no other commit stands in")
				if rp.count("go", "install") != 1 {
					t.Errorf("go built more than the requested ref: %q", rp.calls("go"))
				}
			}
		}},
		{"the fitness-install action", func(t *testing.T) {
			t.Parallel()
			keys := scriptCIFetch(t, "index-key", "RUNNER_OS=Linux", "RUNNER_ARCH=ARM64", "GITHUB_JOB=check")
			index := "fitness-checks-Linux-ARM64-" + indexHashCIFetch(t) + "-"
			want := map[string]string{"base": "fitness-checks-Linux-ARM64-", "index": index, "job": index + "check-"}
			if !maps.Equal(keys, want) {
				t.Errorf("the checks cache keys must hold the release index hash, want %q, got %q", want, keys)
			}
			text := actionCIFetch(t, "fitness-install")
			restore := stepCIFetch(t, text, "Restore the installed checks")
			for _, key := range []string{"job", "index", "base"} {
				if !hasLine(restore, "${{ steps.index.outputs."+key+" }}") || strings.Contains(text, ".fitnessrc.json')") {
					t.Errorf("the restore must fall back to the %s set, never key on .fitnessrc.json:\n%s", key, restore)
				}
			}
		}},
		{"the fitness-command action and this repo's CI workflow", func(t *testing.T) {
			t.Parallel()
			install := stepCIFetch(t, actionCIFetch(t, "fitness-command"), "Install fitness")
			if want := "with-checks: ${{ github.event_name == 'workflow_dispatch' && 'pinned' || 'on-demand' }}"; !strings.Contains(install, want) {
				t.Errorf("fitness-command must install on demand for events and every pinned check for a manual run, want %q:\n%s", want, install)
			}
			ci, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
			mustDo(t, err)
			suite := jobsCheckout(string(ci))["fitness"]
			if !strings.Contains(string(ci), "push:\n    branches: [main]") || !exportsChecksDir(t, suite) {
				t.Errorf("a push to main must install every check before the suite:\n%s", suite)
			}
			all, err := os.ReadFile(filepath.Join("..", "..", "githooks", "checks-dir"))
			mustDo(t, err)
			if !strings.Contains(string(all), "--with-checks --install-only") {
				t.Errorf("githooks/checks-dir must install every check:\n%s", all)
			}
		}},
		{"a checks cache holding a program the run linked and one it did not", func(t *testing.T) {
			t.Parallel()
			for _, c := range []struct {
				restored string
				kept     []string
			}{
				{"fitness-checks-Linux-ARM64-old-check-1", []string{"a-1"}},
				{"fitness-checks-Linux-ARM64-new-check-1", []string{"a-1", "b-1"}},
			} {
				home := cacheCIFetch(t)
				keys := scriptCIFetch(t, "set-key", "HOME="+home, "XDG_CACHE_HOME=", "FITNESS_INDEX_KEY=fitness-checks-Linux-ARM64-new-", "FITNESS_JOB_KEY=fitness-checks-Linux-ARM64-new-check-", "FITNESS_RESTORED_KEY="+c.restored)
				entries, err := os.ReadDir(filepath.Join(home, ".cache", "fitness", "checks"))
				mustDo(t, err)
				var kept []string
				for _, entry := range entries {
					kept = append(kept, entry.Name())
				}
				if !slices.Equal(kept, c.kept) || !strings.HasPrefix(keys["key"], "fitness-checks-Linux-ARM64-new-check-") {
					t.Errorf("restored from %s, the set must keep %q under the job's prefix, kept %q as %q", c.restored, c.kept, kept, keys["key"])
				}
			}
		}},
		{"an org whose private repo ran a 61 s job and two shorter jobs", func(t *testing.T) {
			t.Parallel()
			out, code, rp := minutesRun(t, []response{
				{Match: []string{"orgs/acme/repos?type=private"}, Stdout: "acme/app\n"},
				{Match: []string{"actions/runs/1/jobs"}, Stdout: actionsJob("success", "2026-10-20T10:00:00Z", "2026-10-20T10:00:08Z") +
					actionsJob("failure", "2026-10-20T10:01:00Z", "2026-10-20T10:01:12Z")},
				{Match: []string{"actions/runs/2/jobs"}, Stdout: actionsJob("success", "2026-10-20T11:00:00Z", "2026-10-20T11:01:01Z")},
				{Match: []string{"repos/acme/app/actions/runs?per_page=100&created=>="}, Stdout: "1\tComment check\n2\tFitness suite\n"},
			}, "--org", "acme", "--days", "4")
			sees(t, out, code, 0, "Actions jobs in 1 repos over the last 4 days",
				"| Comment check | 2 | 2 | 12 |", "| Fitness suite | 1 | 2 | 61 |", "| Total | 3 | 4 | |")
			if !rp.called("gh", "api", "repos/acme/app/actions/runs?per_page=100&created=>=") {
				t.Errorf("want the runs of acme/app read, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a run whose only job was skipped", func(t *testing.T) {
			t.Parallel()
			out, code, _ := minutesRun(t, []response{
				{Match: []string{"actions/runs/3/jobs"}, Stdout: actionsJob("skipped", "2026-10-20T12:00:00Z", "2026-10-20T12:00:00Z")},
				{Match: []string{"actions/runs/4/jobs"}, Stdout: actionsJob("success", "2026-10-20T12:00:00Z", "2026-10-20T12:00:09Z")},
				{Match: []string{"repos/acme/app/actions/runs?"}, Stdout: "3\tAuto-merge\n4\tPR check\n"},
			}, "--repos", "acme/app")
			sees(t, out, code, 0, "| PR check | 1 | 1 | 9 |", "| Total | 1 | 1 | |")
			if strings.Contains(out, "| Auto-merge |") {
				t.Errorf("want the skipped job left out, got:\n%s", out)
			}
		}},
	}.run(t)
}

func Test0007_3(t *testing.T) {
	cases{
		{"an issue closed as completed with an unchecked item", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeCheckClosed(t, closedClosed(t, unfinished, "completed"), []response{
				{Match: []string{"-X POST", "issues/21/comments"}, Stdout: `{"id":901,"body":"🔁 Reopened"}`},
				commentsClosed(""),
				{Match: []string{"graphql", "ClosedEvent"}, Stdout: `{"author":"","merger":""}` + "\n"},
				{Match: []string{"-X PATCH", "state=open", "repos/" + repoPR + "/issues/21"}, Stdout: `{"number":21,"state":"open"}`},
			})
			sees(t, out, code, 0, `unchecked item: "Write the tests."`)
			if !rp.called("gh", "-X PATCH", "state=open", "issues/21") {
				t.Errorf("want issue 21 reopened, got gh calls %q", rp.calls("gh"))
			}
			if !rp.called("gh", "-X POST", "issues/21/comments", `"Write the tests."`, "@closer-login") {
				t.Errorf("want one comment naming the item and @closer-login, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a close that close-check already handled, or one where only another author pasted its marker", func(t *testing.T) {
			t.Parallel()
			sum := sha256.Sum256([]byte(closedAtClosed + "\n" + unfinished))
			mark := "<!-- fitness:close-check:" + hex.EncodeToString(sum[:])[:12] + " -->"
			reply := func(author string) response {
				return commentsClosed(`{"author":"` + author + `","body":"🔁 Reopened: this issue was closed with unchecked checklist items.\n\n` + mark + `"}` + "\n")
			}
			rp, out, code := closeCheckClosed(t, closedClosed(t, unfinished, "completed"), []response{reply("github-actions[bot]")})
			sees(t, out, code, 0)
			if rp.called("gh", "-X POST") {
				t.Errorf("want no second comment, got gh calls %q", rp.calls("gh"))
			}
			rp, out, code = closeCheckClosed(t, closedClosed(t, unfinished, "completed"), []response{
				{Match: []string{"-X POST", "issues/21/comments"}, Stdout: `{"id":901,"body":"🔁 Reopened"}`},
				reply("dev"),
				{Match: []string{"graphql", "ClosedEvent"}, Stdout: `{"author":"","merger":""}` + "\n"},
				{Match: []string{"-X PATCH", "state=open", "repos/" + repoPR + "/issues/21"}, Stdout: `{"number":21,"state":"open"}`},
			})
			sees(t, out, code, 0)
			if !rp.called("gh", "-X PATCH", "state=open", "issues/21") {
				t.Errorf("want a pasted marker ignored and the issue reopened, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"an issue closed as not planned with unchecked items", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeCheckClosed(t, closedClosed(t, unfinished, "not_planned"), nil)
			sees(t, out, code, 0, "closed as not_planned")
			quietClosed(t, rp)
		}},
		{"an issue with no parent, closed with every item ticked", func(t *testing.T) {
			t.Parallel()
			done := "- [x] Write the doc.\n- [x] Write the tests.\n"
			rp, out, code := closeCheckClosed(t, closedClosed(t, done, "completed"), noParentParents(21))
			sees(t, out, code, 0, "Passed.")
			quietClosed(t, rp)
		}},
		{"a parent with another child still open", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", "> A parent.\n"), subIssuesParents(30, 21, 22),
				issueParents(t, 22, "open", "", ""),
			})
			sees(t, out, code, 0)
			stayedOpenReopen(t, rp)
		}},
		{"an issue that names no children", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, mentionedByParents(21, 30), []response{
				issueParents(t, 30, "open", "", "> Mentions #21 but lists no child.\n"), subIssuesParents(30),
			}, noParentParents(30))
			sees(t, out, code, 0)
			stayedOpenReopen(t, rp)
		}},
		{"a parent with a child closed as not planned", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", "> A parent.\n"), subIssuesParents(30, 21, 22),
				issueParents(t, 22, "closed", "not_planned", ""),
			})
			sees(t, out, code, 0)
			stayedOpenReopen(t, rp)
		}},
		{"a parent with an unchecked box of its own", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", listParents(21)+"- [ ] Announce it.\n"), subIssuesParents(30, 21),
			})
			sees(t, out, code, 0)
			stayedOpenReopen(t, rp)
		}},
		{"a closed parent and one of its children reopens", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "closed", "completed", "> A parent.\n"),
			})
			sees(t, out, code, 0)
			stayedOpenReopen(t, rp)
		}},
	}.run(t)
}

func Test0007_4(t *testing.T) {
	cases{
		{"a parent whose only sub-issue is the issue just closed as completed", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", "> A parent.\n"), subIssuesParents(30, 21),
			}, writesParents(30), noParentParents(30))
			sees(t, out, code, 0)
			closedParentParents(t, rp, 30)
		}},
		{"a parent that lists the closed issue as its last reference in the list", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, mentionedByParents(21, 30), []response{
				issueParents(t, 30, "open", "", listParents(20, 21)), subIssuesParents(30),
				issueParents(t, 20, "closed", "completed", ""),
			}, writesParents(30), noParentParents(30))
			sees(t, out, code, 0)
			closedParentParents(t, rp, 30)
		}},
		// This case expects one comment on the parent that names the child once.
		{"a parent that lists its last child both as a sub-issue and in the list", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", listParents(21)), subIssuesParents(30, 21),
			}, writesParents(30), noParentParents(30))
			sees(t, out, code, 0)
			closedParentParents(t, rp, 30)
			if c := rp.count("gh", "-X POST", issuePathParents(30)+"/comments", repoPR+"#21 closed as completed"); c != 1 {
				t.Errorf("want the comment to name child %s#21 and its close reason once, got gh calls %q", repoPR, rp.calls("gh"))
			}
			if n := strings.Count(strings.Join(rp.calls("gh"), "\n"), repoPR+"#21 closed"); n != 1 {
				t.Errorf("want child 21 named once in the comment, got %d in gh calls %q", n, rp.calls("gh"))
			}
		}},
		{"a parent whose own parent has no other open child", func(t *testing.T) {
			t.Parallel()
			rp, out, code := closeParentsParents(t, parentOfParents(21, 30), []response{
				issueParents(t, 30, "open", "", "> A parent.\n"), subIssuesParents(30, 21),
				issueParents(t, 40, "open", "", "> A grandparent.\n"), subIssuesParents(40, 30),
			}, writesParents(30), parentOfParents(30, 40), writesParents(40), noParentParents(40))
			sees(t, out, code, 0)
			closedParentParents(t, rp, 30)
			closedParentParents(t, rp, 40)
		}},
	}.run(t)
}

func Test0007_5(t *testing.T) {
	cases{
		{"a new issue that repeats an older one and another unlike any", func(t *testing.T) {
			t.Parallel()
			issues := []string{listedDupe(12, offlineTitle, offlineBody, 0, ""), listedDupe(21, repeatTitle, repeatBody, 0, ""), listedDupe(22, darkTitle, darkBody, 0, "")}
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
		}},
		{"an issue whose duplicate note dupe-check already posted", func(t *testing.T) {
			t.Parallel()
			note := `{"id":77,"body":"🔁 This issue may repeat an earlier one.\n\n- #12 (score 0.50)\n\n<!-- fitness:dupe-check:000000000000 -->"}` + "\n"
			issues := []string{listedDupe(12, offlineTitle, offlineBody, 0, ""), listedDupe(21, repeatTitle, repeatBody, 0, "")}
			out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("edited", 21, repeatTitle, repeatBody), nil, dupeGitHub(21, note, issues...))
			sees(t, out, code, 0, `\#21 may repeat \#12`)
			if !rp.called("gh", "-X PATCH", "issues/comments/77", "#12 (score") || rp.called("gh", "-X POST") {
				t.Errorf("want note 77 edited in place and no second note, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"an issue that repeats its parent, a sibling sub-issue, and an issue it names", func(t *testing.T) {
			t.Parallel()
			named := repeatBody + "\nFollows #9.\n"
			issues := []string{
				listedDupe(9, offlineTitle, offlineBody, 0, ""), listedDupe(12, offlineTitle, offlineBody, 0, ""),
				listedDupe(20, repeatTitle, repeatBody, 12, ""), listedDupe(21, repeatTitle, named, 12, ""),
			}
			out, code, rp := ghRun(t, "dupe-check", "issues", dupeEvent("opened", 21, repeatTitle, named), nil, dupeGitHub(21, "", issues...))
			sees(t, out, code, 0, `\#21 resembles no earlier issue.`)
			if wrote(rp) {
				t.Errorf("want nothing written on #21, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a close tier on in `.fitnessrc.json`, and a new issue above the close score against an open one", func(t *testing.T) {
			t.Parallel()
			config := map[string]string{".fitnessrc.json": `{"dupeCheck": {"closeScore": 0.75}}`}
			issues := []string{listedDupe(12, offlineTitle, offlineBody, 0, ""), listedDupe(21, offlineTitle, offlineBody, 0, "")}
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
		}},
		{"issues that repeat each other, and one closed before the window", func(t *testing.T) {
			t.Parallel()
			old := time.Now().UTC().AddDate(0, 0, -200).Format(time.RFC3339)
			issues := []string{
				listedDupe(5, offlineTitle, offlineBody, 0, old), listedDupe(12, offlineTitle, offlineBody, 0, ""),
				listedDupe(21, repeatTitle, repeatBody, 0, ""), listedDupe(22, darkTitle, darkBody, 0, ""),
			}
			out, code, rp := ghRun(t, "dupe-check", "workflow_dispatch", map[string]any{}, nil, dupeGitHub(21, "", issues...))
			sees(t, out, code, 0, "3 issues;", "| #21 | #12 |", "| warn |")
			if strings.Contains(out, "#5 |") || strings.Contains(out, "| #22 |") {
				t.Errorf("want only the pair in the window scoring near the warn score, got:\n%s", out)
			}
			if wrote(rp) {
				t.Errorf("want nothing written, got gh calls %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

func Test0007_6(t *testing.T) {
	cases{
		{"a pull request on another pull request's branch, with no native stack", func(t *testing.T) {
			t.Parallel()
			rp, out, code := autoMergeAutoMerge(t, "photos-grid", response{}, "")
			sees(t, out, code, 0, "::notice::This PR targets photos-grid, not main", "GitHub has none for branch chains", "Its auto-merge comment says how to merge it.")
			if rp.count("gh", "pr merge")+rp.count("gh", "merge-async") > 0 {
				t.Fatalf("a PR in a branch chain asked GitHub to merge: %q", rp.calls("gh"))
			}
			if !rp.called("gh", "api "+issuesAutoMerge+"/2/comments", loggedAutoMerge(stackedAutoMerge)) {
				t.Fatalf("a stacked PR got no note saying why it has no auto-merge: %q", rp.calls("gh"))
			}
			rp, out, code = autoMergeAutoMerge(t, "photos-grid", response{}, stackedAutoMerge+"\n\n<!-- fitness:auto-merge -->")
			sees(t, out, code, 0)
			if postedAutoMerge(rp) != "" || rp.called("gh", "-X PATCH") {
				t.Fatalf("a second push to a stacked PR wrote its note again: %q", rp.calls("gh"))
			}
			refused := response{Match: []string{"-f body="}, Stderr: "gh: Resource not accessible by integration (HTTP 403)\n", Exit: 1}
			_, out, code = autoMergeAutoMerge(t, "photos-grid", response{}, "", refused)
			sees(t, out, code, 0, "HTTP 403", "::warning::GitHub refused a change to a PR's auto-merge comment")
		}},
		{"a stacked pull request that GitHub moved onto the default branch after its base merged", func(t *testing.T) {
			t.Parallel()
			rp, out, code := autoMergeAutoMerge(t, "main", response{}, stackedAutoMerge+"\n\n<!-- fitness:auto-merge -->")
			sees(t, out, code, 0)
			if !rp.called("gh", "pr merge --auto --squash https://github.com/"+repoPR+"/pull/2") {
				t.Fatalf("a PR moved onto main did not get auto-merge: %q", rp.calls("gh"))
			}
			on := "Auto-merge is on: this PR squash-merges into `main` once it is approved and green."
			if !rp.called("gh", "-X PATCH "+issuesAutoMerge+"/comments/7", loggedAutoMerge(on)) {
				t.Fatalf("the note on a PR moved onto main was not updated: %q", rp.calls("gh"))
			}
			rp, out, code = autoMergeAutoMerge(t, "main", response{}, "")
			sees(t, out, code, 0)
			if postedAutoMerge(rp) != "" || rp.called("gh", "-X PATCH") {
				t.Fatalf("a PR that never had a note got one: %q", rp.calls("gh"))
			}
		}},
		{"a pull request on the default branch that GitHub still counts as stacked", func(t *testing.T) {
			t.Parallel()
			refusal := "GraphQL: Auto-merge is not supported for stacked pull requests. (enablePullRequestAutoMerge)\n"
			rp, out, code := autoMergeAutoMerge(t, "main", response{Stderr: refusal, Exit: 1}, "")
			sees(t, out, code, 0, strings.TrimSpace(refusal), "::notice::GitHub still counts this PR as stacked, so it gets no auto-merge.")
			note := "Auto-merge is off: this PR targets `main`, but GitHub still counts it as part of a stack and refuses auto-merge. " +
				"Merge it by hand once it is approved and green, or push to it once GitHub moves it out of the stack."
			if !rp.called("gh", "api "+issuesAutoMerge+"/2/comments", loggedAutoMerge(note)) {
				t.Fatalf("a PR GitHub counts as stacked got no note: %q", rp.calls("gh"))
			}
		}},
		{"a pull request on the default branch", func(t *testing.T) {
			t.Parallel()
			refusal := "GraphQL: Pull request is in clean status (enablePullRequestAutoMerge)\n"
			rp, out, code := autoMergeAutoMerge(t, "main", response{Stderr: refusal, Exit: 1}, "")
			sees(t, out, code, 1, strings.TrimSpace(refusal), "::error::GitHub refused auto-merge for this PR: "+strings.TrimSpace(refusal))
			if strings.Contains(out, "::notice::") || rp.called("gh", "comments") {
				t.Fatalf("another auto-merge failure passed with a notice or a note:\n%s\n%q", out, rp.calls("gh"))
			}
		}},
		{"a ready pull request", func(t *testing.T) {
			t.Parallel()
			pr := func(action, autoMerge string) map[string]string {
				return map[string]string{"action": action, "pull_request.state": "open", "pull_request.draft": "false", "pull_request.auto_merge": autoMerge}
			}
			if !runsAutoMerge(t, pr("reopened", "on")) {
				t.Error("a reopened PR whose event still shows auto-merge skips the job, want it run")
			}
			if !runsAutoMerge(t, pr("reopened", "")) {
				t.Error("a reopened PR without auto-merge skips the job, want it run")
			}
			if runsAutoMerge(t, pr("synchronize", "on")) {
				t.Error("a push to a PR with auto-merge on runs the job, want it skipped")
			}
			if !runsAutoMerge(t, pr("synchronize", "")) {
				t.Error("a push to a PR without auto-merge skips the job, want it run")
			}
		}},
		{"a native stack whose pull requests are all approved, green and mergeable", func(t *testing.T) {
			t.Parallel()
			merged := response{Stdout: `{"status":"merged","details":{"sha":"ccc"}}` + "\n"}
			rp, out, code := stackStack(t, approvedStack, pendingStack, merged)
			sees(t, out, code, 0, "::notice::GitHub merged stack #7 through #3.")
			if !rp.called("gh", "-X PUT "+pullsStack+"/3/merge-async", "merge_method=squash", "sha=bbb", "bypass_rules=false") {
				t.Fatalf("an approved stack did not ask GitHub to merge its top PR at its head without bypassing rules: %q", rp.calls("gh"))
			}
			if rp.count("gh", "pr merge")+rp.count("gh", pullsStack+"/2/merge-async") > 0 {
				t.Fatalf("an approved stack asked GitHub to merge a PR other than its top: %q", rp.calls("gh"))
			}
			notedAllStack(t, rp, "Auto-merge merged stack #7 (#2 #3) into `main` through #3.")
		}},
		{"a native stack with a pull request that is not approved", func(t *testing.T) {
			t.Parallel()
			views := []response{viewStack(2, "APPROVED", "SUCCESS"), viewStack(3, "REVIEW_REQUIRED", "SUCCESS")}
			rp, out, code := stackStack(t, views, response{}, response{})
			waitsStack(t, rp, out, code, "#3 needs approval")
		}},
		{"a native stack with a pull request whose checks are red", func(t *testing.T) {
			t.Parallel()
			views := []response{viewStack(2, "APPROVED", "FAILURE"), viewStack(3, "APPROVED", "SUCCESS")}
			rp, out, code := stackStack(t, views, response{}, response{})
			waitsStack(t, rp, out, code, "#2 is not green")
		}},
		{"an approved, green native stack", func(t *testing.T) {
			t.Parallel()
			refusal := response{Stdout: failedStack, Stderr: "gh: Required workflows 'Fitness suite' failed (HTTP 400)\n", Exit: 1}
			rp, out, code := stackStack(t, approvedStack, refusal, response{})
			refusedStack(t, rp, out, code, "refused to merge",
				"Auto-merge could not merge stack #7 (#2 #3): GitHub refused, Required workflows 'Fitness suite' failed.")
			if rp.called("gh", pullsStack+"/3/merge-async/") {
				t.Fatalf("a refused merge was polled: %q", rp.calls("gh"))
			}
		}},
		{"an approved, green native stack: it fails naming GitHub's answer and comments the reason on the top pull request", func(t *testing.T) {
			t.Parallel()
			rp, out, code := stackStack(t, approvedStack, pendingStack, response{Stdout: failedStack})
			refusedStack(t, rp, out, code, "failed to merge",
				"Auto-merge could not merge stack #7 (#2 #3): GitHub failed, Required workflows 'Fitness suite' failed.")
		}},
		{"a pull request", func(t *testing.T) {
			t.Parallel()
			cond := callerIfStack(t)
			cases := []struct {
				name  string
				event map[string]string
				want  bool
			}{
				{"an approval", reviewStack("approved", repoPR), true},
				{"an approval of a fork's PR", reviewStack("approved", "someone/example"), false},
				{"a comment review", reviewStack("commented", repoPR), false},
				{"a review requesting changes", reviewStack("changes_requested", repoPR), false},
				{"a draft marked ready", map[string]string{"event_name": "pull_request_target", "review.state": "", "pull_request.head.repo.full_name": repoPR}, true},
			}
			for _, c := range cases {
				if holdsAutoMerge(t, cond, c.event) != c.want {
					t.Errorf("%s: caller runs = %v, want %v", c.name, !c.want, c.want)
				}
			}
			approval := map[string]string{"action": "submitted", "pull_request.state": "open", "pull_request.draft": "false", "pull_request.auto_merge": ""}
			if !runsAutoMerge(t, approval) {
				t.Error("an approval of a stacked PR skips the Auto-merge job, want it run")
			}
		}},
	}.run(t)
}

func Test0007_7(t *testing.T) {
	cases{
		{"a release pull request by the automation App that changes only `CHANGELOG.md`, `version.txt` and the manifest, with green checks", func(t *testing.T) {
			t.Parallel()
			_, out, code, written := guardReleaseApprove(t, viewReleaseApprove(authorReleaseApprove), releaseFilesReleaseApprove)
			sees(t, out, code, 0)
			if written != "sha=abc123\n" {
				t.Fatalf("a release PR of release files alone was not passed on: output %q\n%s", written, out)
			}
			_, out, code, written = waitReleaseApprove(t, checksReleaseApprove(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"skipping","link":"https://github.com/may-journal/example/actions/runs/12/job/2"}]`, 0), headReleaseApprove("abc123"))
			sees(t, out, code, 0)
			if written != "green=true\n" {
				t.Fatalf("green checks were not reported green: output %q\n%s", written, out)
			}
			rp, out, code, _ := runReleaseApprove(t, "Approve the PR", reviewsReleaseApprove(`[]`))
			sees(t, out, code, 0, "Approved PR #23 at abc123.")
			if !rp.called("gh", "-X POST repos/"+repoPR+"/pulls/23/reviews -f event=APPROVE -f commit_id=abc123") {
				t.Fatalf("the release PR was not approved at the checked commit: %q", rp.calls("gh"))
			}
			rp, out, code, _ = runReleaseApprove(t, "Approve the PR", reviewsReleaseApprove(`[]`+"\n"+approvedReleaseApprove))
			sees(t, out, code, 0, "PR #23 is already approved at abc123.")
			if rp.called("gh", "-X POST") {
				t.Fatalf("a release PR approved at abc123 was approved again: %q", rp.calls("gh"))
			}
			tokensReleaseApprove(t)
		}},
		{"a release pull request that also changes a workflow, or renames one into a release file", func(t *testing.T) {
			t.Parallel()
			for name, files := range map[string]response{
				".github/workflows/ci.yml": filesReleaseApprove(`[{"filename":"CHANGELOG.md"},{"filename":".github/workflows/ci.yml"}]`),
				".github/release.sh":       filesReleaseApprove(`[{"filename":"version.txt","previous_filename":".github/release.sh"}]`),
			} {
				rp, out, code, written := guardReleaseApprove(t, viewReleaseApprove(authorReleaseApprove), files)
				sees(t, out, code, 0, "::notice::PR #23 changes more than the release files, so a person approves it: "+name)
				if written != "" || rp.called("gh", "pr checks") || rp.called("gh", "reviews") {
					t.Errorf("a release PR changing %s went on toward approval: output %q, gh calls %q", name, written, rp.calls("gh"))
				}
			}
		}},
		{"a pull request by someone other than the automation App", func(t *testing.T) {
			t.Parallel()
			rp, out, code, written := guardReleaseApprove(t, viewReleaseApprove("json-claude-bot"), releaseFilesReleaseApprove)
			sees(t, out, code, 0, "::notice::PR #23 is by json-claude-bot, not "+authorReleaseApprove+", so a person approves it.")
			if written != "" || rp.called("gh", "pulls/23/files") {
				t.Errorf("a PR by another author went on toward approval: output %q, gh calls %q", written, rp.calls("gh"))
			}
		}},
		{"a release pull request with a failed check, then one whose checks never finish", func(t *testing.T) {
			t.Parallel()
			_, out, code, written := waitReleaseApprove(t, checksReleaseApprove(`[{"bucket":"fail","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/13/job/3"}]`, 1), headReleaseApprove("abc123"))
			sees(t, out, code, 0, "::notice::PR #23 has a failed or cancelled check, so it gets no approval.")
			if written != "" {
				t.Errorf("a release PR with a failed check was reported green: %q", written)
			}
			rp, out, code, written := waitReleaseApprove(t, checksReleaseApprove(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/13/job/3"}]`, 8), headReleaseApprove("abc123"))
			sees(t, out, code, 0, "::warning::PR #23 still had checks running after 2 tries, so a person approves it.")
			if written != "" || rp.count("gh", "pr checks") != 2 {
				t.Errorf("unfinished checks were not read once per try, or were reported green: output %q, gh calls %q", written, rp.calls("gh"))
			}
			none := response{Match: []string{"pr checks 23"}, Stderr: "no checks reported on the 'release-please--branches--main' branch\n", Exit: 1}
			_, out, code, written = waitReleaseApprove(t, none, headReleaseApprove("abc123"))
			sees(t, out, code, 0, "::warning::PR #23 still had checks running after 2 tries")
			if written != "" {
				t.Errorf("a release PR with no checks yet was reported green: %q", written)
			}
		}},
		{"a release pull request whose only running check is this run's own, then one pushed to while its checks ran", func(t *testing.T) {
			t.Parallel()
			own := checksReleaseApprove(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/99/job/4"},{"bucket":"pass","link":null}]`, 8)
			_, out, code, written := waitReleaseApprove(t, own, headReleaseApprove("abc123"))
			sees(t, out, code, 0)
			if written != "green=true\n" {
				t.Errorf("Release approve's own running check held up the approval: output %q\n%s", written, out)
			}
			_, out, code, written = waitReleaseApprove(t, own, headReleaseApprove("def456"))
			sees(t, out, code, 0, "::notice::PR #23 moved to def456 while its checks ran, so this run approves nothing.")
			if written != "" {
				t.Errorf("a release PR pushed to while its checks ran was reported green: %q", written)
			}
		}},
		{"a repo without the approver token, then ones whose token is rejected or cannot see the repo", func(t *testing.T) {
			t.Parallel()
			_, out, code, written := runReleaseApprove(t, "Check the approver token", nil, "TOKEN=")
			sees(t, out, code, 0, "::notice::APPROVER_TOKEN is not available to this repo, so a person approves the PR.")
			if written != "" {
				t.Errorf("a missing token was reported ready: %q", written)
			}
			skipsAllButReportReleaseApprove(t)
			user := response{Match: []string{"api user --jq .login"}, Stdout: "may-journal-approver\n", Token: "token"}
			repo := response{Match: []string{"api repos/" + repoPR + " --silent"}, Token: "token"}
			_, _, _, written = runReleaseApprove(t, "Check the approver token", []response{user, repo}, "TOKEN=token")
			if written != "ready=true\nlogin=may-journal-approver\n" {
				t.Errorf("a present token was not reported ready: %q", written)
			}
			for stderr, want := range map[string]string{
				"HTTP 401: Bad credentials\n": "::error::GitHub rejects APPROVER_TOKEN; replace the org secret with a valid token.",
				"HTTP 404: Not Found\n":       "::error::APPROVER_TOKEN cannot see " + repoPR + "; give the token access to it.",
				"HTTP 502: Bad Gateway\n":     "::error::Could not check APPROVER_TOKEN: HTTP 502: Bad Gateway",
			} {
				failing := repo
				failing.Exit, failing.Stderr = 1, stderr
				_, out, code, written = runReleaseApprove(t, "Check the approver token", []response{user, failing}, "TOKEN=token")
				sees(t, out, code, 1, want)
				if written != "" {
					t.Errorf("a token check failing with %q was reported ready: %q", stderr, written)
				}
			}
		}},
		{"a `build(deps)` bump of a dev dependency's minor version, a lockfile-only one, then one with a legacy lockfile tree", func(t *testing.T) {
			t.Parallel()
			types := map[string]any{"node_modules/@typescript-eslint/types": installedDepApprove("8.72.0", true)}
			_, out, code, written := bumpDepApprove(t, titleDepApprove, bothDepApprove, atDepApprove("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", types))
			sees(t, out, code, 0)
			if written != "sha=abc123\n" {
				t.Errorf("a minor dev-dependency bump was not passed on: output %q\n%s", written, out)
			}
			lockOnly := filesReleaseApprove(`[{"filename":"package-lock.json"}]`)
			_, out, code, written = bumpDepApprove(t, "build(deps): update rollup to 4.64.9", lockOnly, atDepApprove("^3.23.0", "^8.71.2", "3.23.8", "8.71.4", nil))
			sees(t, out, code, 0)
			if written != "sha=abc123\n" {
				t.Errorf("a lockfile-only dev bump was not passed on: output %q\n%s", written, out)
			}
			legacy := atDepApprove("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", nil)
			legacy.lock["dependencies"] = map[string]any{"@typescript-eslint/parser": installedDepApprove("8.72.0", true)}
			_, out, code, written = bumpDepApprove(t, titleDepApprove, bothDepApprove, legacy)
			sees(t, out, code, 0)
			if written != "sha=abc123\n" {
				t.Errorf("a bump in a lockfile with the legacy dependencies tree was not passed on: output %q\n%s", written, out)
			}
		}},
		{"a `build(deps)` bump changing a runtime dependency, a runtime package, a download URL, or a package in place", func(t *testing.T) {
			t.Parallel()
			for want, head := range map[string]sideDepApprove{
				"package.json changes more than devDependencies": atDepApprove("^3.24.0", "^8.71.2", "3.24.1", "8.71.2", nil),
				"node_modules/zod is not a dev-only package":     atDepApprove("^3.23.0", "^8.71.2", "3.23.9", "8.71.2", nil),
				"node_modules/@typescript-eslint/parser resolves outside the npm registry": atDepApprove("^3.23.0", "^8.72.0", "3.23.8", "8.72.0",
					map[string]any{"node_modules/@typescript-eslint/parser": map[string]any{"version": "8.72.0", "dev": true, "resolved": "https://example.com/parser.tgz"}}),
				"node_modules/@typescript-eslint/parser changes without a new version": atDepApprove("^3.23.0", "^8.71.2", "3.23.8", "8.71.2",
					map[string]any{"node_modules/@typescript-eslint/parser": map[string]any{"version": "8.71.2", "dev": true, "integrity": "sha512-other"}}),
			} {
				_, out, code, written := bumpDepApprove(t, titleDepApprove, bothDepApprove, head)
				sees(t, out, code, 0, "::notice::PR #23 is not a patch or minor dev-dependency bump, so a person approves it:", want)
				if written != "" {
					t.Errorf("a runtime dependency change was passed on (%s): output %q", want, written)
				}
			}
		}},
		{"a `build(deps)` pull request bumping a dev dependency's major version, then a 0.x minor version", func(t *testing.T) {
			t.Parallel()
			_, out, code, written := bumpDepApprove(t, titleDepApprove, bothDepApprove, atDepApprove("^3.23.0", "^9.0.0", "3.23.8", "9.0.0", nil))
			sees(t, out, code, 0, "@typescript-eslint/parser moves from ^8.71.2 to ^9.0.0, not a patch or minor bump",
				"node_modules/@typescript-eslint/parser moves from 8.71.2 to 9.0.0, not a patch or minor bump")
			if written != "" {
				t.Errorf("a major dev-dependency bump was passed on: output %q", written)
			}
			tiny := map[string]any{"node_modules/tiny": installedDepApprove("0.4.0", true)}
			_, out, code, written = bumpDepApprove(t, titleDepApprove, bothDepApprove, atDepApprove("^3.23.0", "^8.71.2", "3.23.8", "8.71.2", tiny))
			sees(t, out, code, 0, "node_modules/tiny moves from 0.3.5 to 0.4.0, not a patch or minor bump")
			if written != "" {
				t.Errorf("a 0.x minor bump, which semver counts as breaking, was passed on: output %q", written)
			}
		}},
		{"a `build(deps)` pull request adding the lockfile its base lacks", func(t *testing.T) {
			t.Parallel()
			head := atDepApprove("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", nil)
			missing := response{Match: []string{"contents/package-lock.json?ref=base000"}, Stdout: `{"message":"Not Found","status":"404"}`, Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1}
			_, out, code, written := runReleaseApprove(t, "Check the dependency bump", []response{
				{Match: []string{"pr view 23 --json title"}, Stdout: `{"title":"` + titleDepApprove + `","state":"OPEN","isDraft":false,"headRefOid":"abc123","baseRefOid":"tip000"}` + "\n"},
				bothDepApprove, {Match: []string{"compare/tip000...abc123"}, Stdout: "base000\n"}, missing,
				contentsDepApprove(t, "package.json", "base000", baseDepApprove.manifest),
				contentsDepApprove(t, "package.json", "abc123", head.manifest), contentsDepApprove(t, "package-lock.json", "abc123", head.lock),
			})
			sees(t, out, code, 0, "package-lock.json changes more than its packages")
			if strings.Contains(out, "message") || written != "" {
				t.Errorf("a missing base lockfile must read as empty and leave the PR to a person: output %q, written %q", out, written)
			}
		}},
		{"a pull request not titled `build(deps)`, then a `build(deps)` one that also changes a workflow", func(t *testing.T) {
			t.Parallel()
			head := atDepApprove("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", nil)
			rp, out, code, written := bumpDepApprove(t, "feat: add a grid", bothDepApprove, head)
			sees(t, out, code, 0, "::notice::PR #23 is not titled build(deps), so a person approves it.")
			if written != "" || rp.called("gh", "contents/") {
				t.Errorf("a PR not titled build(deps) went on: output %q, gh calls %q", written, rp.calls("gh"))
			}
			workflow := filesReleaseApprove(`[{"filename":"package.json"},{"filename":".github/workflows/ci.yml"}]`)
			rp, out, code, written = bumpDepApprove(t, titleDepApprove, workflow, head)
			sees(t, out, code, 0, "::notice::PR #23 changes more than package.json and package-lock.json, so a person approves it: .github/workflows/ci.yml")
			if written != "" || rp.called("gh", "contents/") {
				t.Errorf("a bump that also changes a workflow went on: output %q, gh calls %q", written, rp.calls("gh"))
			}
		}},
		{"a caller passing kind `dependency`, one passing no kind, then one misspelling it", func(t *testing.T) {
			t.Parallel()
			_, out, code, _ := runReleaseApprove(t, "Check the approver token", nil, "KIND=dependencies", "TOKEN=token")
			sees(t, out, code, 1, "::error::Release approve got kind 'dependencies'; pass release or dependency.")
			// Actions fills in the input's default when a caller passes none.
			for kind, want := range map[string]string{"dependency": "Check the dependency bump", defaultDepApprove(t): "Check the release PR"} {
				for _, step := range []string{"Check the dependency bump", "Check the release PR"} {
					if got := runsReleaseApprove(t, step, map[string]string{"secrets.ready": "true", "inputs.kind": kind}); got != (step == want) {
						t.Errorf("with kind %q, step %q runs = %v", kind, step, got)
					}
				}
			}
		}},
	}.run(t)
}

func Test0007_8(t *testing.T) {
	cases{
		{"a recorded Dependabot payload whose title is not a semantic commit", func(t *testing.T) {
			t.Parallel()
			got, _ := completeDependabot(t, payloadDependabot(t, "dependabot[bot]"), "")
			if !got.issueMade || !strings.HasPrefix(got.title, "build(deps): bump github.com/spf13/cobra") {
				t.Fatalf("want a new issue and a semantic title, got %+v", got)
			}
			wants := map[string]string{"closes the issue": "Closes #31", "keeps the notes": "Fix a crash when a flag is missing"}
			for what, want := range wants {
				if !strings.Contains(got.body, want) {
					t.Errorf("want the description to hold %q (%s):\n%s", want, what, got.body)
				}
			}
			out, code := judgeDependabot(t, got)
			sees(t, out, code, 0, "pr-check #7", "Passed.")
		}},
		{"an open issue for the same bump", func(t *testing.T) {
			t.Parallel()
			got, _ := completeDependabot(t, payloadDependabot(t, "dependabot[bot]"), `{"number": 31, "title": "Bump github.com/spf13/cobra"}`+"\n")
			if got.issueMade || !strings.Contains(got.body, "Closes #31") {
				t.Errorf("want the open bump issue reused, got %+v", got)
			}
		}},
		{"a pull request that a person opened", func(t *testing.T) {
			t.Parallel()
			_, rp := completeDependabot(t, payloadDependabot(t, "jason"), "")
			if rp.called("gh", "-X") {
				t.Errorf("want a person's PR left alone, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a Dependabot pull request that is already complete: it completes the title, then the body, then the changelog entry", func(t *testing.T) {
			t.Parallel()
			got, _ := completeDependabot(t, payloadDependabot(t, "dependabot[bot]"), "")
			pr := payloadDependabot(t, "dependabot[bot]")["pull_request"].(map[string]any)
			pr["title"], pr["body"] = got.title, got.body
			entry, err := json.Marshal(map[string]string{"filename": got.file, "status": "added", "patch": ""})
			mustDo(t, err)
			rp := standIn(t, map[string][]response{"gh": {{Match: []string{"pulls/7/files"}, Stdout: string(entry) + "\n"}}})
			env := actionsEvent(t, "pull_request_target", map[string]any{"action": "synchronize", "pull_request": pr})
			out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "dependabot-pr")
			sees(t, out, code, 0)
			if rp.called("gh", "-X") {
				t.Errorf("want a completed PR left alone, got gh calls %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

// ciStepCases run each subcommand a CI workflow or shared action step calls,
// the way the step calls it, for 0008.7.
func ciStepCases() cases {
	return cases{
		{"a repo with Go modules at several depths", func(t *testing.T) {
			t.Parallel()
			repo := t.TempDir()
			commit(t, repo, map[string]string{"z/go.mod": "module z\n", "a/go.mod": "module a\n", "b/c/go.mod": "module c\n"}, "build(go): add modules")
			run := runFitnessCI(t, ciStep{file: "ci-reusable.yml", command: "find-go-module", dir: repo})
			sees(t, run.out, run.code, 0)
			if run.outputs != "file=a/go.mod\n" {
				t.Errorf("want the shallowest go.mod, first by name, got %q", run.outputs)
			}
		}},
		{"a repo pinning Java only in .sdkmanrc", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "ci-reusable.yml", command: "find-jdk-version-file",
				files: map[string]string{".tool-versions": "nodejs 22\n", ".sdkmanrc": "gradle=8\njava=21.0.2-tem\n"}})
			sees(t, run.out, run.code, 0)
			if run.outputs != "file=.sdkmanrc\n" {
				t.Errorf("want the .sdkmanrc with a java line, got %q", run.outputs)
			}
		}},
		{"a cached SwiftLint zip that matches its published SHA-256", func(t *testing.T) {
			t.Parallel()
			home, temp := t.TempDir(), t.TempDir()
			write(t, home, map[string]string{".cache/fitness-swiftlint/swiftlint_linux_arm64-0.65.1.zip": "zip"})
			sum := sha256.Sum256([]byte("zip"))
			unzipped := filepath.Join(temp, "swiftlint")
			run := runFitnessCI(t, ciStep{file: "ci-reusable.yml", command: "install-swiftlint",
				env: []string{"HOME=" + home, "RUNNER_TEMP=" + temp, "VERSION=0.65.1", "ASSET=swiftlint_linux_arm64"},
				saved: map[string][]response{
					"gh":    {{Match: []string{"repos/realm/SwiftLint/releases/tags/0.65.1"}, Stdout: "sha256:" + hex.EncodeToString(sum[:]) + "\n"}},
					"unzip": {{Match: []string{"swiftlint_linux_arm64-0.65.1.zip -d " + unzipped}, Files: map[string]string{filepath.Join(unzipped, "swiftlint"): tool("swiftlint")}}},
					"curl":  nil,
				}})
			sees(t, run.out, run.code, 0, "swiftlint version")
			if run.path != unzipped+"\n" || run.rp.called("curl") {
				t.Errorf("a cached zip that matches must not download again, and later steps find swiftlint; PATH file %q, curl calls %q", run.path, run.rp.calls("curl"))
			}
		}},
		{"an issue-check caller passing a command it does not run", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "issue-check-reusable.yml", command: "check-command-input", env: []string{"FITNESS_COMMAND=dupe-check"}})
			sees(t, run.out, run.code, 1, "::error::command must be issue-check or plan-check, not dupe-check")
			run = runFitnessCI(t, ciStep{file: "issue-check-reusable.yml", command: "check-command-input", env: []string{"FITNESS_COMMAND=plan-check"}})
			sees(t, run.out, run.code, 0)
		}},
		{"a repo whose App key secret did not arrive", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "app-token", command: "check-secrets", env: []string{"APP_ID=1", "APP_KEY="}})
			sees(t, run.out, run.code, 1, "::error::AUTOMATION_APP_KEY is not available to this repo")
		}},
		{"fitness commands with a typo, then two where the first fails", func(t *testing.T) {
			t.Parallel()
			saved := map[string][]response{"fitness": {{Match: []string{"pr-check"}, Exit: 1}, {Match: []string{"dupe-check"}}}}
			run := runFitnessCI(t, ciStep{file: "fitness-command", command: "run-commands", env: []string{"FITNESS_COMMANDS=pr-check Dupe_Check"}, saved: saved})
			sees(t, run.out, run.code, 1, "::error::The fitness-command action got Dupe_Check, which is not a fitness command name.")
			if run.rp.called("fitness") || run.env != "FITNESS_FAILURE_REPORTED=true\n" {
				t.Errorf("a typo must run nothing and mark the failure reported; calls %q, env %q", run.rp.calls("fitness"), run.env)
			}
			run = runFitnessCI(t, ciStep{file: "fitness-command", command: "run-commands", env: []string{"FITNESS_COMMANDS=pr-check dupe-check"}, saved: saved})
			sees(t, run.out, run.code, 1)
			if got := run.rp.calls("fitness"); !slices.Equal(got, []string{"pr-check", "dupe-check"}) {
				t.Errorf("each command must run after an earlier one fails, got %q", got)
			}
		}},
		{"a source-built engine installing the pinned checks", func(t *testing.T) {
			t.Parallel()
			gopath := t.TempDir()
			write(t, gopath, map[string]string{"bin/fitness-install": "#!/bin/sh\necho /engine/bin\necho /cache/checks\n"})
			run := runFitnessCI(t, ciStep{file: "fitness-install", command: "install-source-checks",
				saved: map[string][]response{"go": {{Match: []string{"env GOPATH"}, Stdout: gopath + "\n"}}}})
			sees(t, run.out, run.code, 0)
			if run.env != "FITNESS_CHECKS_DIR=/cache/checks\n" {
				t.Errorf("want the checks folder the install printed second, got %q", run.env)
			}
		}},
		{"a job whose suite wrote a performance record, then failed without an annotation", func(t *testing.T) {
			t.Parallel()
			dir, temp := t.TempDir(), t.TempDir()
			write(t, dir, map[string]string{"go/bin/fitness": tool("fitness")})
			write(t, filepath.Join(temp, "fitness-performance"), map[string]string{"fitness.json": "{}"})
			run := runFitnessCI(t, ciStep{file: "fitness-reports", command: "draw-box-diagram", dir: dir, env: []string{"FITNESS=go/bin/fitness", "RUNNER_TEMP=" + temp}})
			sees(t, run.out, run.code, 0, "fitness boxes "+filepath.Join(temp, "fitness-performance", "fitness.json"))
			run = runFitnessCI(t, ciStep{file: "fitness-reports", command: "report-outcome",
				env: []string{"JOB_STATUS=failure", "ALREADY_ANNOTATED=false", "GITHUB_JOB=check", "REPORTS_URL=https://example.com/reports"}})
			sees(t, run.out, run.code, 0, "::error::Workflow job check failed; open its logs for the failed step and cause.")
			if run.summary != "[Download complete Fitness reports](https://example.com/reports)\n" {
				t.Errorf("the summary must link the reports, got %q", run.summary)
			}
		}},
		{"a Fitness action run whose installation failed unannotated", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "action.yml", command: "report-action-outcome",
				env: []string{"REPORT_STATUS=failure", "REPORT_ALREADY_ANNOTATED=false", "GITHUB_SERVER_URL=https://github.com", "GITHUB_RUN_ID=9"}})
			sees(t, run.out, run.code, 0, "::error::Fitness installation or execution failed")
			if run.env != "FITNESS_FAILURE_REPORTED=true\n" || !strings.Contains(run.summary, "### Fitness action: failure") {
				t.Errorf("want the failure marked reported and summarized, got env %q and summary %q", run.env, run.summary)
			}
		}},
		{"a CI job whose suite wrote a performance record, and one whose suite did not", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, map[string]string{"go/bin/fitness": tool("fitness")})
			run := runFitnessCI(t, ciStep{file: "ci.yml", command: "show-performance", dir: dir})
			sees(t, run.out, run.code, 0)
			if strings.Contains(run.out, "fitness perf") {
				t.Errorf("no record means nothing to show:\n%s", run.out)
			}
			record := filepath.Join(t.TempDir(), "fitness-performance", "fitness.json")
			write(t, filepath.Dir(record), map[string]string{"fitness.json": "{}"})
			run = runFitnessCI(t, ciStep{file: "ci.yml", command: "show-performance", dir: dir, env: []string{"RUNNER_TEMP=" + filepath.Dir(filepath.Dir(record))}})
			sees(t, run.out, run.code, 0, "fitness perf "+record)
		}},
		{"a Go build cache family on a push to main", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "setup-go-cache", command: "cache-key",
				env: []string{"FAMILY=fitness", "RUNNER_OS=Linux", "RUNNER_ARCH=ARM64", "GO_VERSION=1.26.1", "SOURCE_HASH=abc", "SAVE=true"}})
			sees(t, run.out, run.code, 0)
			want := "prefix=go-build-fitness-Linux-ARM64-go1.26.1-\nkey=go-build-fitness-Linux-ARM64-go1.26.1-abc\nsave=true\n"
			if run.outputs != want {
				t.Errorf("want outputs %q, got %q", want, run.outputs)
			}
		}},
		{"this repo's CI building its own fitness", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "ci.yml", command: "build-fitness", saved: map[string][]response{"go": {{Match: []string{"build"}}}}})
			sees(t, run.out, run.code, 0)
			if got := run.rp.calls("go"); !slices.Equal(got, []string{"-C go build -o bin/ ./cmd/..."}) {
				t.Errorf("want every command built into go/bin, got %q", got)
			}
		}},
	}
}
