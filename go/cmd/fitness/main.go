// Command fitness runs the configured check suite: it resolves check
// binaries, execs them in a bounded parallel pool with per-check timeouts,
// and renders the results table. See plans/01-go-rewrite.md.
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
)

// defaultChecks is the run order when config has no checks list — the Go
// twin of the bundle's defaultChecks (the bundle concept dissolves when
// checks are sibling binaries).
var defaultChecks = []string{
	"read-repo-first",
	"changelog",
	"changelog-updated",
	"cspell",
	"eslint",
	"markdown-no-bold-italic",
	"prettier",
	"node-version",
	"markdown-front-matter",
	"semantic-commit",
	"jscpd",
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
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	spec, _, jobs, passthrough := parseArgv(argv)

	cfg, err := loadConfig(root, spec)
	if err != nil {
		errRed(err.Error())
		return 1
	}

	fmt.Fprintln(os.Stderr, "Resolving checks...")
	checks, err := prepareChecks(root, cfg, spec)
	if err != nil {
		errRed(err.Error())
		return 1
	}
	env, passthrough := singleCheckArgs(root, checks, passthrough)

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env, jobs)

	rows, success, failure, files := summarize(checks, outcomes)
	printSummary(rows, success, failure, files, time.Since(start).Milliseconds())
	if failure > 0 {
		return 1
	}
	return 0
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

// prepareChecks resolves the check list, turning an empty resolution into
// the Unknown-check error the CLI renders.
func prepareChecks(root string, cfg *conf.Config, spec string) ([]resolved, error) {
	checks, err := resolveChecks(root, cfg, spec)
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
func singleCheckArgs(root string, checks []resolved, passthrough []string) (env, remaining []string) {
	env = contextEnv(root, checks)
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
	return i == specIdx || strings.HasPrefix(a, "--check=") || strings.HasPrefix(a, "--jobs=")
}

// resolveChecks builds the ordered, deduped check list.
func resolveChecks(root string, cfg *conf.Config, spec string) ([]resolved, error) {
	if spec != "" {
		return resolveSingle(root, spec)
	}
	names, fromConfig := configuredNames(cfg)
	return resolveList(root, names, fromConfig, disabledSet(cfg))
}

// resolveSingle resolves a CLI spec into a one-element list; an empty list
// means Unknown check.
func resolveSingle(root, spec string) ([]resolved, error) {
	c, err := resolveOne(root, spec, true, false)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return []resolved{*c}, nil
}

// configuredNames returns the check names to run (the config list overrides
// the default list) and whether they came from config.
func configuredNames(cfg *conf.Config) ([]string, bool) {
	if cfg != nil && len(cfg.Checks) > 0 {
		return cfg.Checks, true
	}
	return defaultChecks, false
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

// resolveList resolves an ordered name list, dropping disabled entries,
// silent misses, and duplicate names (first occurrence wins).
func resolveList(root string, names []string, fromConfig bool, disabled map[string]bool) ([]resolved, error) {
	var out []resolved
	seen := map[string]bool{}
	for _, entry := range names {
		c, err := resolveEntry(root, entry, fromConfig, disabled)
		if err != nil {
			return nil, err
		}
		if c == nil || seen[c.name] {
			continue
		}
		seen[c.name] = true
		out = append(out, *c)
	}
	return out, nil
}

// resolveEntry resolves one list entry; nil means skip (a disabled name, or
// a config entry whose binary is missing).
func resolveEntry(root, entry string, fromConfig bool, disabled map[string]bool) (*resolved, error) {
	if !isPathSpec(entry) && disabled[entry] {
		return nil, nil
	}
	return resolveOne(root, entry, false, fromConfig)
}

func isPathSpec(spec string) bool {
	return strings.ContainsAny(spec, "/\\")
}

// resolveOne maps a spec to a check binary. Missing name specs: nil for CLI
// (caller renders Unknown check) and config entries (skipped silently), an
// error for default-list names. Missing path specs always error unless CLI.
func resolveOne(root, spec string, cli, fromConfig bool) (*resolved, error) {
	if isPathSpec(spec) {
		return resolvePathSpec(root, spec, cli)
	}
	bin, err := findCheckBinary(spec)
	if err != nil {
		if cli || fromConfig {
			return nil, nil
		}
		return nil, fmt.Errorf("Check binary not installed: fitness-check-%s", spec)
	}
	return &resolved{name: spec, bin: bin, desc: describe(bin)}, nil
}

// resolvePathSpec resolves a local path spec to its executable; the check
// name comes from --describe metadata when present, else the basename sans
// extension.
func resolvePathSpec(root, spec string, cli bool) (*resolved, error) {
	abs := spec
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, spec)
	}
	if !isExecutableFile(abs) {
		if cli {
			return nil, nil
		}
		return nil, fmt.Errorf("Local check not found or not executable: %s", spec)
	}
	name := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	desc := describe(abs)
	if desc.Name != "" {
		name = desc.Name
	}
	return &resolved{name: name, bin: abs, desc: desc}, nil
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

// contextEnv is the environment every check receives.
func contextEnv(root string, checks []resolved) []string {
	env := os.Environ()
	env = append(env, "FITNESS_STAGED_FILES="+strings.Join(gitx.StagedFiles(root), "\n"))
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

func errRed(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
