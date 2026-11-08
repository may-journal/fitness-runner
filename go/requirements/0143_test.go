package requirements

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// startEvent is the SessionStart event from source.
func startEvent(source string) map[string]any {
	return map[string]any{"hook_event_name": "SessionStart", "source": source}
}

// editTranscript is a subagent transcript whose Write calls name files.
func editTranscript(t *testing.T, files ...string) string {
	t.Helper()
	transcript := filepath.Join(t.TempDir(), "agent.jsonl")
	lines := `{"type":"user","message":{"content":"Fix the notes."}}` + "\n"
	for _, file := range files {
		lines += fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":%q}}]}}`, file) + "\n"
	}
	mustDo(t, os.WriteFile(transcript, []byte(lines), 0o600))
	return transcript
}

// subagentStop is the SubagentStop event of agent in session s1. Its last
// message differs per try, as each real stop's does.
func subagentStop(agent string, try int, active bool, transcript string) map[string]any {
	return map[string]any{"hook_event_name": "SubagentStop", "session_id": "s1", "agent_id": agent, "stop_hook_active": active,
		"agent_transcript_path": transcript, "last_assistant_message": fmt.Sprintf("Done, try %d.", try)}
}

func Test0143_1(t *testing.T) {
	t.Parallel()
	repo, cache := example(t, "happyRepo", nil), t.TempDir()
	write(t, repo, map[string]string{"notes.md": misspelledNote, "theirs.md": misspelledNote})
	transcript := editTranscript(t, filepath.Join(repo, "notes.md"))
	out, code := hookEventIn(t, repo, cache, subagentStop("a1", 1, false, transcript))
	sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on the subagent's changed files", "[cspell] notes.md:1")
	if strings.Contains(out, "theirs.md") {
		t.Errorf("a subagent was blocked on a file it never edited:\n%s", out)
	}
	for try, active := range []bool{false, true} {
		out, code = hookEventIn(t, repo, cache, subagentStop("a1", try+2, active, transcript))
		silentPass(t, out, code)
	}
	out, code = hookEventIn(t, repo, cache, subagentStop("a2", 1, false, transcript))
	sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
	out, code = hookEventIn(t, repo, cache, subagentStop("a3", 1, false, filepath.Join(t.TempDir(), "none.jsonl")))
	silentPass(t, out, code)
}

// onBranch puts repo on a feature branch off origin/main, which points at
// its current commit.
func onBranch(t *testing.T, repo string) {
	t.Helper()
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "feature")
}

func Test0143_2(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{"theirs.md": misspelledNote})
	onBranch(t, repo)
	out, code := hookEvent(t, repo, startEvent("startup"))
	silentPass(t, out, code)
	commit(t, repo, map[string]string{"committed.md": misspelledNote}, "docs(notes): add a note")
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	runner := filepath.Join(installOnly(t), "fitness")
	best := time.Hour
	for _, source := range []string{"startup", "resume", "compact"} {
		out, code, took := timedHook(t, runner, repo, t.TempDir(), startEvent(source))
		sees(t, out, code, 0, `"hookEventName":"SessionStart"`, `"additionalContext"`, "fitness: cspell failed on the branch's changed files",
			"[cspell] notes.md:1", "[cspell] committed.md:1")
		if strings.Contains(out, "theirs.md") {
			t.Errorf("the start hook judged a file only the default branch changed:\n%s", out)
		}
		best = min(best, took)
	}
	if best > time.Second {
		t.Errorf("the start hook took %s at best; it must answer within one second", best)
	}
}

func Test0143_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	cache := t.TempDir()
	long := fmt.Sprintf(`{"session":"s0","event":"Stop","error":%q}`, strings.Repeat("stderr ", 30000))
	mustDo(t, os.WriteFile(filepath.Join(cache, "claude-hooks.jsonl"), []byte(long+"\nnot json\n"), 0o600))
	write(t, repo, map[string]string{"notes.md": misspelledNote, "clean.md": "A short note.\n"})
	for _, event := range []map[string]any{editEvent(repo, "notes.md"), editEvent(repo, "clean.md"), {"hook_event_name": "SessionEnd"}} {
		event["session_id"] = "s1"
		hookEventIn(t, repo, cache, event)
	}
	lines := strings.Split(strings.TrimSpace(fiFile(t, filepath.Join(cache, "claude-hooks.jsonl"))), "\n")
	if len(lines) != 5 || !loggedEditsAndTotal(lines[2:]) {
		t.Errorf("want two edit lines and a SessionEnd total of 2 runs, 1 block, 1 failed check, 1 finding line, got:\n%s", strings.Join(lines[min(2, len(lines)):], "\n"))
	}
	if _, err := os.Stat(filepath.Join(repo, "claude-hooks.jsonl")); err == nil {
		t.Error("the hook log must stay out of the repo")
	}
}

// hookCounts are a log line's or total's counts.
type hookCounts struct {
	Runs, Blocks int
	Failed       int `json:"failedChecks"`
	Lines        int `json:"findingLines"`
}

