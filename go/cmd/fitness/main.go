// Command fitness runs the configured check suite: it resolves check
// binaries, execs them in a bounded parallel pool with per-check timeouts,
// and renders the results table. See docs/plans/archive/01-go-rewrite.md.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/render"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// allChecks is the org default and the catalog for explicit external lists.
// Org checks self-gate when their language, tool, config, or input is absent.
// External policy also defaults to the full catalog, with optional selection.
var allChecks = []string{
	"read-repo-first",
	"semantic-commit",
	"plan-trailer",
	"issue-link-once",
	"commit-attribution",
	"changelog",
	"changelog-updated",
	"changelog-bullets",
	"release-changelog",
	"cspell",
	"jscpd",
	"gitignore-why",
	"no-plans-dir",
	"doc-template",
	"no-contrastive-reframing",
	"repeated-string-literals",
	"markdown-front-matter",
	"markdown-filename-kebab-case",
	"markdown-filename-camel-case",
	"markdown-links",
	"markdown-no-bold-italic",
	"text-readability",
	"prose-budget",
	"mermaid-callouts",
	"mermaid-callout-why",
	"mermaid-diagram-prose",
	"mermaid-diagram-table-gap",
	"mermaid-legend",
	"mermaid-level-bleed",
	"plan-structure",
	"pr-structure",
	"pr-closes-issue",
	"go-complexity",
	"go-vet",
	"gofmt",
	"go-test",
	"build-output-untracked",
	"swiftlint",
	"eslint",
	"prettier",
	"no-eslint-disable",
	"node-version",
	"dependency-currency",
	"vitest-coverage-exclude",
	"vitest-coverage-full",
}

const defaultTimeout = 5000 * time.Millisecond

// resolved is one runnable check: its binary and describe metadata.
type resolved struct {
	name string
	bin  string
	desc checkkit.Describe
}

// outcome is one finished check, in dispatch order.
type outcome struct {
	res    checkkit.Result
	ms     int64
	stderr []byte
	// timedOut marks a budget kill; crashed carries a non-protocol exit.
	timedOut bool
	crashed  int
}

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// dispatch routes help and the subcommands (`init`, `hook <name>`,
// `pr-check`, `plan-check`, `close-check`) before falling through to the check runner, so the
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
	"init":        runInit,
	"hook":        runHookArgs,
	"pr-check":    func([]string) int { return runPRCheck() },
	"plan-check":  func([]string) int { return runPlanCheck() },
	"close-check": func([]string) int { return runCloseCheck() },
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
  fitness --check=<name>   run one check by name
  fitness --policy=external --checks=name,name
                           run exactly the selected checks
  fitness --all            scan every file, even with files staged
  fitness init             install the shared git hooks into this repo
  fitness hook <name>      run a git hook (commit-msg | pre-commit | pre-push)
  fitness pr-check         validate PR titles and descriptions (GitHub Actions)
  fitness plan-check       validate Plan issues and comment the result (GitHub Actions)
  fitness close-check      reopen an issue closed with unchecked items (GitHub Actions)

Docs: https://github.com/may-journal/fitness-runner#usage`)
	return 0
}

func run(argv []string) int {
	root, err := os.Getwd()
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	checks, jobs, passthrough, err := selectedChecks(root, argv)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	env, passthrough := singleCheckArgs(root, checks, passthrough, hasAllFlag(argv))

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env, jobs)

	rows, success, failure, files := summarize(checks, outcomes)
	elapsed := time.Since(start).Milliseconds()
	printSummary(rows, success, failure, files, elapsed)
	return finish(rows, success, failure, files, elapsed)
}

// loadConfig loads the repo config; a legacy JS/TS config (kept while the TS
// suite coexists) is tolerated in single-check mode, which never consults
// the config.
func loadConfig(root, spec string) (*conf.Config, error) {
	cfg, err := conf.Load(root)
	if err != nil && (spec == "" || !errors.Is(err, conf.ErrLegacyConfig)) {
		return nil, err
	}
	return cfg, nil
}

// warnLegacyChecks notes on stderr that a retired checks list is ignored.
func warnLegacyChecks(cfg *conf.Config) {
	if cfg != nil && len(cfg.LegacyChecks) > 0 {
		fmt.Fprintln(os.Stderr, "fitness: the checks list in .fitnessrc.json is ignored; every check runs. Turn one off with disabledChecks.")
	}
}

