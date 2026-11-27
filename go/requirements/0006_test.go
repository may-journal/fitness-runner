// cspell:ignore autorelease
// cspell:ignore fbody
// cspell:ignore zorblat

package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test0006_1(t *testing.T) {
	cases{
		{"a draft text and a repo with a file finding", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			out, code := fitness(t, repo, nil, "--blob", cleanText)
			if strings.Contains(out, "README.md") {
				t.Errorf("--blob must judge only the text, not repo files:\n%s", out)
			}
			for _, check := range []string{"cspell", "markdown-no-bold-italic", "prose-budget"} {
				passedWith(t, out, check, "1")
			}
			sees(t, out, code, 0, "Skipped (needs files):", "duplicate-code", "go-test", "markdown-links")
			if row(out, "duplicate-code") != "" {
				t.Errorf("duplicate-code needs files and must be skipped, not run:\n%s", out)
			}
		}},
		{"a draft text the chosen checks accept: the exit code says it passes", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "--blob="+cleanText, "--checks=cspell,markdown-no-bold-italic,duplicate-code")
			sees(t, out, code, 0, "All 2 checks passed", "Skipped (needs files): duplicate-code")
			passedWith(t, out, "cspell", "1")
			if row(out, "prose-budget") != "" {
				t.Errorf("--checks must narrow the run to the named checks:\n%s", out)
			}
		}},
		{"pipes a draft text with a finding on stdin", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := piped(t, repo, "Some **bold** text.\n", nil, "--blob", "-", "--checks=markdown-no-bold-italic")
			sees(t, out, code, 1, "[markdown-no-bold-italic] Please fix these items:", "disallowed **bold**", "0 of 1 checks passed, 1 failed")
		}},
		{"gives no text, selects no blob check, or adds `--all` or a check name", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			for _, args := range [][]string{
				{"--blob", ""},
				{"--blob", cleanText, "--all"},
				{"cspell", "--blob", cleanText},
				{"--blob", "--checks=cspell"},
				{"--blob", cleanText, "--checks=duplicate-code"},
			} {
				out, code := fitness(t, repo, nil, args...)
				sees(t, out, code, 2, "fitness --blob")
				if strings.Contains(out, "Running checks:") {
					t.Errorf("%q must run no check:\n%s", args, out)
				}
			}
		}},
		{"a text that follows its template, such as a Plan whose steps name issues", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			for _, c := range []struct {
				text, skipped string
				passed        []string
			}{
				{planBody("- [ ] Write the tests."), "comment-structure, epic-structure, issue-checklist, pr-closes-issue, pr-structure", []string{"plan-structure"}},
				{planBody("- [ ] #12 Add the schema."), "comment-structure, epic-structure, issue-checklist, pr-closes-issue, pr-structure", []string{"plan-structure"}},
				{epicBody("1. #22\n2. #23, after #22"), "comment-structure, issue-checklist, plan-structure, pr-closes-issue, pr-structure", []string{"epic-structure"}},
				{prPR(11), "comment-structure, epic-structure, issue-checklist, plan-structure", []string{"pr-structure", "pr-closes-issue"}},
			} {
				out, code := piped(t, repo, c.text, nil, "--blob", "-")
				sees(t, out, code, 0, "Skipped (does not apply to this text): "+c.skipped)
				for _, check := range c.passed {
					passedWith(t, out, check, "1")
				}
			}
		}},
		{"a PR description text whose changelog section is empty or its heading misnamed", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			misnamed := strings.Replace(prPR(11), "## Changelog", "### Changelog:", 1)
			for _, text := range []string{prBodyPR("", 11), misnamed} {
				out, code := piped(t, repo, text, nil, "--blob", "-")
				sees(t, out, code, 1, "[pr-structure] Please fix these items:", "Changelog")
			}
		}},
		{"a repo config that, under external policy, sets a checks list or disabledChecks", func(t *testing.T) {
			t.Parallel()
			for _, setting := range []string{`{"policy": "external", "disabledChecks": ["markdown-no-bold-italic"]}`, `{"policy": "external", "checks": ["cspell"]}`} {
				repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": setting + "\n"})
				out, code := piped(t, repo, "Some **bold** text.\n", nil, "--blob", "-")
				sees(t, out, code, 0)
				if row(out, "markdown-no-bold-italic") != "" {
					t.Errorf("--blob under %s ran markdown-no-bold-italic:\n%s", setting, out)
				}
			}
		}},
		{"a drafted comment with a change-log note and a callout table missing a diagram number", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "comment-check", "--blob", draft)
			sees(t, out, code, 1, sloppyError, "diagram callout 2 has no matching table row")
			out, code = piped(t, repo, draft, nil, "comment-check")
			sees(t, out, code, 1, sloppyError, "diagram callout 2 has no matching table row")
		}},
		{"a drafted comment whose numbered diagram matches its callout table", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "comment-check", "--blob=The parser now handles tabs.\n\n"+numbered)
			sees(t, out, code, 0, "All 11 checks passed")
		}},
	}.run(t)
}