// loggedEditsAndTotal reports whether lines are a red and a clean edit and
// then a SessionEnd total of both, one blocked with one finding.
func loggedEditsAndTotal(lines []string) bool {
	var red, clean hookCounts
	var total struct {
		Event string
		Hooks map[string]hookCounts
	}
	err := errors.Join(json.Unmarshal([]byte(lines[0]), &red), json.Unmarshal([]byte(lines[1]), &clean), json.Unmarshal([]byte(lines[2]), &total))
	edits := hookCounts{Runs: 2, Blocks: 1, Failed: 1, Lines: 1}
	return err == nil && red == hookCounts{Failed: 1, Lines: 1} && clean == hookCounts{} && total.Event == "SessionEnd" && total.Hooks["PostToolUse"] == edits
}

func Test0143_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	git(t, repo, "checkout", "-q", "-B", "main")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code := hookEvent(t, repo, startEvent("startup"))
	silentPass(t, out, code)
	git(t, repo, "checkout", "-q", "--detach")
	out, code = hookEvent(t, repo, startEvent("resume"))
	silentPass(t, out, code)
}

// failingGoTest is a Go module whose one test fails wherever go-test runs.
var failingGoTest = map[string]string{
	".fitnessrc.json": `{"policy": "external", "disabledChecks": ["requirements"]}` + "\n",
	"go.mod":          "module example.com/hooked\n\ngo 1.22\n",
	"hooked.go":       "// Package hooked holds one constant.\npackage hooked\n\n// Answer is the answer.\nconst Answer = 42\n",
	"hooked_test.go":  "package hooked\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) { t.Fatal(\"go-test ran\") }\n",
}

func Test0143_5(t *testing.T) {
	t.Parallel()
	installer(t)
	file := filepath.Join(t.TempDir(), "settings.json")
	mustDo(t, os.WriteFile(file, []byte("{}"), 0o600))
	group := claudeSettings(t, file)["hooks"].(map[string]any)["SubagentStop"].([]any)[0]
	if timeout := group.(map[string]any)["hooks"].([]any)[0].(map[string]any)["timeout"]; timeout != 300.0 {
		t.Errorf("SubagentStop's timeout must be 300 seconds, under Claude Code's ten-minute default, got %v", timeout)
	}
	repo := example(t, "happyRepo", failingGoTest)
	write(t, repo, map[string]string{"notes.md": misspelledNote, "hooked_test.go": strings.ReplaceAll(failingGoTest["hooked_test.go"], "ran", "ran again")})
	transcript := editTranscript(t, filepath.Join(repo, "hooked_test.go"), filepath.Join(repo, "notes.md"))
	out, code := hookEvent(t, repo, subagentStop("a1", 1, false, transcript))
	sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
	if strings.Contains(out, "go-test") {
		t.Errorf("a subagent stop ran go-test, which is in the test stage:\n%s", out)
	}
}

// claudeStop is Claude's Stop event in session s1 with its transcript, never
// marked active. Its last message differs per try, as each real stop's does.
func claudeStop(try int, transcript string) map[string]any {
	return map[string]any{"hook_event_name": "Stop", "session_id": "s1", "transcript_path": transcript, "stop_hook_active": false,
		"last_assistant_message": fmt.Sprintf("Done, try %d.", try)}
}

// addUserLine appends a user line of prompt to the transcript, as Claude
// Code writes a prompt or a Stop hook's feedback.
func addUserLine(t *testing.T, transcript, prompt, text string) {
	t.Helper()
	f, err := os.OpenFile(transcript, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	mustDo(t, err)
	defer f.Close()
	_, err = fmt.Fprintf(f, `{"type":"user","promptId":%q,"uuid":"%s-%d","message":{"role":"user","content":%q}}`+"\n", prompt, prompt, len(text), text)
	mustDo(t, err)
}

func Test0143_6(t *testing.T) {
	t.Parallel()
	repo, cache := example(t, "happyRepo", nil), t.TempDir()
	transcript := filepath.Join(t.TempDir(), "session.jsonl")
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	addUserLine(t, transcript, "p1", "Fix the notes.")
	out, code := hookEventIn(t, repo, cache, claudeStop(1, transcript))
	sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
	addUserLine(t, transcript, "p1", "Stop hook feedback:\n"+out)
	out, code = hookEventIn(t, repo, cache, claudeStop(2, transcript))
	silentPass(t, out, code)
	addUserLine(t, transcript, "p2", "Now the readme.")
	out, code = hookEventIn(t, repo, cache, claudeStop(3, transcript))
	sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
}

// slowChecks are local checks, four per processor, that each pass after
// four seconds, so together they outlast the start hook's deadline.
func slowChecks() map[string]string {
	files := map[string]string{".fitnessrc.json": `{"localChecks": "checks"}` + "\n"}
	for i := range 4 * runtime.NumCPU() {
		name := fmt.Sprintf("slow-%d", i)
		files["checks/fitness-check-"+name] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"" + name +
			"\",\"scope\":\"files\"}'; exit 0; }\nsleep 4\necho '" + passing + "'\n"
	}
	return files
}

func Test0143_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", slowChecks())
	onBranch(t, repo)
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code, took := timedHook(t, filepath.Join(installOnly(t), "fitness"), repo, t.TempDir(), startEvent("startup"))
	sees(t, out, code, 0, `"additionalContext"`, "[cspell] notes.md:1", "fitness: skipped slow-", "past the hook's deadline")
	if took > 20*time.Second {
		t.Errorf("the start hook took %s; past its deadline it must answer within 20 seconds", took)
	}
}
