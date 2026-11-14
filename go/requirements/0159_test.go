package requirements

import (
	"os"
	"path/filepath"
	"testing"
)

// openQuestions runs `fitness-install -- open-questions` on happyRepo with
// files written over it.
func openQuestions(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "open-questions")
}

// askInChat is the fix every open-questions finding names.
const askInChat = "ask the owner in chat, or turn each unknown into a checklist step"

// questionsLabel is a label line that opens a list of open questions, and
// the finding it earns.
const (
	questionsLabel   = "Open questions:\n\n- Which sync wins?\n"
	questionsFinding = `a label line opens a list of open questions: "Open questions:"`
)

func Test0159_1(t *testing.T) {
	t.Parallel()
	doc := "# Sync\n\n## Open Questions\n\n- Which wins?\n\n**unresolved questions**\n\n- When?\n\nQuestions for the owner: which store?\n\n- **Open questions:** none yet\n"
	out, code := openQuestions(t, map[string]string{"docs/sync.md": doc})
	sees(t, out, code, 1,
		`docs/sync.md:3: a heading opens a list of open questions: "## Open Questions" — `+askInChat,
		`docs/sync.md:7: a bold line opens a list of open questions: "**unresolved questions**"`,
		`docs/sync.md:11: a label line opens a list of open questions: "Questions for the owner: which store?"`,
		`docs/sync.md:13: a label line opens a list of open questions: "- **Open questions:** none yet"`)
	out, code = openQuestions(t, map[string]string{"docs/sync.md": "# Sync\n\n## Decisions\n\n- The server wins.\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0159_2(t *testing.T) {
	t.Parallel()
	doc := "# Sync\n\nOpen questions go to chat, and we have no open questions left.\n\n**No open questions remain**\n\nUse `## Open questions` nowhere.\n\n```\nOpen questions:\n```\n\n<!-- Open questions: none -->\n"
	out, code := openQuestions(t, map[string]string{"docs/sync.md": doc})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = openQuestions(t, map[string]string{"docs/sync.md": doc + "\n" + questionsLabel})
	sees(t, out, code, 1, "docs/sync.md:15: "+questionsFinding)
}

func Test0159_3(t *testing.T) {
	t.Parallel()
	page := "<!doctype html>\n<main>\n  <h1>Sync</h1>\n  <h2 class=\"x\">Open <em>questions</em></h2>\n  <ul><li>Which wins?</li></ul>\n</main>\n"
	out, code := openQuestions(t, map[string]string{"site/index.html": page})
	sees(t, out, code, 1, `site/index.html:4: a heading opens a list of open questions: "Open questions" — `+askInChat)
	out, code = openQuestions(t, map[string]string{"site/index.html": "<main>\n  <h1>Sync</h1>\n  <p>We have no open questions.</p>\n  <pre>Open questions:</pre>\n</main>\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0159_4(t *testing.T) {
	t.Parallel()
	plan := planBody("- [x] Write the tests.")
	out, code := judge0014(t, "test(app): add one file", pr0014(11), issue0014(t, 11, plan, "Plan"))
	sees(t, out, code, 0, "Passed.")
	out, code = judge0014(t, "test(app): add one file", questionsLabel+"\n"+pr0014(11), issue0014(t, 11, plan, "Plan"))
	sees(t, out, code, 1, "(description):", questionsFinding)
	out, code, _ = judgeKind(t, plan0014("- [ ] Write the tests."), nil, kindGitHub("plan", "Plan"))
	sees(t, out, code, 0, "Passed.")
	out, code, _ = judgeKind(t, plan0014("- [ ] Write the tests.\n\n"+questionsLabel), nil, kindGitHub("plan", "Plan"))
	sees(t, out, code, 1, "(description):", questionsFinding)
}

// artifactEvent is the PreToolUse event for an Artifact call publishing page.
func artifactEvent(page string) map[string]any {
	return map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Artifact", "tool_input": map[string]any{"file_path": page}}
}

func Test0159_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	page := filepath.Join(t.TempDir(), "plan.html")
	mustDo(t, os.WriteFile(page, []byte("<h1>Sync plan</h1>\n<h2>Questions for the owner</h2>\n<ul><li>Which wins?</li></ul>\n"), 0o644))
	out, code := hookEvent(t, repo, artifactEvent(page))
	sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on the artifact page plan.html", "[open-questions]",
		`(description):2: a heading opens a list of open questions: \"Questions for the owner\"`)
	mustDo(t, os.WriteFile(page, []byte("<h1>Sync plan</h1>\n<p>The server wins.</p>\n"), 0o644))
	out, code = hookEvent(t, repo, artifactEvent(page))
	silentPass(t, out, code)
}
