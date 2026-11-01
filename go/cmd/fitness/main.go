// Command fitness runs a repo's fitness checks: small programs that each
// judge one rule, such as spelling, tests, or pull request shape. One binary holds every built-in check; a repo may add local
// checks. Each check decides whether it applies, so any repo runs the same
// suite with nothing to configure.
//
// A run resolves the checks, runs them side by side under time budgets, and
// prints a table of passed, failed, or info results. Info never fails a run. With files staged it judges only those files; --all judges
// every tracked file.
//
// # Exit codes
//
//   - 0: every check passed, or a command succeeded.
//   - 1: a check found problems, or the run could not start, such as a bad
//     config.
//   - 2: a command got arguments it does not take, or explain or docs
//     named an unknown check or topic.
//
// # Learn the rest
//
// `fitness list` names every check, `fitness explain <check>` says why one
// exists and how to pass it, and `fitness docs` serves the guides. All of
// them, and this page, come from the code's own doc comments.
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
// It holds every built-in or installed check minus ignored ones, set by main
// only when running as the runner.
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
	// needs says why this fitness is too old for the check's minFitness,
	// which stops a run before any check starts.
	needs string
}

// outcome is one finished check, in dispatch order.
type outcome struct {
	res checkkit.Result
	ms  int64
	// stderr holds what the check wrote there, which may have streamed live.
	stderr *stream
	// timedOut marks a budget kill and crashed says how a check with no result ended.
	// exit is its protocol code, and startErr says why it could not start.
	timedOut bool
	crashed  string
	exit     int
	startErr string
}

//go:generate go run ../fitness-release checks --root ../../..

// main runs a built-in check when started under its fitness-check-<name>
// program name, and is the runner otherwise.
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
	checkkit.Document(builtinSummaries[name])
	os.Args = append([]string{checkkit.BinPrefix + name}, args...)
	entry()
	return 0
}

// runCheckCommand handles `fitness check <name> [args]`, and `fitness check
// --list`, which prints every built-in check name, one per line. A name no
// built-in check has, such as a repo's own check, runs as `fitness <name>`
// runs it, alone, with its result row.
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
	if _, ok := builtins[args[0]]; !ok {
		return run(args)
	}
	return runBuiltin(args[0], args[1:])
}