func Test0006_2(t *testing.T) {
	cases{
		{"a fitness repo and Claude drafted a red PR body or comment", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			body := func(text string) string {
				return "gh pr create --title 'Add notes' --body \"$(cat <<'EOF'\n" + text + "EOF\n)\""
			}
			out, code := hookEvent(t, repo, bashEvent(body("Some **bold** text.\n")))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on the pr body", "[markdown-no-bold-italic]")
			out, code = hookEvent(t, repo, bashEvent(body(cleanPRBody)))
			silentPass(t, out, code)
			write(t, repo, map[string]string{"body.md": "Some **bold** text.\n", "sub/body.md": "Some **bold** text.\n"})
			for _, command := range []string{
				"cd sub && /opt/homebrew/bin/gh -R owner/repo pr create --title t -F body.md",
				"gh pr comment 5 --body-file - <<-EOF\n\tSome **bold** text.\n\tEOF\n",
				`gh pr create --title t --body='Some **bold** text.'`,
				`gh pr edit 5 -b "Some **bold** \"text\"."`,
				"gh pr create --title t --body-file - <<< 'Some **bold** text.'",
				"git push && gh pr create --title t -F body.md",
			} {
				out, code = hookEvent(t, repo, bashEvent(command))
				sees(t, out, code, 0, `"permissionDecision":"deny"`, "[markdown-no-bold-italic]")
			}
			comment := "cd sub && gh pr comment 5 --body-file - <<'EOF'\n```mermaid\nflowchart TB\n    a --> b\n```\nEOF\n"
			out, code = hookEvent(t, repo, bashEvent(comment))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on the pr comment body", "[comment-structure]")
		}},
		{"a fitness repo", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			for _, command := range []string{
				"ls -la && gh pr view 5 --json body",
				"gh pr create --fill",
				"gh pr edit 5 --add-reviewer someone",
				"gh issue close 5",
			} {
				out, code := hookEvent(t, repo, bashEvent(command))
				silentPass(t, out, code)
			}
		}},
		{"a fitness repo and Claude drafted a red comment", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			write(t, repo, map[string]string{"reply.md": redComment + "\n"})
			denied(t, repo,
				"gh api repos/o/r/issues/5/comments -f body='"+redComment+"'",
				"gh api -X PATCH repos/{owner}/{repo}/issues/comments/9 --raw-field body='"+redComment+"'",
				"gh api repos/o/r/pulls/5/comments -f path=a.go -F line=3 -F body=@reply.md",
				"gh api repos/o/r/pulls/comments/9/replies -fbody='"+redComment+"'",
				"gh api https://api.github.com/repos/o/r/commits/abc/comments -f body='"+redComment+"'",
				"gh api repos/o/r/pulls/5/reviews/9/events -f event=COMMENT -f body='"+redComment+"'",
			)
		}},
		{"a fitness repo and Claude drafted a review with a red line comment", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			review := `{"event": "COMMENT", "body": "Fine.", "comments": [{"path": "a.go", "body": "Fine."}, {"path": "b.go", "body": "` + redComment + `"}]}`
			write(t, repo, map[string]string{"review.json": review + "\n"})
			out, code := hookEvent(t, repo, bashEvent("gh api repos/o/r/pulls/5/reviews --input review.json"))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on review line comment 2")
			fields := "gh api repos/o/r/pulls/5/reviews -f event=COMMENT -f body=Fine. -f 'comments[][path]=a.go' -f 'comments[][body]="
			out, code = hookEvent(t, repo, bashEvent(fields+redComment+"'"))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on review line comment 1")
			out, code = hookEvent(t, repo, bashEvent(fields+"Fine.'"))
			silentPass(t, out, code)
		}},
		{"a fitness repo and Claude drafted a red comment: it denies the call with the findings", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			mutation := `mutation($id: ID!, $body: String!) { addComment(input: {subjectId: $id, body: $body}) { clientMutationId } }`
			thread := `mutation($id: ID!, $text: String!) { addPullRequestReviewThread(input: {pullRequestId: $id, path: "a.go", line: 3, body: $text}) { clientMutationId } }`
			request := `{"query": "mutation($b: String!) { addDiscussionComment(input: {discussionId: \"D\", body: $b}) { clientMutationId } }", "variables": {"b": "` + redComment + `"}}`
			write(t, repo, map[string]string{"request.json": request + "\n"})
			denied(t, repo,
				"gh api graphql -f query='"+mutation+"' -f id=X -f body='"+redComment+"'",
				"gh api graphql -f query='"+thread+"' -f id=X -f text='"+redComment+"'",
				"gh api graphql --input request.json",
				"gh issue close 5 --comment '"+redComment+"'",
				"gh pr close 5 -c '"+redComment+"'",
			)
		}},
		{"a fitness repo: it exits 0 with no output", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			passed(t, repo,
				"gh api repos/o/r/pulls/5/comments",
				"gh api -X GET repos/o/r/issues/5/comments -f per_page=100",
				"gh api -X DELETE repos/o/r/issues/comments/9",
				"gh api -X POST repos/o/r/pulls/5/reviews/9/events -f event=APPROVE",
				"gh api graphql -f query='query { viewer { login } }'",
				"gh api repos/o/r/issues/5/labels -f 'labels[]=bug'",
			)
		}},
		{"a fitness repo whose `cspell.json` allows a word, and Claude drafted a comment with it", func(t *testing.T) {
			t.Parallel()
			repo := wordRepo(t)
			comment := "--body '" + allowedWord + " is ready.'"
			for _, command := range []string{
				"cd " + repo + " && gh pr comment 5 " + comment,
				"cd " + repo + " && gh -R o/r pr comment 5 " + comment,
				"cd " + repo + " && gh api repos/O/R/issues/5/comments -f body='" + allowedWord + " is ready.'",
				"cd " + repo + " && gh pr comment 5 --body 'https://github.com/a/b/pull/3 " + allowedWord + " is ready.'",
			} {
				out, code := hookEvent(t, t.TempDir(), bashEvent(command))
				silentPass(t, out, code)
			}
		}},
		{"a fitness repo whose `cspell.json` allows a word, and Claude drafted a comment with it: it denies the call with the org's cspell findings", func(t *testing.T) {
			t.Parallel()
			repo := wordRepo(t)
			comment := "--body '" + allowedWord + " is ready.'"
			want := []string{"[cspell]", allowedWord}
			deniedWith(t, repo, want,
				"gh pr comment 5 -R other/repo "+comment,
				"gh -R other/repo pr comment 5 "+comment,
				`GH_REPO="$R" gh pr comment 5 `+comment,
				`GH_REPO=o/other bash -c "gh pr comment 5 `+comment+`"`,
			)
			deniedWith(t, t.TempDir(), want, "gh pr comment 5 "+comment)
		}},
		{"a fitness repo: it denies the call, saying how to pass a body it can read", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			write(t, repo, map[string]string{"notes.md": cleanText, "reply.md": cleanText})
			deniedWith(t, repo, []string{"can't read", "--body-file"},
				`gh pr comment 5 --body "$BODY"`,
				`gh pr comment 5 --body="$BODY"`,
				"cat notes.md | gh pr comment 5 --body-file -",
				"printf 'Fine.' > fresh.md && gh pr create --title t -F fresh.md",
				"gh pr create --title t --body-file - <<EOF\nSee $(cat notes.md)\nEOF\n",
				`gh api repos/o/r/issues/5/comments -f body="$BODY"`,
				`gh api repos/o/r/issues/5/comments -F "body=@${DIR}reply.md"`,
				`gh api repos/o/r/issues/$N/comments --input "$F"`,
				"gh api graphql -f query='mutation($input: AddCommentInput!) { addComment(input: $input) { clientMutationId } }' -F 'input[body]=Fine.'",
			)
		}},
		{"a fitness repo and Claude drafted a red comment: it denies the call with the findings", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			red := "--body '" + redComment + "'"
			deniedWith(t, repo, []string{"[markdown-no-bold-italic]"},
				"gh pr view 5 && gh pr comment 5 --body 'Fine.' && gh issue comment 6 "+red,
				"for n in 1 2; do gh pr comment $n "+red+"; done",
				`bash -lc "gh pr comment 5 `+red+`"`,
				"echo 5 | xargs -I{} gh pr comment {} "+red,
				"timeout -s KILL 30 gh pr comment 5 "+red,
				"sudo -u bot env -i GH_HOST=github.com gh pr comment 5 "+red,
				`bash --norc -c "gh pr comment 5 `+red+`"`,
			)
		}},
		{"a fitness repo and Claude drafted a red comment: it denies the call with the findings", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			deniedWith(t, repo, []string{"[markdown-no-bold-italic]"}, "gh api repos/o/r/pulls/5/comments/9/replies -f body='"+redComment+"'")
			out, code := hookEvent(t, repo, mcpEvent("mcp__github__add_issue_comment",
				map[string]any{"owner": "o", "repo": "r", "issue_number": 5, "body": redComment}))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "[markdown-no-bold-italic]")
			review := map[string]any{"owner": "o", "repo": "r", "pullNumber": 5, "event": "COMMENT", "body": "Fine.",
				"comments": []map[string]any{{"path": "a.go", "line": 3, "body": redComment}}}
			out, code = hookEvent(t, repo, mcpEvent("mcp__plugin_github_github__create_pull_request_review", review))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on review line comment 1")
		}},
		{"a fitness repo and Claude drafted a PR body or Plan with no Background section", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			noBackground := "> Adds the notes.\n\n## Changelog\n\n- Notes.\n"
			plan := "> The notes are kept.\n\n## What needs to happen\n\n- [ ] Keep the notes\n"
			write(t, repo, map[string]string{"pr.md": noBackground, "plan.md": plan})
			want := "Missing `## Background` section"
			deniedWith(t, repo, []string{"[pr-structure]", want}, "gh pr create --title t -F pr.md")
			deniedWith(t, repo, []string{"[plan-structure]", want},
				"gh issue create --title 'Keep notes' --label Plan -F plan.md",
				"gh issue create --title '[Plan] Keep notes' -F plan.md",
				"gh issue edit 7 --add-label Plan -F plan.md",
			)
			out, code := hookEvent(t, repo, mcpEvent("mcp__github__create_pull_request",
				map[string]any{"owner": "o", "repo": "r", "title": "t", "head": "b", "base": "main", "body": noBackground}))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "[pr-structure]", want)
			out, code = hookEvent(t, repo, bashEvent("gh pr create --title t --body '"+cleanPRBody+"'"))
			silentPass(t, out, code)
		}},
	}.run(t)
}

