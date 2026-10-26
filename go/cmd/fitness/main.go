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
	"io"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// allChecks is the org default and the catalog for explicit external lists.
// Org checks self-gate when their language, tool, config, or input is absent.
// External policy also defaults to the full catalog, with optional selection.
// It is every check built into this executable or installed beside it,
// minus the ignored ones, in the order the language registry gives:
// repo-wide checks, then each module's. installErr says why no checks were
// found; a run stops on it. main sets both when it runs as the runner, so a
// process started as one check skips the work.
var (
	allChecks  []string
	installErr error
)

// besidePrograms are the fitness-check-<name> programs beside this
// executable, which win over the built-in check of their name. main sets it
// when it runs as the runner.
var besidePrograms map[string]string

// catalog lists the default checks, or the error that leaves it empty.
func catalog() ([]string, error) {
	names := slices.Collect(maps.Keys(builtins))
	for name := range besidePrograms {
		if _, ok := builtins[name]; !ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, errors.New("this fitness executable has no built-in checks; reinstall fitness")
	}
	slices.Sort(names)
	return lang.Catalog(names, langs.All), nil
}

// resolved is one runnable check: its program and describe metadata.
type resolved struct {
	name string
	prog checkkit.Program
	desc checkkit.Describe
	// describeErr says why --describe gave no metadata; the check still
	// runs with defaults and its row carries this as info.
	describeErr string
	// overran says the handshake ran past its budget, so a compiled check
	// gets one more try once the crowd of handshakes is done.
	overran bool
	// configMs is the repo config's timeoutMs, the budget of a check that
	// declares none. It stays apart from desc, which holds only what the
	// check declares.
	configMs int
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

//go:generate go run ../fitness-release checks --root ../../..

// main runs a built-in check when started under its fitness-check-<name>
// program name, as the runner and the installed links start it; otherwise it
// is the runner.
func main() {
	metadata.RepoChecks.Find = builtinProgram
	if name, ok := strings.CutPrefix(filepath.Base(os.Args[0]), checkkit.BinPrefix); ok {
		os.Exit(runBuiltin(name, os.Args[1:]))
	}
	besidePrograms = checkkit.Beside()
	allChecks, installErr = catalog()
	os.Exit(dispatch(os.Args[1:]))
}

// runBuiltin runs built-in check name with args exactly as its own program
// would; the check exits the process itself.
func runBuiltin(name string, args []string) int {
	entry, ok := builtins[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "fitness: no built-in check %s; `fitness check --list` names them\n", name)
		return 2
	}
	os.Args = append([]string{checkkit.BinPrefix + name}, args...)
	entry()
	return 0
}

// runCheckCommand handles `fitness check <name> [args]`, and `fitness check
// --list`, which prints every built-in check name, one per line.
func runCheckCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fitness check: name a check, or pass --list")
		return 2
	}
	if args[0] == "--list" {
		for _, name := range slices.Sorted(maps.Keys(builtins)) {
			fmt.Println(name)
		}
		return 0
	}
	return runBuiltin(args[0], args[1:])
}

// dispatch routes help and the subcommands (`init`, `hook <name>`,
// `pr-check`, `issue-check`, `plan-check`, `close-check`, `dupe-check`, `comment-check`, `boxes`, `strip-front-matter`) before falling through to the check runner, so the
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
// arguments after the name. init fills it, since a handler runs checks that
// look the names up.
var subcommands map[string]func(args []string) int

func init() {
	subcommands = map[string]func(args []string) int{
		"init":               runInit,
		"hook":               runHookArgs,
		"pr-check":           func([]string) int { return runGH("pr-check") },
		"issue-check":        func([]string) int { return runGH("issue-check") },
		"plan-check":         func([]string) int { return runGH("plan-check") },
		"close-check":        func([]string) int { return runGH("close-check") },
		"dupe-check":         func([]string) int { return runGH("dupe-check") },
		"comment-check":      runCommentCheck,
		"branch-check":       func(args []string) int { return run(append([]string{"local-branch"}, args...)) },
		"boxes":              runBoxes,
		"strip-front-matter": runStripFrontMatter,
		"check":              runCheckCommand,
	}
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
  fitness --dir=<folder>   judge only the files under one folder
  fitness --blob <text>    run every blob-capable check on text (- reads stdin)
  fitness --json=<file>    also write each check's time, budget, files, binary size, and error lines as JSON
  fitness boxes <report.json> <page.html>
                           draw a --json report as an HTML box diagram
  fitness init             install the shared git hooks into this repo
  fitness strip-front-matter
                           remove front matter that holds only fitness keys from tracked .md files
  fitness hook <name>      run a git hook (commit-msg | pre-commit | pre-push)
  fitness pr-check         validate PR titles and descriptions (GitHub Actions)
  fitness branch-check     validate this branch's PR and linked issues locally
  fitness issue-check      validate an issue by its kind label and comment the result (GitHub Actions)
  fitness plan-check       issue-check for Plan issues only (GitHub Actions)
  fitness close-check      reopen an issue closed with unchecked items (GitHub Actions)
  fitness dupe-check       note when a new issue resembles an older one (GitHub Actions)
  fitness comment-check --blob <text>
                           judge a drafted comment (- or no arguments reads stdin)
  fitness comment-check    judge issue and PR comments and reply (GitHub Actions)

Docs: https://github.com/may-journal/fitness-runner#usage`)
	return 0
}

