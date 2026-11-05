package main

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// hookLogName is the file in Fitness's cache folder that holds one JSON line
// per judged hook event and one total per ended session.
const hookLogName = "claude-hooks.jsonl"

// hookLogLimit is the size past which the log moves to a .old file and
// starts again, so it never grows without bound.
const hookLogLimit = 4 << 20

// hookLogLine is one judged event in the log.
type hookLogLine struct {
	Time    time.Time `json:"time"`
	Session string    `json:"session"`
	Event   string    `json:"event"`
	Blocked bool      `json:"blocked"`
	// Failed counts the failed checks, and Lines their finding lines.
	Failed int    `json:"failedChecks"`
	Lines  int    `json:"findingLines"`
	Ms     int64  `json:"ms"`
	Error  string `json:"error,omitempty"`
}

// hookTally sums the judged events of one kind in a session.
type hookTally struct {
	Runs   int   `json:"runs"`
	Blocks int   `json:"blocks"`
	Failed int   `json:"failedChecks"`
	Lines  int   `json:"findingLines"`
	Ms     int64 `json:"ms"`
}

// logErrorLimit caps the bytes of a run's error one log line keeps.
const logErrorLimit = 2000

// sessionTotal is the line SessionEnd adds: each event's tally.
type sessionTotal struct {
	Time    time.Time            `json:"time"`
	Session string               `json:"session"`
	Event   string               `json:"event"`
	Hooks   map[string]hookTally `json:"hooks"`
}

// blockMarksName is the folder in Fitness's cache folder that marks each
// block a key allows once.
const blockMarksName = "claude-hook-blocks"

// blockMarkAge is how long a block's mark lasts, which keeps the folder small.
const blockMarkAge = 24 * time.Hour

// subagentChecks is the longest budget a check may declare to run as a
// subagent stops. Slower checks, such as the test suites, wait for Claude's stop.
const subagentChecks = 4 * time.Minute

// subagentDeadline cuts the subagent's checks inside the hook's 300 s timeout,
// so its findings still reach Claude.
const subagentDeadline = subagentChecks + 30*time.Second

// startDeadline cuts the start's checks well inside the hook's 30 s timeout.
const startDeadline = 15 * time.Second

// startFiles caps the files a session start judges.
const startFiles = 200

// startJob runs the edit checks on the files the session's branch changed,
// committed or not, whenever a session starts, resumes, clears or compacts.
// On the default branch or a detached HEAD it adds nothing.
func startJob(in claudeHookInput) (hookJob, bool, error) {
	root, ok := fitnessRoot(in.Cwd)
	if !ok || gitx.FeatureBranch(root) == "" {
		return hookJob{}, false, nil
	}
	scope, err := branchScope(root)
	subject := "the branch's changed files"
	if len(scope) > startFiles {
		subject = fmt.Sprintf("the first %d of the branch's %d changed files", startFiles, len(scope))
		scope = scope[:startFiles]
	}
	job := hookJob{root: root, subject: subject, keep: editCheck, args: scope, deadline: startDeadline}
	return job, len(scope) > 0, err
}

// branchScope is a `--file` argument per existing file that HEAD changes
// since its merge base with origin's default branch, or leaves uncommitted.
func branchScope(root string) ([]string, error) {
	repo := gitx.Repo(root)
	committed, err := repo.ChangedSince(gitx.DefaultBase(root))
	uncommitted, err2 := repo.Uncommitted()
	return fileScope(root, append(committed, uncommitted...)), errors.Join(err, err2)
}

// subagentJob runs the file and git checks that fit a subagent stop on the
// uncommitted files it edited, in its last edit's repo. It blocks each
// subagent once, and passes one that edited nothing.
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
	once := "SubagentStop\x00" + in.Session + "\x00" + cmp.Or(in.AgentID, in.AgentTranscript)
	job := hookJob{root: root, subject: "the subagent's changed files", keep: subagentCheck, args: scope,
		once: once, deadline: subagentDeadline}
	return job, len(scope) > 0, err
}

