// Command fitness runs the configured check suite: it resolves check
// binaries, execs them in a bounded parallel pool with per-check timeouts,
// and renders the results table. See docs/plans/archive/01-go-rewrite.md.
package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/render"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// allChecks is the org default and the catalog for explicit external lists.
// Org checks self-gate when their language, tool, config, or input is absent.
// External policy also defaults to the full catalog, with optional selection.
// It is every check installed beside the runner, minus the ignored ones, in
// the order the language registry gives: repo-wide checks, then each module's.
// installErr says why no checks were found; a run stops on it.
var allChecks, installErr = catalog()

// catalog lists the default checks, or the error that leaves it empty.
func catalog() ([]string, error) {
	installed, err := checkkit.Installed()
	if err != nil {
		return nil, err
	}
	return lang.Catalog(installed, langs.All), nil
}

// resolved is one runnable check: its binary and describe metadata.
type resolved struct {
	name string
	bin  string
	desc checkkit.Describe
	// describeErr says why --describe gave no metadata; the check still
	// runs with defaults and its row carries this as info.
	describeErr string
	// overran says the handshake ran past its budget, so it gets one more
	// try once the crowd of handshakes is done.
	overran bool
	// declaredMs is the budget the check declares itself, before the repo
	// config fills an empty one; the pool starts the largest first.
	declaredMs int
}

// outcome is one finished check, in dispatch order.
type outcome struct {
	res    checkkit.Result
	ms     int64
	stderr []byte
	// timedOut marks a budget kill; crashed says how a check that printed
	// no result ended, exit is its protocol exit code, and startErr says why
	// the check could not start.
	timedOut bool
	crashed  string
	exit     int
	startErr string
}

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// dispatch routes help and the subcommands (`init`, `hook <name>`,
// `pr-check`, `issue-check`, `plan-check`, `close-check`, `comment-check`) before falling through to the check runner, so the
// git hooks and workflow steps stay one line.
func dispatch(args []string) int {
	if isHelp(args) {
		return printUsage()
	}
	return route(args)
}

// isHelp reports whether the first argument asks for usage.
func isHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")
}

// subcommands maps each subcommand name to its handler, which gets the
// arguments after the name.
var subcommands = map[string]func(args []string) int{
	"init":          runInit,
	"hook":          runHookArgs,
	"pr-check":      func([]string) int { return metadata.RunGH("pr-check") },
	"issue-check":   func([]string) int { return metadata.RunGH("issue-check") },
	"plan-check":    func([]string) int { return metadata.RunGH("plan-check") },
	"close-check":   func([]string) int { return metadata.RunGH("close-check") },
	"comment-check": runCommentCheck,
	"branch-check":  func(args []string) int { return run(append([]string{"local-branch"}, args...)) },
}

// route dispatches the subcommands, defaulting to the check runner.
func route(args []string) int {
	if len(args) > 0 {
		if handler, ok := subcommands[args[0]]; ok {
			return handler(args[1:])
		}
	}
	return run(args)
}

// runHookArgs runs `fitness hook <name> [args]`; a bare `hook` with no name
// falls through to the check runner, as before.
func runHookArgs(args []string) int {
	if len(args) == 0 {
		return run([]string{"hook"})
	}
	return runHook(args[0], args[1:])
}

// printUsage writes the command summary.
func printUsage() int {
	fmt.Println(`fitness — run the configured checks.

Usage:
  fitness                  run the full configured suite
  fitness <name>           run one check by name
  fitness --policy=external --checks=name,name
                           run exactly the selected checks
  fitness --all            scan every file, even with files staged
  fitness --blob <text>    run every blob-capable check on text (- reads stdin)
  fitness init             install the shared git hooks into this repo
  fitness hook <name>      run a git hook (commit-msg | pre-commit | pre-push)
  fitness pr-check         validate PR titles and descriptions (GitHub Actions)
  fitness branch-check     validate this branch's PR and linked issues locally
  fitness issue-check      validate an issue by its kind label and comment the result (GitHub Actions)
  fitness plan-check       issue-check for Plan issues only (GitHub Actions)
  fitness close-check      reopen an issue closed with unchecked items (GitHub Actions)
  fitness comment-check --blob <text>
                           judge a drafted comment (- or no arguments reads stdin)
  fitness comment-check    judge issue and PR comments and reply (GitHub Actions)

Docs: https://github.com/may-journal/fitness-runner#usage`)
	return 0
}

