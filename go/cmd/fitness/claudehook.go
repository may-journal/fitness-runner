package main

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// claudeHookInput is the part of a Claude Code hook's stdin JSON the hook
// reads.
type claudeHookInput struct {
	Event string `json:"hook_event_name"`
	Cwd   string `json:"cwd"`
	Tool  string `json:"tool_name"`
	Input struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
		Command      string `json:"command"`
	} `json:"tool_input"`
	// Session and Source name the session and, for SessionStart, whether it
	// started, resumed, cleared or compacted.
	Session string `json:"session_id"`
	Source  string `json:"source"`
	// StopActive is true when Claude is already continuing because a Stop
	// hook blocked it, so the hook lets it stop this time.
	StopActive bool `json:"stop_hook_active"`
	// raw is the event as Claude Code sent it, which marks its delivery.
	raw []byte
}

// hookJob is what one event asks the hook to judge. It holds the repo, the
// subject's name, the checks, the runner arguments and any stdin.
type hookJob struct {
	root, subject string
	keep          func(listed) bool
	args          []string
	stdin         string
}

// editTools are the Claude tools whose edit the PostToolUse hook judges.
var editTools = []string{"Edit", "Write", "MultiEdit", "NotebookEdit"}

// findingLines caps the finding lines one verdict hands Claude.
const findingLines = 40

// duplicateWindow is how long an identical hook input counts as the same
// event, which the skill's hooks and the installed settings both received.
const duplicateWindow = 5 * time.Second

// runClaudeHook judges what one Claude Code hook event on stdin names. A
// red verdict blocks with the findings; anything else exits 0 silently.
func runClaudeHook(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "fitness claude-hook: takes no arguments; Claude Code sends the event on stdin")
		return usageExit
	}
	withBundlePath()
	start := time.Now()
	in, err := readHookInput(os.Stdin)
	var result hookResult
	if err == nil {
		result, err = judgeHookEvent(in)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness claude-hook: "+err.Error())
		return 1
	}
	logHookEvent(in, result, time.Since(start))
	return writeBlock(os.Stdout, in.Event, result.reason)
}

// withBundlePath puts this fitness's folder first on PATH, as the installer
// does, so checks find the tools the bundle ships.
func withBundlePath() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Setenv("PATH", filepath.Dir(exe)+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
}

// readHookInput decodes the hook event JSON from r.
func readHookInput(r io.Reader) (claudeHookInput, error) {
	var in claudeHookInput
	raw, err := io.ReadAll(r)
	if err == nil {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		return in, fmt.Errorf("reading the hook event: %w", err)
	}
	in.raw = raw
	return in, nil
}

// hookResult is what judging one event found: the findings that block it,
// or "" when it passes, their count, and whether any check ran.
type hookResult struct {
	reason   string
	findings int
	ran      bool
}

// judgeHookEvent judges the event, or passes it when it names nothing
// Fitness judges.
func judgeHookEvent(in claudeHookInput) (hookResult, error) {
	job, ok, err := hookJobFor(in)
	if err != nil || !ok || !firstDelivery(in.raw) {
		return hookResult{}, err
	}
	return job.judge()
}

// hookJobFor picks the job an event asks for, or false when it asks none.
func hookJobFor(in claudeHookInput) (hookJob, bool, error) {
	switch in.Event {
	case "PostToolUse":
		return editJob(in)
	case "PreToolUse":
		return postJob(in)
	case "Stop", "SubagentStop":
		return stopJob(in)
	case "SessionStart":
		return startJob(in)
	}
	return hookJob{}, false, nil
}

// editJob runs the fast file checks on the file an edit tool wrote.
func editJob(in claudeHookInput) (hookJob, bool, error) {
	path := editedPath(in)
	if path == "" {
		return hookJob{}, false, nil
	}
	root, ok := fitnessRoot(filepath.Dir(path))
	if !ok {
		return hookJob{}, false, nil
	}
	if _, err := namedFiles(root, []string{fileFlag + "=" + path}); err != nil {
		return hookJob{}, false, nil
	}
	rel, _ := filepath.Rel(realPath(root), realPath(path))
	return hookJob{root: root, subject: filepath.ToSlash(rel), keep: editCheck, args: []string{fileFlag + "=" + path}}, true, nil
}