// subagentCheck keeps the stop's checks whose budget fits a subagent stop.
func subagentCheck(c listed) bool {
	return stopCheck(c) && c.BudgetMs <= subagentChecks.Milliseconds()
}

// blockOnce passes a block whose key Fitness already recorded, and records a
// new one. Claude Code may not mark a repeat stop active, so this stops a loop.
func (job hookJob) blockOnce(result hookResult) hookResult {
	if job.once != "" && result.reason != "" && !firstBlock(job.once) {
		result.reason = ""
	}
	return result
}

// firstBlock records key in the cache folder and reports whether it was new.
// When no mark can be kept, every block counts as the first.
func firstBlock(key string) bool {
	cache, err := distribution.CacheDir()
	dir := filepath.Join(cache, blockMarksName)
	if err != nil || os.MkdirAll(dir, 0o700) != nil {
		return true
	}
	defer pruneMarks(dir, blockMarkAge)
	sum := sha256.Sum256([]byte(key))
	mark, err := os.OpenFile(filepath.Join(dir, hex.EncodeToString(sum[:])), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err == nil {
		_ = mark.Close()
	}
	return !errors.Is(err, os.ErrExist)
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
	var edited []string
	_ = eachLine(path, func(raw []byte) { edited = append(edited, lineEdits(raw)...) })
	return edited
}

// promptLine is the part of a transcript line that names its turn.
type promptLine struct {
	Type     string `json:"type"`
	PromptID string `json:"promptId"`
	UUID     string `json:"uuid"`
	Message  struct {
		Content any `json:"content"`
	} `json:"message"`
}

// feedbackPrefix starts the user line that carries a Stop hook's block.
const feedbackPrefix = "Stop hook feedback:"

// turnOf names the turn a transcript is in: its last prompt's ID, which a
// Stop hook's feedback shares. Without IDs it is the last typed prompt's uuid.
func turnOf(path string) string {
	turn := ""
	_ = eachLine(path, func(raw []byte) {
		var line promptLine
		if json.Unmarshal(raw, &line) == nil && line.Type == "user" {
			turn = cmp.Or(line.PromptID, typedPrompt(line), turn)
		}
	})
	return turn
}

// typedPrompt is the uuid of a user line that holds a prompt, or "" for a
// tool result or a Stop hook's feedback.
func typedPrompt(line promptLine) string {
	text, ok := line.Message.Content.(string)
	if !ok || strings.HasPrefix(text, feedbackPrefix) {
		return ""
	}
	return line.UUID
}

// eachLine calls fn with each line of the file at path, however long. It
// returns the error that stopped reading, or nil at the end.
func eachLine(path string, fn func([]byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			fn(line)
		}
		if err != nil {
			return ignoreEOF(err)
		}
	}
}

// ignoreEOF is err, or nil at the end of the input.
func ignoreEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
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
			Blocked: result.reason != "", Failed: result.failed, Lines: result.lines, Ms: took.Milliseconds(), Error: errText(judgeErr)})
	case endsSession(in):
		appendTotal(file, in.Session)
	}
}

// endsSession reports whether the event is the first copy of a named
// session's end.
func endsSession(in claudeHookInput) bool {
	return in.Event == "SessionEnd" && in.Session != "" && firstDelivery(in.raw)
}

// errText is err's message, cut to the log's limit, or "" for none.
func errText(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	if len(text) > logErrorLimit {
		text = strings.ToValidUTF8(text[:logErrorLimit], "") + "…"
	}
	return text
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
	err := eachLine(file, func(raw []byte) {
		if line, ok := judgedIn(raw, session); ok {
			sums[line.Event] = sums[line.Event].add(line)
		}
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
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
	t.Failed += line.Failed
	t.Lines += line.Lines
	t.Ms += line.Ms
	if line.Blocked {
		t.Blocks++
	}
	return t
}