// prepareChecks resolves the check list, turning an empty resolution into
// the Unknown-check error the CLI renders.
func prepareChecks(cfg *conf.Config, spec string) ([]resolved, error) {
	checks, err := resolveChecks(cfg, spec)
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
	return checks, nil
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
func summarize(checks []resolved, outcomes []outcome) (rows []render.Row, success, failure, files int) {
	rows = make([]render.Row, len(checks))
	for i, c := range checks {
		o := outcomes[i]
		os.Stderr.Write(o.stderr)
		row := rowFor(c, o)
		if row.Ok {
			success++
		} else {
			failure++
		}
		if row.FilesChecked >= 0 {
			files += row.FilesChecked
		}
		rows[i] = row
	}
	return rows, success, failure, files
}

// rowFor builds the table row for one outcome, overriding the check's own
// result for budget kills and non-protocol crashes.
func rowFor(c resolved, o outcome) render.Row {
	row := render.Row{Name: c.name, Ok: o.res.Ok, FilesChecked: o.res.FilesChecked, Ms: o.ms, Errors: o.res.Errors}
	if o.timedOut {
		row.Ok = false
		row.FilesChecked = -1
		row.Errors = []string{fmt.Sprintf("Check timed out after %ss", trimFloat(timeoutFor(c).Seconds()))}
	} else if o.crashed != 0 {
		row.Ok = false
		row.FilesChecked = -1
		row.Errors = []string{fmt.Sprintf("check failed (exit %d)", o.crashed)}
	}
	return row
}

// printSummary renders the table and total line — to stderr when any check
// failed, so failure output stays together.
func printSummary(rows []render.Row, success, failure, files int, elapsedMs int64) {
	p := render.NewPalette()
	table := render.Table(rows, render.TermCols(), p)
	total := render.TotalLine(success, failure, files, elapsedMs, p)
	out := os.Stdout
	if failure > 0 {
		out = os.Stderr
	}
	fmt.Fprintln(out, table)
	fmt.Fprintln(out, total)
}

// parseArgv extracts the check spec (--check= wins over the first
// positional; empty values are absent), --jobs, and the passthrough args.
func parseArgv(argv []string) (spec string, fromFlag bool, jobs int, passthrough []string) {
	s := scanArgv(argv)
	var specIdx int
	spec, specIdx, fromFlag = s.chooseSpec()
	if spec == "" {
		return spec, fromFlag, s.jobs, nil
	}
	return spec, fromFlag, s.jobs, collectPassthrough(argv, specIdx, fromFlag)
}

// argScan accumulates the classification pass over argv: the first
// non-empty --check= value, the first non-empty positional, and --jobs.
type argScan struct {
	checkSpec, posSpec string
	checkIdx, posIdx   int
	jobs               int
}

// scanArgv classifies every argument; indices start at -1 (absent).
func scanArgv(argv []string) argScan {
	s := argScan{checkIdx: -1, posIdx: -1}
	for i, a := range argv {
		s.visit(i, a)
	}
	return s
}

// visit records one argument: runner flags by prefix, then anything that is
// not a flag as a positional candidate; unknown flags are ignored.
func (s *argScan) visit(i int, a string) {
	switch {
	case strings.HasPrefix(a, "--check="):
		s.setCheck(i, strings.TrimPrefix(a, "--check="))
	case strings.HasPrefix(a, "--jobs="):
		s.setJobs(strings.TrimPrefix(a, "--jobs="))
	case !strings.HasPrefix(a, "-"):
		s.setPositional(i, a)
	}
}

// setCheck keeps the first non-empty --check= value.
func (s *argScan) setCheck(i int, v string) {
	if v != "" && s.checkIdx < 0 {
		s.checkSpec, s.checkIdx = v, i
	}
}

// setJobs applies a valid --jobs= value (an integer >= 1); anything else is
// ignored.
func (s *argScan) setJobs(v string) {
	if n, err := strconv.Atoi(v); err == nil && n >= 1 {
		s.jobs = n
	}
}

// setPositional keeps the first non-empty positional argument.
func (s *argScan) setPositional(i int, a string) {
	if a != "" && s.posIdx < 0 {
		s.posSpec, s.posIdx = a, i
	}
}

// chooseSpec picks the winning spec: --check= beats the positional; specIdx
// is -1 when no spec was given.
func (s argScan) chooseSpec() (spec string, specIdx int, fromFlag bool) {
	if s.checkSpec != "" {
		return s.checkSpec, s.checkIdx, true
	}
	if s.posSpec != "" {
		return s.posSpec, s.posIdx, false
	}
	return "", -1, false
}

// collectPassthrough gathers the args forwarded to a single check, dropping
// the spec itself and the runner's own flags. Flag-form keeps args anywhere;
// positional form keeps only args after the spec.
func collectPassthrough(argv []string, specIdx int, fromFlag bool) []string {
	var passthrough []string
	for i, a := range argv {
		if isRunnerArg(a, i, specIdx) {
			continue
		}
		if fromFlag || i > specIdx {
			passthrough = append(passthrough, a)
		}
	}
	return passthrough
}

// isRunnerArg reports whether argv[i] belongs to the runner itself (the
// spec or a --check=/--jobs= flag) and must not be forwarded.
func isRunnerArg(a string, i, specIdx int) bool {
	return i == specIdx || a == allFlag || strings.HasPrefix(a, "--check=") || strings.HasPrefix(a, "--jobs=")
}

// resolveChecks builds the check list: the one named on the CLI, else every
// check minus the config's disabledChecks.
func resolveChecks(cfg *conf.Config, spec string) ([]resolved, error) {
	if spec != "" {
		return resolveSingle(spec)
	}
	return resolveList(allChecks, disabledSet(cfg))
}

// resolveSingle resolves a CLI check name into a one-element list; an empty
// list means Unknown check.
func resolveSingle(spec string) ([]resolved, error) {
	c, err := resolveName(spec, true)
	if err != nil || c == nil {
		return nil, err
	}
	return []resolved{*c}, nil
}

// disabledSet builds the lookup of config-disabled check names.
func disabledSet(cfg *conf.Config) map[string]bool {
	disabled := map[string]bool{}
	if cfg == nil {
		return disabled
	}
	for _, d := range cfg.DisabledChecks {
		disabled[d] = true
	}
	return disabled
}

// resolveList resolves each name not disabled, in order.
func resolveList(names []string, disabled map[string]bool) ([]resolved, error) {
	var out []resolved
	for _, name := range names {
		if disabled[name] {
			continue
		}
		c, err := resolveName(name, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

// resolveName maps a check name to its binary. A missing binary is nil for
// a CLI name (the caller renders Unknown check) and an error for the list,
// since every check ships together.
func resolveName(name string, cli bool) (*resolved, error) {
	bin, err := findCheckBinary(name)
	if err == nil {
		return &resolved{name: name, bin: bin, desc: describe(bin)}, nil
	}
	if cli {
		return nil, nil
	}
	return nil, fmt.Errorf("Check binary not installed: fitness-check-%s", name)
}

// isExecutableFile reports whether path is a non-directory with any execute
// bit set.
func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// findCheckBinary locates fitness-check-<name> beside this executable, then
// on PATH.
func findCheckBinary(name string) (string, error) {
	binName := "fitness-check-" + name
	if self, err := os.Executable(); err == nil {
		if sibling := filepath.Join(filepath.Dir(self), binName); isExecutableFile(sibling) {
			return sibling, nil
		}
	}
	return exec.LookPath(binName)
}

// describe asks a check binary for its metadata; zero value on any failure.
// The handshake gets its own short budget and process-group kill — a local
// script that ignores --describe and starts real work must not hang
// resolution.
func describe(bin string) checkkit.Describe {
	cmd := exec.Command(bin, "--describe")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return checkkit.Describe{}
	}
	if !waitDescribe(cmd) {
		return checkkit.Describe{}
	}
	var d checkkit.Describe
	if json.Unmarshal(lastJSONLine(stdout.Bytes()), &d) != nil {
		return checkkit.Describe{}
	}
	return d
}

// waitDescribe waits for the handshake to exit cleanly within its short
// budget, killing the process group on overrun; false means no usable
// output.
func waitDescribe(cmd *exec.Cmd) bool {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(2 * time.Second):
		killGroup(cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return false
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

func timeoutFor(c resolved) time.Duration {
	if c.desc.TimeoutMs > 0 {
		return time.Duration(c.desc.TimeoutMs) * time.Millisecond
	}
	return defaultTimeout
}

// runPool executes every check with bounded parallelism, dispatching in
// order and printing the progress line as each starts.
func runPool(root string, checks []resolved, passthrough, env []string, jobs int) []outcome {
	if jobs < 1 {
		jobs = runtime.NumCPU()
		if jobs > 8 {
			jobs = 8
		}
	}
	outcomes := make([]outcome, len(checks))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i, c := range checks {
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
		return outcome{res: checkkit.Fail(-1, err.Error()), ms: 0, crashed: 0}
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
// process group, then KILLs after a grace period. Reports whether the check
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
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		killGroup(cmd.Process.Pid, syscall.SIGKILL)
		<-done
	}
	return true
}

// decodeOutcome parses the check's protocol result from stdout; a non-JSON
// tail marks the outcome crashed with the process exit code (1 when the
// process exited 0).
func decodeOutcome(o outcome, cmd *exec.Cmd, stdout []byte) outcome {
	var res checkkit.Result
	if err := json.Unmarshal(lastJSONLine(stdout), &res); err != nil {
		o.crashed = cmd.ProcessState.ExitCode()
		if o.crashed == 0 {
			o.crashed = 1
		}
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

// lastJSONLine returns the last non-empty stdout line (the protocol allows
// stray tool noise ahead of the result object).
func lastJSONLine(out []byte) []byte {
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if line := bytes.TrimSpace(lines[i]); len(line) > 0 {
			return line
		}
	}
	return nil
}

// trimFloat renders seconds without trailing zeros (5, 0.2, 1.5).
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