func run(argv []string) int {
	argv, err := ready(argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	if hasBlobFlag(argv) {
		return runBlob(argv)
	}
	s, argv, err := runSetup(argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	checks, passthrough, err := selectedChecks(s.root, argv)
	if err != nil {
		s.cleanup()
		report.Error("fitness setup", err)
		return 1
	}
	env, passthrough := singleCheckArgs(s.root, checks, passthrough, hasAllFlag(argv), s.scope)
	return runChecks(s.root, s.perfPath, checks, passthrough, env, s.cleanup)
}

// runChecks runs checks in one pool with env and passthrough, prints the
// summary table, writes the `--json` report when perfPath names one, and
// returns the run's exit code. cleanup removes the caller's shared files
// when the run ends or is interrupted.
func runChecks(root, perfPath string, checks []resolved, passthrough, env []string, cleanup func()) int {
	checks = commitChecks(checks)
	tracked, removeTracked := shareTrackedFiles(root, checks)
	remove := func() {
		removeTracked()
		cleanup()
	}
	defer remove()
	defer stopOnInterrupt(remove)()
	env = append(env, tracked...)

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env)

	rows, totals := summarize(checks, outcomes)
	totals.Ms = time.Since(start).Milliseconds()
	printSummary(rows, totals)
	found := allFindings(checks, outcomes, rows)
	printJSON(rows, found, totals)
	perf := perfFor(wantsPerf(perfPath), checks, outcomes, rows, totals.Ms)
	code := finish(rows, found, perf, totals)
	return withPerfReport(perfPath, perf, code)
}

// setup is what a run reads from argv before it selects checks.
type setup struct {
	root, perfPath string
	// scope is the `--dir` folder's env entry, and cleanup removes its file.
	scope   []string
	cleanup func()
}

// runSetup finds the repo root and takes the `--json` report path and the
// `--dir` folder out of argv, returning the rest. The folder's files are
// shared once here, so the caller removes them with the setup's cleanup.
func runSetup(argv []string) (s setup, rest []string, err error) {
	s.root, err = os.Getwd()
	if err != nil {
		return s, nil, err
	}
	s.perfPath, rest, err = jsonPath(argv)
	if err != nil {
		return s, nil, err
	}
	only, rest, err := dirScope(s.root, rest)
	if err != nil {
		return s, nil, err
	}
	s.scope, s.cleanup, err = shareScope(only)
	return s, rest, err
}

// withPerfReport writes the `--json` report when one was asked for; a report
// that cannot be written fails the run, since the caller relies on it.
func withPerfReport(path string, perf perfReport, code int) int {
	if path == "" {
		return code
	}
	if err := writePerfReport(path, perf); err != nil {
		report.Error("fitness --json", err)
		return 1
	}
	return code
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
func singleCheckArgs(root string, checks []resolved, passthrough []string, all bool, scope []string) (env, remaining []string) {
	env = contextEnv(root, checks, all, scope)
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
// failed, so failure output stays together, or when stdout carries JSON.
func printSummary(rows []render.Row, totals render.Totals) {
	p := render.NewPalette()
	table := render.Table(rows, render.TermCols(), p)
	total := render.TotalLine(totals, p)
	out := humanOut(totals)
	fmt.Fprintln(out, table)
	fmt.Fprintln(out, total)
}

// humanOut is where text for people goes after a run: stdout, unless a
// check failed or stdout carries JSON.
func humanOut(totals render.Totals) *os.File {
	if totals.Failure > 0 || formatJSON {
		return os.Stderr
	}
	return os.Stdout
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
	return resolveList(checkCatalog())
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
// then run side by side and come back in catalog order. A compiled check
// whose handshake overran in that crowd gets a second try, so a slow first
// start after an install does not drop its own budget.
func resolveList(names []string) ([]resolved, error) {
	progs := make([]checkkit.Program, len(names))
	for i, name := range names {
		prog, err := findCheckBinary(name)
		if err != nil {
			return nil, fmt.Errorf("%w: fitness-check-%s", errNotInstalled, name)
		}
		progs[i] = prog
	}
	checks := par.Map(len(names), par.Processes(), func(i int) resolved {
		return describeCheck(names[i], progs[i])
	})
	if len(checks) > 1 {
		retryOverran(checks)
	}
	return checks, nil
}

// retryOverran runs once more, side by side, the handshake of each compiled
// check that overran in the crowd, so the retries cost one budget at most.
// A script never gets a retry: it starts no slower after an install, so an
// overrun means it ignores --describe, and its work must not run twice.
func retryOverran(checks []resolved) {
	var again []int
	for i, c := range checks {
		if c.overran && !isScript(c.prog.Path) {
			again = append(again, i)
		}
	}
	retried := par.Map(len(again), par.Processes(), func(k int) resolved {
		return describeCheck(checks[again[k]].name, checks[again[k]].prog)
	})
	for k, i := range again {
		checks[i] = retried[k]
	}
}

// isScript reports whether bin starts with a #! line, as a local check
// script does; an unreadable file counts as one, so it is never retried.
func isScript(bin string) bool {
	f, err := os.Open(bin)
	if err != nil {
		return true
	}
	defer f.Close()
	head := make([]byte, 2)
	n, _ := io.ReadFull(f, head)
	return n < 2 || string(head) == "#!"
}

// describeCheck runs one check's --describe handshake.
func describeCheck(name string, prog checkkit.Program) resolved {
	desc, describeErr, overran := describe(prog)
	return resolved{name: name, prog: prog, desc: desc, describeErr: describeErr, overran: overran}
}

// describe asks a check binary for its metadata. On any failure it returns
// the zero value, so the check runs with defaults, and an info note saying
// why. The handshake gets its own short budget and process-group kill — a
// local script that ignores --describe and starts real work must not hang
// resolution. overran reports a handshake killed at its budget.
func describe(prog checkkit.Program) (d checkkit.Describe, note string, overran bool) {
	const failed = "--describe gave no metadata, so the check ran with defaults: "
	cmd := prog.Command("--describe")
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

// shareTrackedFiles lists tracked files once for a run of several checks,
// returning the env entry that hands the list over and a cleanup. A run of
// one check gains nothing, so that check lists its own, as the commit-msg
// hook's single-check runs do.
func shareTrackedFiles(root string, checks []resolved) (env []string, cleanup func()) {
	if len(checks) < 2 {
		return nil, func() {}
	}
	return walkfs.ShareTracked(root)
}

// running holds the process group of every check still running, so an
// interrupt can stop them with the runner.
var running = struct {
	sync.Mutex
	pids map[int]bool
}{pids: map[int]bool{}}

// trackRunning records a started check's process group until the returned
// func runs.
func trackRunning(pid int) (done func()) {
	running.Lock()
	running.pids[pid] = true
	running.Unlock()
	return func() {
		running.Lock()
		delete(running.pids, pid)
		running.Unlock()
	}
}

// signalRunning passes sig to every running check's process group.
func signalRunning(sig syscall.Signal) {
	running.Lock()
	defer running.Unlock()
	for pid := range running.pids {
		killGroup(pid, sig)
	}
}

// stopOnInterrupt makes an interrupt, hangup, or terminate signal pass to
// the running checks and run cleanup, then end the run as the signal would
// have. A signal the runner was started with ignored stays ignored. The
// returned func stops listening.
func stopOnInterrupt(cleanup func()) (stop func()) {
	var caught []os.Signal
	for _, s := range []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		if !signal.Ignored(s) {
			caught = append(caught, s)
		}
	}
	if len(caught) == 0 {
		return func() {}
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, caught...)
	done := make(chan struct{})
	go func() {
		select {
		case sig := <-sigs:
			signalRunning(sig.(syscall.Signal))
			cleanup()
			signal.Reset(sig)
			_ = syscall.Kill(os.Getpid(), sig.(syscall.Signal))
		case <-done:
		}
	}()
	return func() {
		signal.Stop(sigs)
		close(done)
	}
}

// contextEnv is the environment every check receives. FITNESS_CHANGED_FILES
// is set only for a scoped run, and the tracked-file and `--dir` lists only
// by run, so an inherited value never leaks into a full scan or another
// repo. A `--dir` scope, the env entry from shareScope, replaces the diff,
// so no check reads the folder as changes. The staged files are read once
// for both variables.
func contextEnv(root string, checks []resolved, all bool, scope []string) []string {
	env := os.Environ()
	for _, name := range []string{"FITNESS_CHANGED_FILES", walkfs.TrackedFilesEnv, checkkit.ScopeFileEnv} {
		env = withoutVar(env, name)
	}
	staged := gitx.StagedFiles(root)
	// A check's `git status` would otherwise refresh and rewrite the index,
	// which voids the shared tracked-file list for every later check.
	env = append(env, "GIT_OPTIONAL_LOCKS=0", "FITNESS_STAGED_FILES="+strings.Join(staged, "\n"))
	if len(scope) > 0 {
		env = append(env, scope...)
	} else if changed := changedScope(root, all, staged); len(changed) > 0 {
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

// startOrder lists check indices by descending declared budget, a check
// that declares none counting as the default. A budget from the repo config
// says nothing about cost, so it never moves a check. Equal budgets keep
// catalog order.
func startOrder(checks []resolved) []int {
	order := make([]int, len(checks))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(declaredBudget(checks[b]), declaredBudget(checks[a]))
	})
	return order
}

// runOne execs a single check in its own process group with a timeout that
// kills the whole group (TERM, then KILL after a grace period).
func runOne(root string, c resolved, passthrough, env []string) outcome {
	args := append([]string{"--root", root}, passthrough...)
	cmd := c.prog.Command(args...)
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
	defer trackRunning(cmd.Process.Pid)()
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
