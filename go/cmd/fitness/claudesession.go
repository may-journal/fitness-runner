package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
)

// hookLogName is the file in Fitness's cache folder that holds one JSON line
// per judged hook event and one total per ended session.
const hookLogName = "claude-hooks.jsonl"

// hookLogLimit is the size past which the log moves to a .old file and
// starts again, so it never grows without bound.
const hookLogLimit = 4 << 20

// hookLogLine is one judged event in the log.
type hookLogLine struct {
	Time     time.Time `json:"time"`
	Session  string    `json:"session"`
	Event    string    `json:"event"`
	Blocked  bool      `json:"blocked"`
	Findings int       `json:"findings"`
	Ms       int64     `json:"ms"`
	Error    string    `json:"error,omitempty"`
}

// hookTally sums the judged events of one kind in a session.
type hookTally struct {
	Runs     int   `json:"runs"`
	Blocks   int   `json:"blocks"`
	Findings int   `json:"findings"`
	Ms       int64 `json:"ms"`
}

// sessionTotal is the line SessionEnd adds: each event's tally.
type sessionTotal struct {
	Time    time.Time            `json:"time"`
	Session string               `json:"session"`
	Event   string               `json:"event"`
	Hooks   map[string]hookTally `json:"hooks"`
}

// startJob runs the edit checks on the uncommitted files whenever a session
// starts, resumes, clears or compacts, so Claude knows what is red.
// The checks run side by side within their budgets, so a big tree stays fast.
func startJob(in claudeHookInput) (hookJob, bool, error) {
	root, ok := fitnessRoot(in.Cwd)
	if !ok {
		return hookJob{}, false, nil
	}
	scope, err := uncommittedScope(root)
	job := hookJob{root: root, subject: "the uncommitted files", keep: editCheck, args: scope}
	return job, len(scope) > 0, err
}

// subagentJob runs every file and git check on the uncommitted files a
// subagent edited, in the repo of its last edit. It blocks a stop once, and
// passes a subagent that edited nothing.
func subagentJob(in claudeHookInput) (hookJob, bool, error) {
	edited := transcriptEdits(in.AgentTranscript)
	if in.StopActive || len(edited) == 0 {
		return hookJob{}, false, nil
	}
	root, ok := fitnessRoot(filepath.Dir(edited[len(edited)-1]))
	if !ok {
		return hookJob{}, false, nil
	}
	scope, err := uncommittedScope(root)
	scope = slices.DeleteFunc(scope, func(arg string) bool {
		return !slices.Contains(edited, realPath(filepath.Join(root, strings.TrimPrefix(arg, fileFlag+"="))))
	})
	job := hookJob{root: root, subject: "the subagent's changed files", keep: stopCheck, args: scope}
	return job, len(scope) > 0, err
}

// transcriptLine is the part of a transcript line that names a tool call.
type transcriptLine struct {
	Message struct {
		Content []struct {
			Type, Name string
			Input      struct {
				FilePath     string `json:"file_path"`
				NotebookPath string `json:"notebook_path"`
			}
		}
	}
}

// transcriptEdits lists the real paths the edit tools wrote in a transcript,
// in order, or none when it cannot be read.
func transcriptEdits(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var edited []string
	lines := bufio.NewScanner(f)
	lines.Buffer(nil, 64<<20)
	for lines.Scan() {
		edited = append(edited, lineEdits(lines.Bytes())...)
	}
	return edited
}

// lineEdits lists the real paths the edit tool calls of one transcript line
// wrote. A line that is no assistant message names none.
func lineEdits(raw []byte) []string {
	var line transcriptLine
	_ = json.Unmarshal(raw, &line)
	var edited []string
	for _, c := range line.Message.Content {
		path := cmp.Or(c.Input.FilePath, c.Input.NotebookPath)
		if c.Type == "tool_use" && slices.Contains(editTools, c.Name) && path != "" {
			edited = append(edited, realPath(path))
		}
	}
	return edited
}