func run(argv []string) int {
	if installErr != nil {
		report.Error("fitness setup", installErr)
		return 1
	}
	if hasBlobFlag(argv) {
		return runBlob(argv)
	}
	root, err := os.Getwd()
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	checks, passthrough, err := selectedChecks(root, argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	env, passthrough := singleCheckArgs(root, checks, passthrough, hasAllFlag(argv))
	checks = commitChecks(checks)

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env)

	rows, totals := summarize(checks, outcomes)
	totals.Ms = time.Since(start).Milliseconds()
	printSummary(rows, totals)
	return finish(rows, totals)
}

// loadConfig loads the repo config.
func loadConfig(root string) (*conf.Config, error) {
	return conf.Load(root)
}

// prepareChecks resolves the check list, turning an empty resolution into
// the Unknown-check error the CLI renders.
func prepareChecks(cfg *conf.Config, spec string) ([]resolved, error) {
	checks, err := resolveChecks(spec)
	if err != nil {
		return nil, err
	}
	if len(checks) == 0 {
		which := spec
		if which == "" {
			which = "(none)"
		}
		return nil, errors.New("Unknown check: " + which)
	}
	return withBudgets(cfg, checks), nil
}

// singleCheckArgs builds the check env and forwarded args. Passthrough and
// context-inline apply only to a single-check run: multi-check runs drop
// passthrough entirely.
func singleCheckArgs(root string, checks []resolved, passthrough []string, all bool) (env, remaining []string) {
	env = contextEnv(root, checks, all)
	if len(checks) != 1 {
		return env, nil
	}
	arg := checks[0].desc.ContextInlineArg
	if arg == "" {
		return env, passthrough
	}
	value, remaining, found := extractInline(passthrough, arg)
	if found {
		env = append(env, "FITNESS_CTX_MESSAGE="+value)
	}
	return env, remaining
}

// summarize converts outcomes to table rows, streaming each check's stderr
// in dispatch order, and tallies the totals-line inputs.
func summarize(checks []resolved, outcomes []outcome) ([]render.Row, render.Totals) {
	rows := make([]render.Row, len(checks))
	var totals render.Totals
	for i, c := range checks {
		os.Stderr.Write(outcomes[i].stderr)
		rows[i] = rowFor(c, outcomes[i])
		totals.Add(rows[i])
	}
	return rows, totals
}

// rowFor builds the table row for one outcome. A budget kill fails as timed
// out. A check that could not judge (it did not start, printed no result, or
// exited other than 0 or 1) becomes an info row, which never fails the run.
func rowFor(c resolved, o outcome) render.Row {
	row := render.Row{Name: c.name, Ok: o.res.Ok, FilesChecked: o.res.FilesChecked, Ms: o.ms, Errors: o.res.Errors, Info: o.res.Info}
	if o.timedOut {
		row = render.Row{Name: c.name, FilesChecked: -1, Ms: o.ms,
			Errors: []string{fmt.Sprintf("Check timed out after %ss", trimFloat(timeoutFor(c).Seconds()))}}
	} else if note, internal := internalNote(o); internal {
		row = render.Row{Name: c.name, Ok: true, FilesChecked: -1, Ms: o.ms, Info: appendNote(o.res.Info, note)}
	}
	row.Info = appendNote(row.Info, c.describeErr)
	return row
}

// internalNote reports whether the check could not judge and says why. A
// printed result with ok false is a finding whatever the exit code, and a
// check that exits 3 with its own info needs no further note.
func internalNote(o outcome) (string, bool) {
	switch {
	case o.startErr != "":
		return "check could not start: " + o.startErr, true
	case o.crashed != "":
		return "check ended without a result (" + o.crashed + ")", true
	case judged(o.exit) || !o.res.Ok:
		return "", false
	}
	return exitNote(o), true
}

// exitNote says why a check that printed a result exited other than 0 or 1;
// exit 3 with the check's own info needs no further note.
func exitNote(o outcome) string {
	if o.exit == checkkit.ExitInternal && len(o.res.Info) > 0 {
		return ""
	}
	return fmt.Sprintf("check could not judge (exit %d)", o.exit)
}

// judged reports whether a protocol exit code carries a verdict: 0 passed,
// 1 found problems.
func judged(exit int) bool {
	return exit == 0 || exit == 1
}

// appendNote appends note to notes unless it is empty.
func appendNote(notes []string, note string) []string {
	if note == "" {
		return notes
	}
	return append(append([]string{}, notes...), note)
}

// printSummary renders the table and total line — to stderr when any check
// failed, so failure output stays together.
func printSummary(rows []render.Row, totals render.Totals) {
	p := render.NewPalette()
	table := render.Table(rows, render.TermCols(), p)
	total := render.TotalLine(totals, p)
	out := os.Stdout
	if totals.Failure > 0 {
		out = os.Stderr
	}
	fmt.Fprintln(out, table)
	fmt.Fprintln(out, total)
}

