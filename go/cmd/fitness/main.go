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

	cfg, err := conf.Load(root)
	if err != nil {
		// single-check mode never consults the config, so a legacy JS/TS
		// config (kept while the TS suite coexists) must not block it
		if spec == "" || !errors.Is(err, conf.ErrLegacyConfig) {
			errRed(err.Error())
			return 1
		}
	}

	fmt.Fprintln(os.Stderr, "Resolving checks...")
	checks, err := resolveChecks(root, cfg, spec)
	if err != nil {
		errRed(err.Error())
		return 1
	}
	if len(checks) == 0 {
		which := spec
		if which == "" {
			which = "(none)"
		}
		errRed("Unknown check: " + which)
		return 1
	}

	// Passthrough and context-inline apply only to a single-check run.
	env := contextEnv(root, checks)
	if len(checks) == 1 {
		if arg := checks[0].desc.ContextInlineArg; arg != "" {
			var value string
			var found bool
			value, passthrough, found = extractInline(passthrough, arg)
			if found {
				env = append(env, "FITNESS_CTX_MESSAGE="+value)
			}
		}
	} else {
		passthrough = nil
	}

	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, passthrough, env, jobs)

	rows := make([]render.Row, len(checks))
	success, failure, files := 0, 0, 0
	for i, c := range checks {
		o := outcomes[i]
		os.Stderr.Write(o.stderr)
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

	p := render.NewPalette()
	table := render.Table(rows, render.TermCols(), p)
	total := render.TotalLine(success, failure, files, time.Since(start).Milliseconds(), p)
	out := os.Stdout
	if failure > 0 {
		out = os.Stderr
	}
	fmt.Fprintln(out, table)
	fmt.Fprintln(out, total)
	if failure > 0 {
		return 1
	}
	return 0
}

// parseArgv extracts the check spec (--check= wins over the first
// positional; empty values are absent), --jobs, and the passthrough args.
func parseArgv(argv []string) (spec string, fromFlag bool, jobs int, passthrough []string) {
	checkSpec, posSpec := "", ""
	checkIdx, posIdx := -1, -1
	for i, a := range argv {
		switch {
		case strings.HasPrefix(a, "--check="):
			if v := strings.TrimPrefix(a, "--check="); v != "" && checkIdx < 0 {
				checkSpec, checkIdx = v, i
			}
		case strings.HasPrefix(a, "--jobs="):
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "--jobs=")); err == nil && n >= 1 {
				jobs = n
			}
		case strings.HasPrefix(a, "-"):
		default:
			if a != "" && posIdx < 0 {
				posSpec, posIdx = a, i
			}
		}
	}
	specIdx := -1
	if checkSpec != "" {
		spec, specIdx, fromFlag = checkSpec, checkIdx, true
	} else if posSpec != "" {
		spec, specIdx = posSpec, posIdx
	}
	if spec == "" {
		return spec, fromFlag, jobs, nil
	}
	for i, a := range argv {
		if i == specIdx || strings.HasPrefix(a, "--check=") || strings.HasPrefix(a, "--jobs=") {
			continue
		}
		// flag-form keeps args anywhere; positional form keeps only args after the spec
		if fromFlag || i > specIdx {
			passthrough = append(passthrough, a)
		}
	}
	return spec, fromFlag, jobs, passthrough
}

// resolveChecks builds the ordered, deduped check list.
func resolveChecks(root string, cfg *conf.Config, spec string) ([]resolved, error) {
	if spec != "" {
		c, err := resolveOne(root, spec, true, false)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, nil
		}
		return []resolved{*c}, nil
	}
	names := defaultChecks
	fromConfig := false
	if cfg != nil && len(cfg.Checks) > 0 {
		names = cfg.Checks
		fromConfig = true
	}
	disabled := map[string]bool{}
	if cfg != nil {
		for _, d := range cfg.DisabledChecks {
			disabled[d] = true
		}
	}
	var out []resolved
	seen := map[string]bool{}
	for _, entry := range names {
		if !isPathSpec(entry) && disabled[entry] {
			continue
		}
		c, err := resolveOne(root, entry, false, fromConfig)
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

func isPathSpec(spec string) bool {
	return strings.ContainsAny(spec, "/\\")
}

// resolveOne maps a spec to a check binary. Missing name specs: nil for CLI
// (caller renders Unknown check) and config entries (skipped silently), an
// error for default-list names. Missing path specs always error unless CLI.
func resolveOne(root, spec string, cli, fromConfig bool) (*resolved, error) {
	if isPathSpec(spec) {
		abs := spec
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, spec)
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
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
	bin, err := findCheckBinary(spec)
	if err != nil {
		if cli || fromConfig {
			return nil, nil
		}
		return nil, fmt.Errorf("Check binary not installed: fitness-check-%s", spec)
	}
	return &resolved{name: spec, bin: bin, desc: describe(bin)}, nil
}

// findCheckBinary locates fitness-check-<name> beside this executable, then
// on PATH.
func findCheckBinary(name string) (string, error) {
	binName := "fitness-check-" + name
	if self, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(self), binName)
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
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
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return checkkit.Describe{}
		}
	case <-time.After(2 * time.Second):
		killGroup(cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return checkkit.Describe{}
	}
	var d checkkit.Describe
	if json.Unmarshal(lastJSONLine(stdout.Bytes()), &d) != nil {
		return checkkit.Describe{}
	}
	return d
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
		a := args[i]
		if !found && strings.HasPrefix(a, argName+"=") {
			value, found = strings.TrimPrefix(a, argName+"="), true
			continue
		}
		if !found && a == argName {
			found = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
			continue
		}
		remaining = append(remaining, a)
	}
	return value, remaining, found
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
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timedOut := false
	budget := timeoutFor(c)
	select {
	case <-done:
	case <-time.After(budget):
		timedOut = true
		killGroup(cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			killGroup(cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
	}
	ms := time.Since(start).Milliseconds()

	o := outcome{ms: ms, stderr: stderr.Bytes(), timedOut: timedOut}
	if timedOut {
		return o
	}
	var res checkkit.Result
	if err := json.Unmarshal(lastJSONLine(stdout.Bytes()), &res); err != nil {
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