func Test0006_3(t *testing.T) {
	cases{
		{"a failing comment on an issue with no reply yet", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgeComment(t, eventComment(sloppy, "dev", "User"), commentGitHub(""))
			sees(t, out, code, 1, "comment-check #9", "Update:", "Re-check a draft before posting with `fitness comment-check --blob -`")
			reply := rp.called("gh", "-X POST", "body=❌ @dev [This comment]("+commentURL+")", "Update:", "comment-check --blob -", "issues/9/comments")
			if !reply || !rp.called("gh", "-X POST", "content=-1", "issues/comments/31/reactions") {
				t.Errorf("want one reply naming the author and a thumbs-down, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a failing comment whose reply already states the same violations", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgeComment(t, eventComment(sloppy, "dev", "User"), commentGitHub(commentNotice(sloppyError)))
			sees(t, out, code, 1, "Update:")
			if rp.called("gh", "body=") {
				t.Errorf("want no comment written, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a failing comment that is then fixed", func(t *testing.T) {
			t.Parallel()
			fixed := eventComment("The parser now handles tabs.\n", "dev", "User")
			out, code, rp := judgeComment(t, fixed, commentGitHub(commentNotice(sloppyError)))
			sees(t, out, code, 0, "Passed.")
			edit := rp.called("gh", "-X PATCH", "body=✅ [This comment]("+commentURL+") now passes", "issues/comments/41")
			if !edit || !rp.called("gh", "-X DELETE", "issues/comments/31/reactions/55") {
				t.Errorf("want the reply turned into a passing note and the reaction removed, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"comments from a known bot, an ignored author, fitness under a custom list, and a pasted fitness marker", func(t *testing.T) {
			t.Parallel()
			verdict := "✅ Plan looks good.\n\n<!-- fitness:plan-structure:0123456789ab -->"
			for _, c := range []struct {
				comment map[string]any
				ignore  string
			}{
				{eventComment(sloppy, "may-journal-automation[bot]", "Bot"), ""},
				{eventComment(verdict, "github-actions[bot]", "Bot"), "renovate[bot]"},
				{eventComment(sloppy, "someone", "User"), "release-bot[bot], someone"},
			} {
				out, code, rp := judgeComment(t, c.comment, commentGitHub(""), "FITNESS_COMMENT_IGNORE_AUTHORS="+c.ignore)
				sees(t, out, code, 0, "Skipped "+commentURL, "0 targets checked")
				if len(rp.calls("gh")) > 0 {
					t.Errorf("want nothing read or written, got gh calls %q", rp.calls("gh"))
				}
			}
			out, code, rp := judgeComment(t, eventComment(verdict, "dev", "User"), commentGitHub(""))
			sees(t, out, code, 1, "carries a fitness marker")
			if !rp.called("gh", "-X POST", "body=❌ @dev", "issues/9/comments") {
				t.Errorf("want the pasted marker judged and replied to, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a failing review comment in a pull request thread", func(t *testing.T) {
			t.Parallel()
			comment := eventComment(sloppy, "dev", "User")
			comment["in_reply_to_id"] = 30
			pr := map[string]any{"number": 9, "title": "t", "body": ""}
			payload := map[string]any{"action": "created", "pull_request": pr, "comment": comment}
			out, code, rp := commentCheck(t, "pull_request_review_comment", payload, commentGitHub(""))
			sees(t, out, code, 1, "Update:")
			reply := rp.called("gh", "-X POST", "body=❌ @dev", "pulls/9/comments/30/replies")
			if !reply || !rp.called("gh", "-X POST", "content=-1", "pulls/comments/31/reactions") {
				t.Errorf("want a reply in the review thread and a thumbs-down, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a manual workflow run", func(t *testing.T) {
			t.Parallel()
			saved := append([]response{
				{Match: []string{"issues/comments?per_page=100&since="}, Stdout: sweptComment(31, "issues/9")},
				{Match: []string{"pulls/comments?per_page=100&since="}, Stdout: sweptComment(32, "pulls/9")},
			}, commentGitHub("")...)
			out, code, rp := commentCheck(t, "workflow_dispatch", map[string]any{}, saved)
			sees(t, out, code, 1, "2 targets checked")
			issue := rp.called("gh", "-X POST", "body=❌", "issues/9/comments")
			if !issue || !rp.called("gh", "-X POST", "body=❌", "pulls/9/comments/32/replies") {
				t.Errorf("want the issue and review comments judged, got gh calls %q", rp.calls("gh"))
			}
			since := strings.Contains(strings.Join(rp.calls("gh"), "\n"), "since=")
			if !since {
				t.Errorf("want the sweep limited to recent comments, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a submitted review with ten comments, two of them failing", func(t *testing.T) {
			t.Parallel()
			saved := append([]response{{Match: []string{"pulls/9/reviews/70/comments"}, Stdout: reviewComments()}}, commentGitHub("")...)
			pr := map[string]any{"number": 9, "title": "t", "body": ""}
			review := map[string]any{"id": 70, "user": map[string]any{"login": "dev", "type": "User"}}
			payload := map[string]any{"action": "submitted", "pull_request": pr, "review": review}
			out, code, rp := commentCheck(t, "pull_request_review", payload, saved)
			sees(t, out, code, 1, "10 targets checked")
			if rp.count("gh", "-X POST", "body=❌ @dev") != 2 {
				t.Errorf("want two replies, one per failing comment, got gh calls %q", rp.calls("gh"))
			}
			for _, root := range []string{"31", "32"} {
				if !rp.called("gh", "-X POST", "body=❌ @dev", "pulls/9/comments/"+root+"/replies") {
					t.Errorf("want a reply in comment %s's thread, got gh calls %q", root, rp.calls("gh"))
				}
			}
		}},
	}.run(t)
}

func Test0006_4(t *testing.T) {
	cases{
		{"a Plan that follows the template: the Plan shows one current verdict, then one current label", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), planGitHub(""))
			sees(t, out, code, 0, "Passed.")
			if !commented(rp, "body=✅ Plan looks good.") || !labeled(rp, "fitness-valid") {
				t.Errorf("want a passing verdict and fitness-valid, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a Plan that breaks the template", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgePlan(t, brokenPlan, planGitHub(""))
			sees(t, out, code, 1, "Background")
			if !commented(rp, `body=❌ Plan needs work:\n- `) || !labeled(rp, "fitness-invalid") {
				t.Errorf("want a failing verdict listing errors and fitness-invalid, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a Plan whose verdict another author pasted, alone or after plan-check's own", func(t *testing.T) {
			t.Parallel()
			body := planBody("- [ ] Write the tests.")
			own := planVerdict("✅ Plan looks good.", body)
			pasted := verdictBy("dev", "User", 12, "✅ Plan looks good.", body)
			for _, c := range []struct {
				verdicts string
				posts    bool
			}{{own, false}, {pasted, true}, {own + pasted, false}} {
				out, code, rp := judgePlan(t, body, planGitHub(c.verdicts, "fitness", "fitness-valid"))
				sees(t, out, code, 0, "Passed.")
				if commented(rp, "body=✅ Plan looks good.") != c.posts || touched(rp, "12") {
					t.Errorf("want a new verdict %v and the pasted one untouched, got gh calls %q", c.posts, rp.calls("gh"))
				}
			}
		}},
		{"a failed Plan that is then fixed", func(t *testing.T) {
			t.Parallel()
			failed := planVerdict("❌ Plan needs work:\n- Missing section: Background", brokenPlan)
			out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), planGitHub(failed, "fitness", "fitness-invalid"))
			sees(t, out, code, 0, "Passed.")
			if !commented(rp, "body=✅ Plan looks good.") || !rp.called("gh", "minimizeComment", "id=IC_11") {
				t.Errorf("want a passing verdict and the failed one hidden, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a passing Plan edited so it still passes", func(t *testing.T) {
			t.Parallel()
			passed := planVerdict("✅ Plan looks good.", planBody("- [ ] Write the tests."))
			edited := planBody("- [ ] Write the tests.\n- [ ] Write the doc.")
			out, code, rp := judgePlan(t, edited, planGitHub(passed, "fitness", "fitness-valid"))
			sees(t, out, code, 0, "Passed.")
			edit := rp.called("gh", "-X PATCH", "body=✅ Validated (updated ", "issues/comments/11")
			if !edit || commented(rp) {
				t.Errorf("want the passing verdict updated in place, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a manual workflow run", func(t *testing.T) {
			t.Parallel()
			plan, _ := json.Marshal(map[string]any{"number": 7, "body": planBody("- [ ] Write the tests.")})
			saved := append([]response{{Match: []string{"issues?labels=Plan&state=open"}, Stdout: string(plan) + "\n"}}, planGitHub("")...)
			out, code, rp := planCheck(t, "workflow_dispatch", map[string]any{}, saved)
			sees(t, out, code, 0, "plan-check #"+planIssue)
			if !labeled(rp, "fitness-valid") {
				t.Errorf("open Plan #%s was not judged, got gh calls %q", planIssue, rp.calls("gh"))
			}
		}},
		{"a malformed Plan in both the PR and trailer, or from a fork whose PR base exists only upstream", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
			commit(t, repo, nil, "feat(app): add one file\n\nPlan #11")
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, brokenPlan, "Plan"))})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 1, "plan-check #11", "Background")
			readOnce(t, rp, "11")

			fork, forkPR := forkBranch(t, "release")
			git(t, fork, "update-ref", "-d", "refs/remotes/origin/main")
			rp = standIn(t, map[string][]response{"gh": branchGitHub(forkPR, issuePR(t, 11, "An ordinary issue."), issuePR(t, 12, brokenPlan, "Plan"))})
			out, code = fitness(t, fork, rp.env, "branch-check")
			sees(t, out, code, 1, "plan-check #12", "Background")
			readOnce(t, rp, "12")
		}},
		{"a Plan moved to another repo, one with only its title edited, or one given another label", func(t *testing.T) {
			t.Parallel()
			gone := response{Match: []string{"fitness:plan-structure:"}, Stderr: "gh: Could not resolve to an Issue with the number of " + planIssue + ".\n", Exit: 1}
			out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), append([]response{gone}, planGitHub("")...))
			sees(t, out, code, 0, "#"+planIssue+" no longer exists")
			if rp.called("gh", "-X POST") {
				t.Errorf("want nothing written to a moved Plan, got gh calls %q", rp.calls("gh"))
			}

			skipsIdle(t, "plan-check", idleEdits("Bug", "fitness-invalid"))
		}},
	}.run(t)
}

func Test0006_5(t *testing.T) {
	cases{
		{"an Epic that lists each of its sub-issues as a plain reference in run order", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgeKind(t, epicBody("1. #22\n2. #23, after #22"), nil, epicGitHub("22", "23"))
			sees(t, out, code, 0, "Passed.")
			if !commentedOn(rp, kindIssue, "body=✅ Epic looks good.") || !labeledOn(rp, kindIssue, "fitness-valid") {
				t.Errorf("want a passing Epic verdict and fitness-valid, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"an Epic list with a checkbox, an emoji, a repeated reference, a table, an out-of-order item, or an oversized group", func(t *testing.T) {
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
		}},
		{"an Epic whose list and GitHub sub-issues differ", func(t *testing.T) {
			t.Parallel()
			saved := append([]response{
				{Match: []string{"labels: [.labels[].name]", "issues/24"}, Stdout: `{"body": "", "labels": []}` + "\n"},
				{Match: []string{"issues/26"}, Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1},
			}, epicGitHub("22", "23")...)
			out, code, _ := judgeKind(t, epicBody("1. #22\n2. #24\n3. #26"), nil, saved)
			sees(t, out, code, 1, "24 is listed but is not a sub-issue of this epic", "26 is listed but cannot be read",
				"sub-issue \\#23 is not listed under")
		}},
		{"an issue of a kind that has only a template, or a Plan whose repo template adds a section", func(t *testing.T) {
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
		}},
		{"an issue with two kind labels", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgeKind(t, planBody("- [ ] Write the tests."), nil, kindGitHub("plan", "Plan", "Epic"))
			sees(t, out, code, 1, "the issue has 2 kind labels (Plan, Epic); keep the one label that names its kind")
			if !labeledOn(rp, kindIssue, "fitness-invalid") {
				t.Errorf("want fitness-invalid, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a Plan whose body plan-check already judged", func(t *testing.T) {
			t.Parallel()
			body := planBody("- [ ] Write the tests.")
			issue := map[string]any{"number": 7, "title": "t", "body": body}
			saved := planGitHub(planVerdict("✅ Plan looks good.", body), "fitness", "fitness-valid")
			out, code, rp := ghRun(t, "issue-check", "issues", bodyEdit(issue), nil, saved)
			sees(t, out, code, 0, "Passed.")
			if rp.called("gh", "body=") {
				t.Errorf("want no comment written, got gh calls %q", rp.calls("gh"))
			}
		}},
		{"a manual issue-check workflow run", func(t *testing.T) {
			t.Parallel()
			plan, _ := json.Marshal(map[string]any{"number": 7, "body": planBody("- [ ] Write the tests."), "labelNames": []string{"Plan"}})
			epic, _ := json.Marshal(map[string]any{"number": 21, "body": epicBody("1. #22"), "labelNames": []string{"Epic"}})
			saved := append([]response{
				{Match: []string{"issues?labels=Plan&state=open"}, Stdout: string(plan) + "\n"},
				{Match: []string{"issues?labels=Epic&state=open"}, Stdout: string(epic) + "\n"},
				{Match: []string{"issues?labels=ADR&state=open"}},
				{Match: []string{"issues?labels=Research&state=open"}},
			}, epicGitHub("22")...)
			saved = append(saved, issueGitHub(planIssue, "fitness:issue-check:plan:", "", "Plan")...)
			research := map[string]string{".github/ISSUE_TEMPLATE/research.md": "---\nname: Research\nlabels: Research\n---\n\n> REPLACE-ME\n\n## Question\n\n<!-- What to find out. -->\n"}
			out, code, rp := ghRun(t, "issue-check", "workflow_dispatch", map[string]any{}, research, saved)
			sees(t, out, code, 0, "issue-check #7", "issue-check #21")
			if !labeledOn(rp, planIssue, "fitness-valid") || !labeledOn(rp, kindIssue, "fitness-valid") {
				t.Errorf("want both open kind issues judged, got gh calls %q", rp.calls("gh"))
			}
			for _, kind := range []string{"Plan", "Epic", "ADR", "Research"} {
				if !rp.called("gh", "issues?labels="+kind+"&state=open") {
					t.Errorf("want the open %s issues listed, as every kind is judged, got gh calls %q", kind, rp.calls("gh"))
				}
			}
		}},
		{"an issue with no kind label, one moved away, one with only its title edited, or a verdict label added", func(t *testing.T) {
			t.Parallel()
			out, code, rp := judgeKind(t, "Something odd happened.", nil, kindGitHub("plan", "question"))
			sees(t, out, code, 0, "has no issue kind label, so it is not checked.")
			if rp.called("gh", "-X POST") {
				t.Errorf("want nothing written to an issue with no kind, got gh calls %q", rp.calls("gh"))
			}

			issue := map[string]any{"number": 21, "title": "t", "body": epicBody("1. #22"), "labels": []map[string]string{{"name": "Epic"}}}
			gone := response{Match: []string{"fitness:issue-check:epic:"}, Stderr: "gh: Could not resolve to an Issue with the number of 21.\n", Exit: 1}
			out, code, rp = ghRun(t, "issue-check", "issues", bodyEdit(issue), nil, append([]response{gone}, epicGitHub("22")...))
			sees(t, out, code, 0, "21 no longer exists")
			if rp.called("gh", "-X POST") || rp.called("gh", "labels: [.labels[].name]") {
				t.Errorf("want the event's labels used and nothing written to a moved issue, got gh calls %q", rp.calls("gh"))
			}

			skipsIdle(t, "issue-check", idleEdits("fitness", "fitness-valid", "fitness-invalid"))
		}},
		{"an `ADR` issue with a `[ADR]` title, Background, Decision and Currently Known Consequences", func(t *testing.T) {
			t.Parallel()
			template, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", "adr.md"))
			mustDo(t, err)
			for name, files := range map[string]map[string]string{
				"the built-in shape": nil,
				"the repo template":  {".github/ISSUE_TEMPLATE/adr.md": string(template)},
			} {
				out, code, rp := judgeADR(t, "[ADR] Store notes in SQLite", adrIssueBody, files)
				sees(t, out, code, 0, "Passed.")
				if !commentedOn(rp, kindIssue, "body=✅ ADR looks good.") {
					t.Errorf("want a passing ADR verdict by %s, got gh calls %q", name, rp.calls("gh"))
				}
			}
		}},
		{"an `ADR` issue with a plain title, a Status section, or no Currently Known Consequences", func(t *testing.T) {
			t.Parallel()
			template, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", "adr.md"))
			mustDo(t, err)
			files := map[string]string{".github/ISSUE_TEMPLATE/adr.md": string(template)}
			body := strings.Replace(adrIssueBody, "## Currently Known Consequences\n\nOne file holds every note.\n", "## Status\n\nAccepted\n", 1)
			out, code, rp := judgeADR(t, "Store notes in SQLite", body, files)
			sees(t, out, code, 1, "the title must read", "Unexpected", "Status", "Missing", "Currently Known Consequences")
			if !labeledOn(rp, kindIssue, "fitness-invalid") {
				t.Errorf("want fitness-invalid, got gh calls %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

func Test0006_6(t *testing.T) {
	cases{
		{"a draft PR that follows the template, linking its changelog entry, and closes a finished Plan: it passes only if the title, then the body, then the closed issues are ready", func(t *testing.T) {
			t.Parallel()
			out, code := judgePR(t, "test(app): add one file", prPR(11), issuePR(t, 11, planPR("- [x] Write the tests."), "Plan"))
			sees(t, out, code, 0, "Passed.")
			repo := branchRepo(t, nil)
			pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"))})
			local, localCode := fitness(t, repo, rp.env, "branch-check")
			sees(t, local, localCode, 0, "pr-check #1", "plan-check #11", "https://github.com/"+repoPR+"/pull/1")
			if rp.called("gh", "-X") || rp.called("gh", "graphql") {
				t.Fatalf("local validation wrote GitHub state: %q", rp.calls("gh"))
			}
		}},
		{"a PR whose title is not a semantic commit", func(t *testing.T) {
			t.Parallel()
			out, code := judgePR(t, "added one file", prPR(11), issuePR(t, 11, planPR("- [x] Write the tests."), "Plan"))
			sees(t, out, code, 1, `"added one file"`)
		}},
		{"a PR that closes a Plan with an unchecked item", func(t *testing.T) {
			t.Parallel()
			out, code := judgePR(t, "test(app): add one file", prPR(11), issuePR(t, 11, planPR("- [ ] Write the tests."), "Plan"))
			sees(t, out, code, 1, `11, which has an unchecked item: "Write the tests."`)
			repo := branchRepo(t, nil)
			pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, planBody("- [ ] Write the tests."), "Plan"))})
			local, localCode := fitness(t, repo, rp.env, "branch-check")
			sees(t, local, localCode, 1, `11, which has an unchecked item: "Write the tests."`)
		}},
		{"a PR that leaves open an issue its Plan closes", func(t *testing.T) {
			t.Parallel()
			plan := planPR("- [x] Fix the bug.") + "\nCloses #12\n"
			out, code := judgePR(t, "test(app): add one file", prPR(11), issuePR(t, 11, plan, "Plan"))
			sees(t, out, code, 1, "#12")
		}},
		{"a manual workflow run", func(t *testing.T) {
			t.Parallel()
			var open string
			for _, n := range []int{5, 6} {
				line, err := json.Marshal(map[string]any{"number": n, "title": "test(app): add one file", "body": prPR(10 + n)})
				mustDo(t, err)
				open += string(line) + "\n"
			}
			done := planPR("- [x] Write the tests.")
			rp := standIn(t, map[string][]response{"gh": {
				{Match: []string{"--paginate", "repos/" + repoPR + "/pulls?state=open"}, Stdout: open},
				issuePR(t, 15, done, "Plan"),
				issuePR(t, 16, done, "Plan"),
				noChangelogPR,
			}})
			env := actionsEvent(t, "workflow_dispatch", map[string]any{})
			out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
			sees(t, out, code, 0, "pr-check #5", "pr-check #6")
		}},
		{"a PR closing an inaccessible issue", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			issue := response{Match: []string{"issues/11"}, Exit: 1, Stderr: "HTTP 404: issue inaccessible"}
			pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue)})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 1, "cannot validate linked issue #11", "retry")
			actions, actionsCode := judgePR(t, "test(app): add one file", prPR(11), issue)
			sees(t, actions, actionsCode, 1, `cannot validate linked issue \#11`)
		}},
		{"fixed a PR's title and body after its check failed", func(t *testing.T) {
			t.Parallel()
			stale := strings.Replace(prPR(11), "## Background", "## Context", 1)
			fixed := prPR(11)
			plan := issuePR(t, 11, planPR("- [x] Write the tests."), "Plan")
			out, code, _ := rerunPR(t, "added one file", stale, livePR(t, "test(app): add one file", fixed), plan)
			sees(t, out, code, 0, "pr-check #1", "Passed.")
		}},
		{"a PR check whose runner cannot reach GitHub", func(t *testing.T) {
			t.Parallel()
			offline := response{Match: []string{"title, body", "repos/" + repoPR + "/pulls/1"}, Stderr: "error connecting to api.github.com\n", Exit: 1}
			plan := issuePR(t, 11, planPR("- [x] Write the tests."), "Plan")
			out, code, _ := rerunPR(t, "added one file", prPR(11), offline, plan)
			sees(t, out, code, 1, "cannot reach GitHub for PR #1", `"added one file"`)
		}},
	}.run(t)
}

func Test0006_7(t *testing.T) {
	cases{
		{"a PR that amends its own or a pending entry, or adds the first `CHANGELOG.md`, even when rebased", func(t *testing.T) {
			t.Parallel()
			entry := entryFileText(entryFirst, "Feat: add one file.")
			repo, pr := entryCommits(t, nil, entryFileBody(entryFirstFile),
				map[string]string{entryFirstFile: entry},
				map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
			out, code := entryActions(t, repo, entryFileBody(entryFirstFile))
			sees(t, out, code, 0, "Passed.")
			local, localCode := entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
			repo, pr = entryCommits(t, withEntries(map[string]string{"2026-10-10-0900-one-file": entry}), prPR(11),
				map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
			out, code = entryActions(t, repo, prPR(11))
			sees(t, out, code, 0, "Passed.")
			local, localCode = entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
			repo, pr = entryCommits(t, map[string]string{"CHANGELOG.md": ""}, entryFileBody(entryFirstFile),
				map[string]string{entryFirstFile: entry, "CHANGELOG.md": changelogBase})
			out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
			sees(t, out, code, 0, "Passed.")
			local, localCode = entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
			repo, pr = entryRebased(t, withEntries(map[string]string{"2026-10-10-1000-merged-file": entryFileText(entryLater, "Fix: merged from another PR.")}),
				entryFileBody(entryFirstFile),
				map[string]string{entryFirstFile: entry},
				map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
			out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
			sees(t, out, code, 0, "Passed.")
			local, localCode = entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
		}},
		{"a PR description that links a different changelog entry or none, or copies one of its bullets", func(t *testing.T) {
			t.Parallel()
			added := map[string]string{entryFirstFile: entryFileText(entryFirst, "Feat: add one file to the app.")}
			repo, _ := entryCommits(t, nil, entryFileBody("changelog/other-file.md"), added)
			out, code := entryActions(t, repo, entryFileBody("changelog/other-file.md"))
			sees(t, out, code, 1, "PR description links changelog/other-file.md; link the entry this PR adds", entryFirstFile)
			out, code = entryActions(t, repo, prBodyPR("- The entry covers the change.", 11))
			sees(t, out, code, 1, "PR description has no changelog/ entry link; link the entry this PR adds", entryFirstFile)
			body := strings.Replace(entryFileBody(entryFirstFile), "\n\nCloses", "\n- Feat: add one file to the app.\n\nCloses", 1)
			out, code = entryActions(t, repo, body)
			sees(t, out, code, 1, "PR description copies the "+entryFirstFile+" line", "Feat: add one file to the app.", "link the entry instead")
		}},
		{"a release PR that adds no entry file, or a PR that edits `CHANGELOG.md` beside its linked entry file", func(t *testing.T) {
			t.Parallel()
			release := prBodyPR(entryShips("abc1234"), 11)
			repo, pr := entryCommits(t, nil, release, map[string]string{"version.txt": "1.1.0\n"})
			out, code := entryActions(t, repo, release)
			sees(t, out, code, 0, "Passed.")
			local, localCode := entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
			repo, pr = entryCommits(t, nil, entryFileBody(entryFirstFile), map[string]string{
				entryFirstFile: entryFileText(entryFirst, "Feat: add one file."),
				"CHANGELOG.md": changelogEntry(entryFirst, "Feat: add one file."),
			})
			out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
			sees(t, out, code, 0, "Passed.")
			local, localCode = entryLocal(t, repo, pr)
			sees(t, local, localCode, 0, "pr-check #1")
		}},
		{"a PR that changes the notes of an entry file the newest release holds", func(t *testing.T) {
			t.Parallel()
			old := entryFileText("2026.10.01.0900", "Feat: ship the old file.")
			for _, edit := range []string{old + "- Fix: a late bullet.\n", ""} {
				repo, pr := entryCommits(t, map[string]string{entryOldFile: old}, prPR(11), map[string]string{entryOldFile: edit})
				out, code := entryActions(t, repo, prPR(11), entryReleased()...)
				sees(t, out, code, 1, entryShipped)
				local, localCode := entryLocal(t, repo, pr, entryReleased()...)
				sees(t, local, localCode, 1, entryShipped)
			}
			entryFrontDropped(t)
		}},
		{"a PR whose commits add two entry files, or one whose name lacks its heading's date", func(t *testing.T) {
			t.Parallel()
			repo, pr := entryCommits(t, nil, entryFileBody("changelog/2026-10-10-0900-a-file.md"),
				map[string]string{"changelog/2026-10-10-0900-a-file.md": entryFileText(entryFirst, "Feat: add one file.")},
				map[string]string{"changelog/2026-10-10-1000-b-file.md": entryFileText(entryLater, "Test: cover the file.")})
			want := "this PR adds 2 changelog entries; keep changelog/2026-10-10-0900-a-file.md and move the bullets of changelog/2026-10-10-1000-b-file.md into it"
			out, code := entryActions(t, repo, entryFileBody("changelog/2026-10-10-0900-a-file.md"))
			sees(t, out, code, 1, want)
			local, localCode := entryLocal(t, repo, pr)
			sees(t, local, localCode, 1, want)
			entryUndated(t)
		}},
	}.run(t)
}