// parseArgv returns the check named by the first non-flag argument and the
// args forwarded to it: everything after its name except --all. With no
// name, the full suite runs and nothing is forwarded.
func parseArgv(argv []string) (spec string, passthrough []string) {
	for i, a := range argv {
		if a != "" && !strings.HasPrefix(a, "-") {
			return a, withoutAll(argv[i+1:])
		}
	}
	return "", nil
}

// withoutAll drops the runner's own --all flag from args.
func withoutAll(args []string) []string {
	var out []string
	for _, a := range args {
		if a != allFlag {
			out = append(out, a)
		}
	}
	return out
}

// resolveChecks builds the check list: the one named on the CLI, else every
// check. No repo turns a check off; each passes clean where it does not apply.
func resolveChecks(spec string) ([]resolved, error) {
	if spec != "" {
		return resolveSingle(spec)
	}
	return resolveList(allChecks)
}

// resolveSingle resolves a CLI check name into a one-element list; an empty
// list means Unknown check.
func resolveSingle(spec string) ([]resolved, error) {
	checks, err := resolveList([]string{spec})
	if errors.Is(err, errNotInstalled) {
		return nil, nil
	}
	return checks, err
}

// errNotInstalled marks a name with no check binary beside the runner.
var errNotInstalled = errors.New("Check binary not installed")

// resolveList resolves each name, in order. Every binary is found first, so
// a missing one fails before any process starts. The --describe handshakes
// then run side by side and come back in catalog order. One that overran in
// that crowd gets a second try alone, so a slow first start after an install
// does not drop a check's own budget.
func resolveList(names []string) ([]resolved, error) {
	bins := make([]string, len(names))
	for i, name := range names {
		bin, err := checkkit.Binary(name)
		if err != nil {
			return nil, fmt.Errorf("%w: fitness-check-%s", errNotInstalled, name)
		}
		bins[i] = bin
	}
	checks := par.Map(len(names), par.Processes(), func(i int) resolved {
		return describeCheck(names[i], bins[i])
	})
	if len(checks) > 1 {
		retryOverran(checks)
	}
	return checks, nil
}

// retryOverran runs each handshake that overran in the crowd once more, alone.
func retryOverran(checks []resolved) {
	for i, c := range checks {
		if c.overran {
			checks[i] = describeCheck(c.name, c.bin)
		}
	}
}

// describeCheck runs one check's --describe handshake.
func describeCheck(name, bin string) resolved {
	desc, describeErr, overran := describe(bin)
	return resolved{name: name, bin: bin, desc: desc, describeErr: describeErr, overran: overran, declaredMs: desc.TimeoutMs}
}

// describe asks a check binary for its metadata. On any failure it returns
// the zero value, so the check runs with defaults, and an info note saying
// why. The handshake gets its own short budget and process-group kill — a
// local script that ignores --describe and starts real work must not hang
// resolution. overran reports a handshake killed at its budget.
func describe(bin string) (d checkkit.Describe, note string, overran bool) {
	const failed = "--describe gave no metadata, so the check ran with defaults: "
	cmd := exec.Command(bin, "--describe")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return d, failed + err.Error(), false
	}
	if ok, overran := waitDescribe(cmd); !ok {
		return d, failed + "it failed or ran past 2s", overran
	}
	if err := json.Unmarshal(checkkit.LastJSONLine(stdout.Bytes()), &d); err != nil {
		return checkkit.Describe{}, failed + err.Error(), false
	}
	return d, "", false
}

// waitDescribe waits for the handshake to exit cleanly within its short
// budget, killing the process group on overrun. ok false means no usable
// output, and overran says the budget ran out.
func waitDescribe(cmd *exec.Cmd) (ok, overran bool) {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err == nil, false
	case <-time.After(2 * time.Second):
		killGroup(cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return false, true
	}
}

// contextEnv is the environment every check receives. FITNESS_CHANGED_FILES
// is set only for a scoped run, so an inherited value never leaks into a
// full scan.
func contextEnv(root string, checks []resolved, all bool) []string {
	env := withoutVar(os.Environ(), "FITNESS_CHANGED_FILES")
	env = append(env, "FITNESS_STAGED_FILES="+strings.Join(gitx.StagedFiles(root), "\n"))
	if changed := changedScope(root, all); len(changed) > 0 {
		env = append(env, "FITNESS_CHANGED_FILES="+strings.Join(changed, "\n"))
	}
	names := make([]string, len(checks))
	for i, c := range checks {
		names[i] = c.name
	}
	env = append(env, "FITNESS_ENABLED_CHECKS="+strings.Join(names, "\n"))
	return env
}