// verdictFor is the JSON Claude Code reads for event: a denial before a
// tool runs, context at a start, else a block.
func verdictFor(event, reason string) any {
	switch event {
	case "PreToolUse":
		return map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName": event, "permissionDecision": "deny", "permissionDecisionReason": reason,
		}}
	case "SessionStart":
		return map[string]any{"hookSpecificOutput": map[string]string{"hookEventName": event, "additionalContext": reason}}
	}
	return map[string]string{"decision": "block", "reason": reason}
}

// logHookEvent records a judged or failed event, or the session's total when
// it ends. The log is best effort: a failure to write it never fails the hook.
func logHookEvent(in claudeHookInput, result hookResult, judgeErr error, took time.Duration) {
	file, err := hookLogFile()
	switch {
	case err != nil:
	case result.ran || judgeErr != nil:
		appendLogLine(file, hookLogLine{Time: time.Now().UTC(), Session: in.Session, Event: in.Event,
			Blocked: result.reason != "", Findings: result.findings, Ms: took.Milliseconds(), Error: errText(judgeErr)})
	case endsSession(in):
		appendTotal(file, in.Session)
	}
}

// endsSession reports whether the event is the first copy of a named
// session's end.
func endsSession(in claudeHookInput) bool {
	return in.Event == "SessionEnd" && in.Session != "" && firstDelivery(in.raw)
}

// errText is err's message, or "" for none.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// appendTotal adds the session's total per event, when its log reads whole
// and holds any judged event.
func appendTotal(file, session string) {
	sums, err := tallies(file, session)
	if err == nil && len(sums) > 0 {
		appendLogLine(file, sessionTotal{Time: time.Now().UTC(), Session: session, Event: "SessionEnd", Hooks: sums})
	}
}

// hookLogFile is the log's path, its folder made.
func hookLogFile() (string, error) {
	dir, err := distribution.CacheDir()
	if err == nil {
		err = os.MkdirAll(dir, 0o700)
	}
	return filepath.Join(dir, hookLogName), err
}

// appendLogLine adds v as one JSON line. Under a lock, so two hooks never
// rotate at once, it first moves a log past the limit to a .old file.
func appendLogLine(file string, v any) {
	raw, err := json.Marshal(v)
	lock, lockErr := lockFile(file + ".lock")
	if err != nil || lockErr != nil {
		return
	}
	defer lock.Close()
	rotateLog(file)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// rotateLog moves a log past the limit to its .old file.
func rotateLog(file string) {
	if info, err := os.Stat(file); err == nil && info.Size() > hookLogLimit {
		_ = os.Rename(file, file+".old")
	}
}

// tallies sums the judged events of session by event, across the log and
// the .old file it rotated into.
func tallies(file, session string) (map[string]hookTally, error) {
	sums := map[string]hookTally{}
	err := errors.Join(tallyFile(file+".old", session, sums), tallyFile(file, session, sums))
	return sums, err
}

// tallyFile adds the judged events of session in one log file to sums. A
// missing file adds none.
func tallyFile(file, session string, sums map[string]hookTally) error {
	f, err := os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		if line, ok := judgedIn(lines.Bytes(), session); ok {
			sums[line.Event] = sums[line.Event].add(line)
		}
	}
	return lines.Err()
}

// judgedIn decodes a log line and reports whether it is a judged event of
// session.
func judgedIn(raw []byte, session string) (hookLogLine, bool) {
	var line hookLogLine
	err := json.Unmarshal(raw, &line)
	return line, err == nil && line.Session == session && line.Event != "SessionEnd"
}

// add counts one judged event into the tally.
func (t hookTally) add(line hookLogLine) hookTally {
	t.Runs++
	t.Findings += line.Findings
	t.Ms += line.Ms
	if line.Blocked {
		t.Blocks++
	}
	return t
}