// dispatch routes help and the subcommands (`init`, `hook <name>`,
// `pr-check`, `issue-check`, `plan-check`, `close-check`, `dupe-check`, `comment-check`, `boxes`, `strip-front-matter`, `comments`, `check`, `list`, `explain`, `docs`) first.
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
	// Each entry's comment is its line in `fitness --help` and the llms
	// page: a usage directive per form, then what it does. `go generate`
	// reads them, and fails on an entry without one.
	subcommands = map[string]func(args []string) int{
		// Prints a page that teaches an AI the whole CLI.
		//fitness:usage llms
		"llms": runLLMs,
		// Names every check with its summary, scope, budget, and whether this
		// repo enables it.
		//fitness:usage list [--json]
		"list": runList,
		// Prints why a check exists, what it judges, how code comments count,
		// and the config keys it reads.
		//fitness:usage explain <check> [--json]
		"explain": runExplain,
		// Lists the guide topics, or prints one, from the code's doc comments.
		//fitness:usage docs [topic]
		"docs": runDocs,
		// Prints every built-in check name, one per line, or runs one check by
		// name, a repo's own check included.
		//fitness:usage check --list
		//fitness:usage check <name> [args]
		"check": runCheckCommand,
		// Shows the resolved settings and each check's state, or prints a JSON
		// Schema for the config file.
		//fitness:usage config [--json] [--policy=..] [--checks=..]
		//fitness:usage config schema
		"config": runConfig,
		// Installs the shared git hooks into this repo.
		//fitness:usage init
		"init": runInit,
		// Runs a git hook: commit-msg, pre-commit, or pre-push.
		//fitness:usage hook <name>
		"hook": runHookArgs,
		// Draws a --json report as an HTML box diagram.
		//fitness:usage boxes <report.json> <page.html>
		"boxes": runBoxes,
		// Deletes the fitness keys from tracked .md files' front matter, and
		// any block they leave empty.
		//fitness:usage strip-front-matter
		"strip-front-matter": runStripFrontMatter,
		// Prints each code comment block of the tracked files, or of those
		// under the named paths, with its line, language and kind.
		//fitness:usage comments [path...]
		"comments": runComments,
		// Judges this branch's PR and the issues it closes locally, posting
		// nothing.
		//fitness:usage branch-check
		"branch-check": func(args []string) int { return run(append([]string{"local-branch"}, args...)) },
		// Judges pull request titles and descriptions, in GitHub Actions.
		//fitness:usage pr-check
		"pr-check": func([]string) int { return runGH("pr-check") },
		// Judges an issue by its kind label and comments the result, in GitHub
		// Actions.
		//fitness:usage issue-check
		"issue-check": func([]string) int { return runGH("issue-check") },
		// Judges Plan issues only, as issue-check does, in GitHub Actions.
		//fitness:usage plan-check
		"plan-check": func([]string) int { return runGH("plan-check") },
		// Reopens an issue closed with unchecked items, or closes the parent it finishes, in GitHub Actions.
		//fitness:usage close-check
		"close-check": func([]string) int { return runGH("close-check") },
		// Notes when a new issue resembles an older one, in GitHub Actions.
		//fitness:usage dupe-check
		"dupe-check": func([]string) int { return runGH("dupe-check") },
		// Judges a drafted comment, where - or no text reads stdin, or judges
		// issue and PR comments and reply, in GitHub Actions.
		//fitness:usage comment-check --blob <text>
		//fitness:usage comment-check
		"comment-check": runCommentCheck,
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

// run runs the full suite, or with a check name only that check, passing it
// the arguments after its name.
//
//fitness:usage
//fitness:usage <check> [args]
func run(argv []string) int {
	argv, err := ready(argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	if hasBlobFlag(argv) {
		return runBlob(argv)
	}
	s, rest, err := runSetup(argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	return runSelected(s, rest)
}

// runSelected selects the checks argv names and runs them with the run's
// environment. A failure to select them or to read git ends the run as a
// setup error after the setup's cleanup.
func runSelected(s setup, argv []string) int {
	checks, passthrough, err := selectedChecks(s.root, argv)
	var env []string
	if err == nil {
		env, passthrough, err = singleCheckArgs(s.root, checks, passthrough, hasAllFlag(argv), s.scope)
	}
	if err != nil {
		s.cleanup()
		report.Error("fitness setup", err)
		return 1
	}
	if changed := pullRequestChanges(s, argv, env); len(changed) > 0 {
		return runStages(s, checks, env, changed)
	}
	return runChecks(s.root, s.perfPath, checks, passthrough, env, s.cleanup)
}

// runChecks runs checks in one pool with env and passthrough, prints the
// summary table, writes any `--json` report, and returns the exit code.
// cleanup removes the caller's shared files when the run ends.
func runChecks(root, perfPath string, checks []resolved, passthrough, env []string, cleanup func()) int {
	return runStage(root, perfPath, checks, passthrough, env, cleanup, true).code
}

// stageResult is what one pass over the checks found: the exit code, the
// table rows, and each row's findings.
type stageResult struct {
	code  int
	rows  []render.Row
	found [][]checkkit.Finding
}

// runStage runs checks as runChecks says and returns what it found. Any
// JSON the run asks for prints for its last stage, or a failed earlier one.
func runStage(root, perfPath string, checks []resolved, passthrough, env []string, cleanup func(), last bool) stageResult {
	checks, tracked, err := prepareRun(root, checks, cleanup)
	if err != nil {
		report.Error("fitness setup", err)
		return stageResult{code: 1}
	}
	defer tracked.Remove()
	defer stopOnInterrupt(tracked.Remove)()
	hint := scopeHint(env, tracked.Files)
	env = append(env, tracked.Env...)

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env)

	rows, totals := summarize(checks, outcomes, hint)
	totals.Ms = time.Since(start).Milliseconds()
	printSummary(rows, totals)
	found := allFindings(checks, outcomes, rows)
	if last || totals.Failure > 0 {
		printJSON(rows, found, totals)
	}
	perf := perfFor(wantsPerf(perfPath), checks, outcomes, rows, totals.Ms)
	code := finish(rows, found, perf, totals)
	return stageResult{code: withPerfReport(perfPath, perf, code), rows: rows, found: found}
}

// prepareRun drops the checks a commit leaves out and shares the
// tracked-file list, whose Remove deletes it and the caller's files. A
// stalled git call runs cleanup and returns why.
func prepareRun(root string, checks []resolved, cleanup func()) (kept []resolved, tracked walkfs.Shared, err error) {
	kept, err = commitChecks(checks)
	if err == nil {
		tracked, err = shareTrackedFiles(root, kept)
	}
	if err != nil {
		cleanup()
		return nil, walkfs.Shared{}, err
	}
	removeTracked := tracked.Remove
	tracked.Remove = func() {
		removeTracked()
		cleanup()
	}
	return kept, tracked, nil
}

// scopeHint is the line a timed-out row adds on an unscoped run, giving the
// tracked file count and how to judge fewer. A scoped run adds none.
func scopeHint(env []string, files int) string {
	if slices.ContainsFunc(env, scopes) {
		return ""
	}
	if files > 0 {
		return fmt.Sprintf("Hint: the repo tracks %d files; %s", files, scopeAdvice)
	}
	return "Hint: " + scopeAdvice
}

// scopeAdvice says how to judge fewer files than the whole repo.
const scopeAdvice = "scope the run with fitness --dir=<folder>, or commit to check only the staged files"

// scopes reports whether an env entry scopes the run to fewer files.
func scopes(entry string) bool {
	return strings.HasPrefix(entry, changedFilesEnv+"=") || strings.HasPrefix(entry, checkkit.ScopeFileEnv+"=")
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

// withPerfReport writes the `--json` report when asked. A report that cannot
// be written fails the run.
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
func singleCheckArgs(root string, checks []resolved, passthrough []string, all bool, scope []string) (env, remaining []string, err error) {
	env, err = contextEnv(root, checks, all, scope, "")
	if err != nil || len(checks) != 1 {
		return env, nil, err
	}
	arg := checks[0].desc.ContextInlineArg
	if arg == "" {
		return env, passthrough, nil
	}
	value, remaining, found := extractInline(passthrough, arg)
	if found {
		env = append(env, "FITNESS_CTX_MESSAGE="+value)
	}
	return env, remaining, nil
}

// summarize converts outcomes to table rows, printing each check's stderr
// that did not stream live, in dispatch order, and tallies the totals-line
// inputs. A timed-out row ends with hint, when set.
func summarize(checks []resolved, outcomes []outcome, hint string) ([]render.Row, render.Totals) {
	rows := make([]render.Row, len(checks))
	var totals render.Totals
	for i, c := range checks {
		outcomes[i].stderr.replay()
		rows[i] = rowFor(c, outcomes[i], hint)
		totals.Add(rows[i])
	}
	return rows, totals
}

// rowFor builds the table row for one outcome. A budget kill fails with hint,
// and a check that could not judge becomes an info row.
func rowFor(c resolved, o outcome, hint string) render.Row {
	row := render.Row{Name: c.name, Ok: o.res.Ok, FilesChecked: o.res.FilesChecked, Ms: o.ms, Errors: o.res.Errors, Info: o.res.Info}
	if o.timedOut {
		row = render.Row{Name: c.name, FilesChecked: -1, Ms: o.ms, Errors: timedOutErrors(c, o, hint)}
	} else if note, internal := internalNote(o); internal {
		row = render.Row{Name: c.name, Ok: true, FilesChecked: -1, Ms: o.ms, Info: appendNote(o.res.Info, note)}
	}
	row.Info = appendNote(row.Info, c.describeErr)
	return promptRow(row, o)
}

// promptRow makes an info row naming the remote when git or ssh refused to
// ask for a password.
func promptRow(row render.Row, o outcome) render.Row {
	if row.Ok && len(row.Info) == 0 {
		return row
	}
	said := strings.Join(append(append([]string{o.stderr.text()}, row.Errors...), row.Info...), "\n")
	remote, refused := gitx.RefusedPrompt(said)
	if !refused {
		return row
	}
	return render.Row{Name: row.Name, Ok: true, FilesChecked: -1, Ms: row.Ms, Info: appendNote(row.Info, gitx.PromptNote(remote))}
}

// timedOutErrors are a timed-out row's error lines: the budget it ran past, its
// last stderr line, then any hint.
func timedOutErrors(c resolved, o outcome, hint string) []string {
	errs := []string{fmt.Sprintf("Check timed out after %ss", trimFloat(timeoutFor(c).Seconds()))}
	if last := o.stderr.lastLine(); last != "" {
		errs = append(errs, "Last output: "+last)
	}
	if hint != "" {
		errs = append(errs, hint)
	}
	return errs
}

// internalNote reports whether the check could not judge and says why. A
// printed result with ok false is a finding whatever the exit code.
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

// exitNote says why a resulting check exited other than 0 or 1.
// Exit 3 with the check's own info needs no note.
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

// resolveList resolves each name in order, finding every binary first and then
// running the --describe handshakes side by side. A compiled check whose handshake overran gets a retry.
func resolveList(names []string) ([]resolved, error) {
	prebuild(names)
	progs := make([]checkkit.Program, len(names))
	for i, name := range names {
		prog, err := programFor(name)
		if err != nil {
			return nil, err
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

// retryOverran reruns side by side the handshake of each compiled check that overran.
// A script never gets a retry, since an overrun means it ignores --describe.
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
	return asLocal(resolved{name: name, prog: prog, desc: desc, describeErr: describeErr, overran: overran})
}

// describe asks a check binary for its metadata, returning the zero value and
// an info note on failure. The handshake gets its own short budget and kill,
// and overran reports one killed at it.
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

// shareTrackedFiles lists tracked files once for a run of several checks.
// A run of one check lists its own.
func shareTrackedFiles(root string, checks []resolved) (walkfs.Shared, error) {
	if len(checks) < 2 {
		return walkfs.Shared{Remove: func() {}}, nil
	}
	return walkfs.ShareTracked(root)
}

// running holds the process group of every running check, so an interrupt
// can stop them and the heartbeat can name them.
var running = struct {
	sync.Mutex
	pids map[int]liveCheck
}{pids: map[int]liveCheck{}}

// trackRunning records a started check's process group until the returned
// func runs.
func trackRunning(pid int, c liveCheck) (done func()) {
	running.Lock()
	running.pids[pid] = c
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

// stopOnInterrupt passes an interrupt, hangup, or terminate signal to the
// running checks, runs cleanup, then ends the run as the signal would have. A
// signal ignored at start stays ignored, and the returned func stops listening.
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

// contextEnv is the environment every check receives, set per run so an
// inherited value never leaks into a full scan or another repo.
func contextEnv(root string, checks []resolved, all bool, scope []string, msg string) ([]string, error) {
	staged, err := stalledOnly(gitx.Repo(root).StagedFiles())
	if err != nil {
		return nil, err
	}
	scope, err = scopeEnv(root, all, scope, staged)
	if err != nil {
		return nil, err
	}
	env := os.Environ()
	for _, name := range []string{changedFilesEnv, walkfs.TrackedFilesEnv, checkkit.ScopeFileEnv, checkkit.NetworkEnv, "GIT_TERMINAL_PROMPT", "GH_PROMPT_DISABLED"} {
		env = withoutVar(env, name)
	}
	// A check's `git status` would otherwise refresh and rewrite the index,
	// which voids the shared tracked-file list for every later check. A
	// password prompt would wait, unseen, until the budget kills the check.
	env = append(append(env, gitx.NoPrompt(root)...), "GIT_OPTIONAL_LOCKS=0", "FITNESS_STAGED_FILES="+strings.Join(staged, "\n"))
	env = append(env, scope...)
	env = append(env, networkEnv(root, checks, msg)...)
	return append(env, "FITNESS_ENABLED_CHECKS="+strings.Join(checkNames(checks), "\n")), nil
}

// scopeEnv is the env entry that scopes the run: `--dir` when set, else the
// changed files, else none.
func scopeEnv(root string, all bool, scope, staged []string) ([]string, error) {
	if len(scope) > 0 {
		return scope, nil
	}
	changed, err := changedScope(root, all, staged)
	if len(changed) == 0 {
		return nil, err
	}
	return []string{changedFilesEnv + "=" + strings.Join(changed, "\n")}, nil
}

// checkNames lists the checks' names, in order.
func checkNames(checks []resolved) []string {
	names := make([]string, len(checks))
	for i, c := range checks {
		names[i] = c.name
	}
	return names
}

// stalledOnly passes on a git answer, keeping its error only when the call
// stalled. Any other failure reads as an empty answer.
func stalledOnly[T any](v T, err error) (T, error) {
	if gitx.Stalled(err) {
		return v, err
	}
	return v, nil
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
// returning the value and the tokens consumed. Zero means no match, and a lone
// `--arg` consumes 1 with an empty value.
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
// progress line as each starts, and the heartbeat while they run. Checks start
// largest budget first, and outcomes stay in catalog order.
func runPool(root string, checks []resolved, passthrough, env []string) []outcome {
	outcomes := make([]outcome, len(checks))
	out := newStreams(len(checks))
	heartbeat := heartbeatOn()
	if heartbeat {
		defer startHeartbeat(heartbeatEvery)()
	}
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
			outcomes[i] = runOne(root, c, passthrough, env, out.watch(i, heartbeat))
			out.finish(i)
		}(i, c)
	}
	wg.Wait()
	return outcomes
}

// startOrder lists check indices by descending declared budget, keeping catalog
// order for ties. A repo config budget never moves a check, since it says
// nothing about cost.
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
// kills the whole group. Its stderr goes to w's stream.
func runOne(root string, c resolved, passthrough, env []string, w watch) outcome {
	args := append([]string{"--root", root}, passthrough...)
	cmd := c.prog.Command(args...)
	cmd.Dir = root
	cmd.Env = append(append(append([]string{}, env...), "FITNESS_CHECK_NAME="+c.name), checkDirEnv(c.name)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = w.stderr

	live := liveCheck{name: c.name, start: time.Now(), budget: timeoutFor(c)}
	if err := cmd.Start(); err != nil {
		return outcome{stderr: w.stderr, startErr: err.Error()}
	}
	defer trackRunning(cmd.Process.Pid, live)()
	defer warnHalfway(live, w.heartbeat)()
	timedOut := waitWithBudget(cmd, live.budget)
	ms := time.Since(live.start).Milliseconds()

	o := outcome{ms: ms, stderr: w.stderr, timedOut: timedOut}
	if timedOut {
		return o
	}
	return brokenScan(c, decodeOutcome(o, cmd, stdout.Bytes()))
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
// exit code. A non-JSON tail marks the outcome crashed.
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