// extractInline pulls the first `--arg=value` or `--arg value` out of args
// (a following token starting with '-' means present-but-empty).
func extractInline(args []string, argName string) (value string, remaining []string, found bool) {
	for i := 0; i < len(args); i++ {
		if !found {
			if v, consumed := inlineMatch(args, i, argName); consumed > 0 {
				value, found = v, true
				i += consumed - 1
				continue
			}
		}
		remaining = append(remaining, args[i])
	}
	return value, remaining, found
}

// inlineMatch matches args[i] against `--arg=value` or `--arg value`,
// returning the value and how many tokens the match consumed (0 means no
// match; a lone `--arg` consumes 1 with an empty value).
func inlineMatch(args []string, i int, argName string) (value string, consumed int) {
	a := args[i]
	if strings.HasPrefix(a, argName+"=") {
		return strings.TrimPrefix(a, argName+"="), 1
	}
	if a != argName {
		return "", 0
	}
	if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
		return args[i+1], 2
	}
	return "", 1
}

// runPool executes every check with bounded parallelism, printing the
// progress line as each starts. Checks start by the budget they declare,
// largest first, so a slow one never waits behind the quick ones; outcomes
// stay in catalog order.
func runPool(root string, checks []resolved, passthrough, env []string) []outcome {
	outcomes := make([]outcome, len(checks))
	sem := make(chan struct{}, par.Processes())
	var wg sync.WaitGroup
	for _, i := range startOrder(checks) {
		c := checks[i]
		sem <- struct{}{}
		fmt.Fprintf(os.Stderr, "  → %s\n", c.name)
		wg.Add(1)
		go func(i int, c resolved) {
			defer wg.Done()
			defer func() { <-sem }()
			outcomes[i] = runOne(root, c, passthrough, env)
		}(i, c)
	}
	wg.Wait()
	return outcomes
}

// startOrder lists check indices by descending declared budget. A budget
// from the repo config says nothing about cost, so those checks count as
// declaring none, and equal budgets keep catalog order.
func startOrder(checks []resolved) []int {
	order := make([]int, len(checks))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(checks[b].declaredMs, checks[a].declaredMs)
	})
	return order
}

// runOne execs a single check in its own process group with a timeout that
// kills the whole group (TERM, then KILL after a grace period).
func runOne(root string, c resolved, passthrough, env []string) outcome {
	args := append([]string{"--root", root}, passthrough...)
	cmd := exec.Command(c.bin, args...)
	cmd.Dir = root
	cmd.Env = append(append([]string{}, env...), "FITNESS_CHECK_NAME="+c.name)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return outcome{startErr: err.Error()}
	}
	timedOut := waitWithBudget(cmd, timeoutFor(c))
	ms := time.Since(start).Milliseconds()

	o := outcome{ms: ms, stderr: stderr.Bytes(), timedOut: timedOut}
	if timedOut {
		return o
	}
	return decodeOutcome(o, cmd, stdout.Bytes())
}

// waitWithBudget waits for cmd within budget; on overrun it TERMs the
// process group, then KILLs the group after a grace period. Reports whether the check
// timed out.
func waitWithBudget(cmd *exec.Cmd, budget time.Duration) bool {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		return false
	case <-time.After(budget):
	}
	killGroup(cmd.Process.Pid, syscall.SIGTERM)
	if exitedWithin(done, 2*time.Second) {
		// The check is gone, but a tool it started may ignore the stop
		// signal; kill whatever is left in its group.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return true
	}
	killGroup(cmd.Process.Pid, syscall.SIGKILL)
	<-done
	return true
}

// exitedWithin reports whether the check exits within grace.
func exitedWithin(done <-chan error, grace time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(grace):
		return false
	}
}

// decodeOutcome parses the check's protocol result from stdout and keeps its
// exit code; a non-JSON tail marks the outcome crashed with how the process
// ended, such as "exit status 2" or "signal: killed".
func decodeOutcome(o outcome, cmd *exec.Cmd, stdout []byte) outcome {
	o.exit = cmd.ProcessState.ExitCode()
	var res checkkit.Result
	if err := json.Unmarshal(checkkit.LastJSONLine(stdout), &res); err != nil {
		o.crashed = cmd.ProcessState.String()
		return o
	}
	o.res = res
	return o
}

// killGroup signals the whole process group, falling back to the pid.
func killGroup(pid int, sig syscall.Signal) {
	if err := syscall.Kill(-pid, sig); err != nil {
		_ = syscall.Kill(pid, sig)
	}
}

// trimFloat renders seconds without trailing zeros (5, 0.2, 1.5).
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
