// cspell:ignore zorblat
package requirements

import (
	"testing"
)

// allowedWord is a word only the repo's own cspell.json allows.
const allowedWord = "Zorblat"

// cleanPRBody is a PR body pr-structure and the doc checks pass.
const cleanPRBody = "> Adds the notes.\n\n## Background\n\nThe notes were missing.\n\n## Changelog\n\n- Notes.\n"

// wordRepo is a fitness repo on origin o/r whose cspell.json allows
// allowedWord.
func wordRepo(t *testing.T) string {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"cspell.json": `{"words": ["` + allowedWord + `"]}` + "\n"})
	git(t, repo, "remote", "add", "origin", "https://github.com/o/r.git")
	return repo
}

// mcpEvent is the PreToolUse event for an MCP tool with input.
func mcpEvent(tool string, input map[string]any) map[string]any {
	return map[string]any{"hook_event_name": "PreToolUse", "tool_name": tool, "tool_input": input}
}

// deniedWith asserts the hook denies each command run from dir, showing
// every want.
func deniedWith(t *testing.T, dir string, want []string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, dir, bashEvent(command))
			sees(t, out, code, 0, append([]string{`"permissionDecision":"deny"`}, want...)...)
		})
	}
}

func Test0155_1(t *testing.T) {
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
}

func Test0155_2(t *testing.T) {
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
}

func Test0155_3(t *testing.T) {
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
}

func Test0155_4(t *testing.T) {
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
}

func Test0155_5(t *testing.T) {
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
}

func Test0155_6(t *testing.T) {
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
}
