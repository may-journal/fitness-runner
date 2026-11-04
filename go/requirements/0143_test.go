package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startEvent is the SessionStart event from source.
func startEvent(source string) map[string]any {
	return map[string]any{"hook_event_name": "SessionStart", "source": source}
}

func Test0143_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code := hookEvent(t, repo, map[string]any{"hook_event_name": "SubagentStop", "stop_hook_active": false})
	sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on the changed files", "[cspell] notes.md:1")
	out, code = hookEvent(t, repo, map[string]any{"hook_event_name": "SubagentStop", "stop_hook_active": true})
	silentPass(t, out, code)
}

func Test0143_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := hookEvent(t, repo, startEvent("startup"))
	silentPass(t, out, code)
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	for _, source := range []string{"startup", "resume", "compact"} {
		start := time.Now()
		out, code = hookEvent(t, repo, startEvent(source))
		sees(t, out, code, 0, `"hookEventName":"SessionStart"`, `"additionalContext"`, "fitness: cspell failed on the uncommitted files", "[cspell] notes.md:1")
		if took := time.Since(start); took > time.Second {
			t.Errorf("the start hook took %s on %s; it must answer within one second", took, source)
		}
	}
}

func Test0143_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	cache := t.TempDir()
	write(t, repo, map[string]string{"notes.md": misspelledNote, "clean.md": "A short note.\n"})
	for _, event := range []map[string]any{editEvent(repo, "notes.md"), editEvent(repo, "clean.md"), {"hook_event_name": "SessionEnd"}} {
		event["session_id"] = "s1"
		hookEventIn(t, repo, cache, event)
	}
	lines := strings.Split(strings.TrimSpace(fiFile(t, filepath.Join(cache, "claude-hooks.jsonl"))), "\n")
	if !loggedEditsAndTotal(lines) {
		t.Errorf("want two edit lines and a SessionEnd total of 2 runs, 1 block, got:\n%s", strings.Join(lines, "\n"))
	}
	if _, err := os.Stat(filepath.Join(repo, "claude-hooks.jsonl")); err == nil {
		t.Error("the hook log must stay out of the repo")
	}
}

// loggedEditsAndTotal reports whether the log holds two edit lines and then
// a SessionEnd total of two edits, one of them blocked with findings.
func loggedEditsAndTotal(lines []string) bool {
	var total struct {
		Event string
		Hooks map[string]struct{ Runs, Blocks, Findings int }
	}
	if len(lines) != 3 || json.Unmarshal([]byte(lines[2]), &total) != nil {
		return false
	}
	edits := total.Hooks["PostToolUse"]
	return total.Event == "SessionEnd" && edits == struct{ Runs, Blocks, Findings int }{2, 1, 1}
}