// editedPath is the absolute path an edit tool wrote, or "" for another tool.
func editedPath(in claudeHookInput) string {
	path := cmp.Or(in.Input.FilePath, in.Input.NotebookPath)
	if !slices.Contains(editTools, in.Tool) || path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(in.Cwd, path)
}

// postJob judges the body of a gh issue, PR or comment call before it posts.
func postJob(in claudeHookInput) (hookJob, bool, error) {
	if in.Tool != "Bash" {
		return hookJob{}, false, nil
	}
	post, ok := ghPostIn(in.Input.Command, in.Cwd)
	if !ok {
		return hookJob{}, false, nil
	}
	root, ok := fitnessRoot(in.Cwd)
	args := []string{blobFlag, "-"}
	if post.comment {
		args = append(args, commentChecks)
	}
	return hookJob{root: root, subject: "the " + post.kind + " body", args: args, stdin: post.body}, ok, nil
}

// stopJob runs every file and git check on the uncommitted files, as Claude
// or a subagent stops. It blocks a stop once: a second try while a stop hook
// already blocked passes.
func stopJob(in claudeHookInput) (hookJob, bool, error) {
	if in.StopActive {
		return hookJob{}, false, nil
	}
	root, ok := fitnessRoot(in.Cwd)
	if !ok {
		return hookJob{}, false, nil
	}
	scope, err := uncommittedScope(root)
	job := hookJob{root: root, subject: "the changed files", keep: stopCheck, args: scope}
	return job, len(scope) > 0, err
}

// uncommittedScope is a `--file` argument per uncommitted file that exists.
func uncommittedScope(root string) ([]string, error) {
	changed, err := gitx.Repo(root).Uncommitted()
	var scope []string
	for _, file := range changed {
		if isFile(filepath.Join(root, file)) {
			scope = append(scope, fileFlag+"="+file)
		}
	}
	return scope, err
}

// editCheck keeps the checks an edit runs: file checks within the default
// budget. Tests, vet, linters and network checks declare more and wait for
// the stop.
func editCheck(c listed) bool {
	return c.Scope == checkkit.ScopeFiles && c.BudgetMs <= defaultTimeout.Milliseconds()
}

// stopCheck keeps the checks a stop runs: every file and git check.
func stopCheck(c listed) bool {
	return c.Scope == checkkit.ScopeFiles || c.Scope == checkkit.ScopeGit
}

// judge runs the job's checks and returns their findings, with no reason
// when every check passed or none applies.
func (job hookJob) judge() (hookResult, error) {
	args := job.args
	if job.keep != nil {
		names, err := hookChecks(job.root, job.keep)
		if err != nil || len(names) == 0 {
			return hookResult{}, err
		}
		args = append(args, "--policy=external", "--checks="+strings.Join(names, ","))
	}
	report, err := runChild(job.root, job.stdin, args)
	_, lines := failedChecks(report)
	return hookResult{reason: findings(job.subject, report), findings: len(lines), ran: true}, err
}

// hookChecks names the checks the repo enables that keep accepts.
func hookChecks(root string, keep func(listed) bool) ([]string, error) {
	err := cmp.Or(os.Chdir(root), installErr)
	if err != nil {
		return nil, err
	}
	all, err := listChecks()
	var names []string
	for _, c := range all {
		if runnable(c) && keep(c) {
			names = append(names, c.Name)
		}
	}
	return names, err
}

// runnable reports whether the repo enables a check that is installed.
func runnable(c listed) bool {
	return c.Enabled && c.Note == ""
}

