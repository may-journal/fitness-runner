package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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

// startJob runs the edit checks on the uncommitted files as a session starts,
// resumes or compacts, so Claude begins knowing what is red.
func startJob(in claudeHookInput) (hookJob, bool, error) {
	root, ok := fitnessRoot(in.Cwd)
	if !ok {
		return hookJob{}, false, nil
	}
	scope, err := uncommittedScope(root)
	job := hookJob{root: root, subject: "the uncommitted files", keep: editCheck, args: scope}
	return job, len(scope) > 0, err
}

// verdictFor is the JSON Claude Code reads for event: a denial before a
// tool runs, context at a start, else a block.
func verdictFor(event, reason string) any {
	field := map[string]string{"PreToolUse": "permissionDecisionReason", "SessionStart": "additionalContext"}[event]
	if field == "" {
		return map[string]string{"decision": "block", "reason": reason}
	}
	out := map[string]string{"hookEventName": event, field: reason}
	if event == "PreToolUse" {
		out["permissionDecision"] = "deny"
	}
	return map[string]any{"hookSpecificOutput": out}
}

// logHookEvent records a judged event, or the session's total when it ends.
// The log is best effort: a failure to write it never fails the hook.
func logHookEvent(in claudeHookInput, result hookResult, took time.Duration) {
	file, err := hookLogFile()
	switch {
	case err != nil:
	case result.ran:
		appendLogLine(file, hookLogLine{Time: time.Now().UTC(), Session: in.Session, Event: in.Event,
			Blocked: result.reason != "", Findings: result.findings, Ms: took.Milliseconds()})
	case in.Event == "SessionEnd" && firstDelivery(in.raw):
		appendLogLine(file, sessionTotal{Time: time.Now().UTC(), Session: in.Session, Event: in.Event, Hooks: tallies(file, in.Session)})
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

// appendLogLine adds v as one JSON line, first moving a log past the limit
// aside.
func appendLogLine(file string, v any) {
	if info, err := os.Stat(file); err == nil && info.Size() > hookLogLimit {
		_ = os.Rename(file, file+".old")
	}
	raw, err := json.Marshal(v)
	f, openErr := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil || openErr != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// tallies sums the log's judged events of session by event.
func tallies(file, session string) map[string]hookTally {
	sums := map[string]hookTally{}
	f, err := os.Open(file)
	if err != nil {
		return sums
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		if line, ok := judgedIn(lines.Bytes(), session); ok {
			sums[line.Event] = sums[line.Event].add(line)
		}
	}
	return sums
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
