package requirements

import (
	"strings"
	"testing"
)

// slopDraft holds one of each slop an agent writes on GitHub.
const slopDraft = "Note that this basically fixes the cache.\n\nIt's not a bug. It's a feature.\n\n🤖 Generated with Claude Code\n"

func Test0142_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "--blob", slopDraft)
	sees(t, out, code, 1, `(description): tutorial-tone: "Note that" opens a sentence like a tutorial`,
		`(description): hedging: "basically" hedges`,
		`(description): rewrite as a direct statement, not the "not X, it's Y"`)
	if strings.Contains(read(out), "decoration") {
		t.Errorf("the attribution footer is no decoration:\n%s", out)
	}
}

func Test0142_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "comment-check", "--blob", "Let's retry once. It's not slow, but careful.\n")
	sees(t, out, code, 1, `tutorial-tone: "Let's" opens a sentence like a tutorial`, `rewrite as a direct statement, not the "not X, it's Y"`)
}

func Test0142_3(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(note): add a note\n\nPreviously it used to miss. It's not a fix. It's a feature.")
	sees(t, out, code, 1, `(commit message): change-narration: "Previously" narrates a change`,
		`(commit message): rewrite as a direct statement, not the "not X, it's Y"`)
}