// runChild runs this fitness in root with args, feeding it stdin, and
// returns its JSON report.
func runChild(root, stdin string, args []string) (runReport, error) {
	var result runReport
	exe, err := os.Executable()
	if err != nil {
		return result, err
	}
	cmd := exec.Command(exe, append([]string{formatFlag + "=json"}, args...)...)
	cmd.Dir, cmd.Stdin = root, strings.NewReader(stdin)
	cmd.Env = append(withoutVar(withoutVar(os.Environ(), "FORCE_COLOR"), "GITHUB_ACTIONS"), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	if json.Unmarshal(stdout.Bytes(), &result) != nil {
		return result, errors.New("fitness gave no report: " + strings.TrimSpace(cmp.Or(stderr.String(), fmt.Sprint(runErr))))
	}
	return result, nil
}

// findings renders the failed checks of a report for Claude, or "" when
// none failed.
func findings(subject string, result runReport) string {
	failed, lines := failedChecks(result)
	if len(failed) == 0 {
		return ""
	}
	if len(lines) > findingLines {
		lines = append(lines[:findingLines], fmt.Sprintf("… and %d more", len(lines)-findingLines))
	}
	head := fmt.Sprintf("fitness: %s failed on %s.", strings.Join(failed, ", "), subject)
	tail := "Fix what they name, never the check; `fitness explain <check>` says why each exists."
	return head + "\n" + strings.Join(lines, "\n") + "\n" + tail
}

// failedChecks names the failed checks of a report and their error lines.
func failedChecks(result runReport) (failed, lines []string) {
	for _, c := range result.Checks {
		if c.Status != "failed" {
			continue
		}
		failed = append(failed, c.Name)
		for _, e := range c.Errors {
			lines = append(lines, "["+c.Name+"] "+e)
		}
	}
	return failed, lines
}

// writeBlock prints the JSON that blocks event with reason, in the shape
// Claude Code reads for it. An empty reason blocks nothing.
func writeBlock(out io.Writer, event, reason string) int {
	if reason == "" {
		return 0
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(verdictFor(event, reason)); err != nil {
		fmt.Fprintln(os.Stderr, "fitness claude-hook: "+err.Error())
		return 1
	}
	return 0
}

// fitnessRoot returns the git root above dir when that repo uses fitness:
// it has a config file or its git hooks run fitness.
func fitnessRoot(dir string) (string, bool) {
	root, err := gitx.Repo(dir).Toplevel()
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(root, conf.FileName)); err == nil {
		return root, true
	}
	return root, hooksRunFitness(root)
}

// hooksRunFitness reports whether the repo's pre-commit hook runs fitness.
func hooksRunFitness(root string) bool {
	hooksPath, err := gitx.Repo(root).Config("core.hooksPath")
	if err != nil || hooksPath == "" {
		return false
	}
	if !filepath.IsAbs(hooksPath) {
		hooksPath = filepath.Join(root, hooksPath)
	}
	shim, err := os.ReadFile(filepath.Join(hooksPath, "pre-commit"))
	return err == nil && bytes.Contains(shim, []byte("fitness"))
}

// firstDelivery reports whether this is the first copy of a hook event, as
// the skill's hooks and the settings may both run one.
func firstDelivery(raw []byte) bool {
	dir := filepath.Join(os.TempDir(), "fitness-claude-hook")
	if os.MkdirAll(dir, 0o700) != nil {
		return true
	}
	sum := sha256.Sum256(raw)
	mark := filepath.Join(dir, hex.EncodeToString(sum[:]))
	lock, err := os.OpenFile(mark+".lock", os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return true
	}
	defer lock.Close()
	// The lock makes reading and renewing the mark one step, so of two
	// copies at once exactly one finds no fresh mark.
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX) != nil {
		return true
	}
	defer pruneMarks(dir)
	return renewMark(mark)
}

// renewMark reports whether mark was missing or older than the duplicate
// window, and stamps it now.
func renewMark(mark string) bool {
	info, err := os.Stat(mark)
	if err == nil && time.Since(info.ModTime()) < duplicateWindow {
		return false
	}
	_ = os.WriteFile(mark, nil, 0o600)
	now := time.Now()
	_ = os.Chtimes(mark, now, now)
	return true
}

// pruneMarks deletes the marks and locks older than twice the duplicate
// window, which no copy of an event still reads.
func pruneMarks(dir string) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil && time.Since(info.ModTime()) > 2*duplicateWindow {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
