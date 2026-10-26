// Package checkkit is the contract between the fitness runner and a check
// binary, plus the main-function scaffolding every check builds on.
//
// Protocol: the runner execs `fitness-check-<name> --root <dir> [args…]` with
// cwd = root and context in FITNESS_* environment variables. The check prints
// exactly one JSON Result on stdout; anything meant for humans (banners, tool
// passthrough) goes to stderr. Exit code 0 when the check passed and 1 when
// it found problems. Exit 3 means the check could not judge (a panic or an
// internal error) and the Result carries that as info, which never fails a
// run. `--describe` prints a Describe JSON and exits without running.
package checkkit

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ExitInternal is the exit code of a check that could not judge the repo.
const ExitInternal = 3

// Result is the single JSON object a check prints on stdout.
type Result struct {
	Ok           bool     `json:"ok"`
	Errors       []string `json:"errors"`
	FilesChecked int      `json:"filesChecked"`
	// Info carries notes that never fail a run, such as an internal error.
	Info []string `json:"info,omitempty"`
	// Findings repeats Errors in a shape an agent can act on without
	// parsing prose: where, which rule, and how to fix it. It is optional,
	// so a check that sets none still speaks the protocol, and Errors
	// stays for the readers that predate it.
	Findings []Finding `json:"findings,omitempty"`
}

// Finding is one problem a check found.
type Finding struct {
	// File is the path the problem is in, relative to the repo root, or
	// empty when it is not in one file.
	File string `json:"file,omitempty"`
	// Line is the 1-based line in File, or 0 when the whole file is meant.
	Line int `json:"line,omitempty"`
	// Column is the 1-based column on Line, or 0 when not known.
	Column int `json:"column,omitempty"`
	// Rule names what the check enforces here, in lowercase words joined
	// by dashes, such as unknown-word.
	Rule string `json:"rule"`
	// Message says what is wrong, in words a person reads.
	Message string `json:"message"`
	// Fix says what to change so the check passes, when one change does.
	Fix string `json:"fix,omitempty"`
	// Explain is the command that prints why the check exists. The check's
	// entry point fills it in from the check's name.
	Explain string `json:"explain,omitempty"`
}

// ExplainCommand is the command that prints why check exists.
func ExplainCommand(check string) string {
	return "fitness explain " + check
}

// Found builds a Result from findings: failing with one error line each,
// rendered by line in the check's own words, or passing with none.
func Found(filesChecked int, line func(Finding) string, findings ...Finding) Result {
	if len(findings) == 0 {
		return Pass(filesChecked)
	}
	res := Fail(filesChecked, Lines(line, findings)...)
	res.Findings = findings
	return res
}

// Lines renders each finding as an error line with line.
func Lines(line func(Finding) string, findings []Finding) []string {
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = line(f)
	}
	return out
}

// Describe is the metadata a check prints for `--describe`.
type Describe struct {
	Name string `json:"name"`
	// TimeoutMs is the check's execution budget; 0 means the runner default.
	TimeoutMs int `json:"timeoutMs,omitempty"`
	// ContextInlineArg names a flag (e.g. "--message") whose value the runner
	// extracts from single-check argv into FITNESS_CTX_MESSAGE and strips
	// from passthrough. Empty when the check takes no inline context.
	ContextInlineArg string `json:"contextInlineArg,omitempty"`
	// Blob marks a check that judges a text given with `--body-file`, so
	// `fitness --blob` runs it; the rest need files and are skipped there.
	Blob bool `json:"blob,omitempty"`
}

// Check is one check implementation: its metadata and run function.
type Check struct {
	Describe Describe
	// Run receives the repo root and passthrough args; the returned Result
	// is emitted verbatim. An error return means the check could not judge:
	// it is reported as info with exit 3, never as a finding.
	Run func(root string, args []string) (Result, error)
}

// Info builds a passing Result that carries info notes.
func Info(filesChecked int, msgs ...string) Result {
	return Result{Ok: true, Errors: []string{}, FilesChecked: filesChecked, Info: msgs}
}

// Warn adds each warning to r as an info note, then closing, so a rule
// still being staged informs without failing. No warnings add nothing.
func (r *Result) Warn(warnings []string, closing string) {
	if len(warnings) == 0 {
		return
	}
	for _, w := range warnings {
		r.Info = append(r.Info, "warning: "+w)
	}
	r.Info = append(r.Info, closing)
}

// Pass builds a passing Result.
func Pass(filesChecked int) Result {
	return Result{Ok: true, Errors: []string{}, FilesChecked: filesChecked}
}

// Fail builds a failing Result.
func Fail(filesChecked int, errors ...string) Result {
	return Result{Ok: false, Errors: errors, FilesChecked: filesChecked}
}

// StagedFiles returns the staged paths the runner provided (may be empty).
func StagedFiles() []string {
	return splitLines(os.Getenv("FITNESS_STAGED_FILES"))
}

// ScopeFileEnv names the file, one path per line, that lists the files under
// a `fitness --dir` folder. A folder can list more paths than one variable
// holds, so the runner hands them over in a file.
const ScopeFileEnv = "FITNESS_SCOPE_FILE"

// ChangedFiles returns the paths the runner scoped this run to: the staged
// files at commit time, a pull request's diff against its base, or the files
// under a `--dir` folder. Nil means the run is unscoped and every file is
// judged.
func ChangedFiles() []string {
	if changed := DiffFiles(); changed != nil {
		return changed
	}
	return scopeFiles()
}

// DiffFiles returns the paths a commit or pull request changes. Nil means
// the run changes nothing, as in a full scan or a `--dir` folder, so a
// check that pairs changes, such as requirements, judges no pairs.
func DiffFiles() []string {
	return splitLines(os.Getenv("FITNESS_CHANGED_FILES"))
}

// scopeFiles reads the `--dir` folder's file list, nil when there is none.
func scopeFiles() []string {
	path := os.Getenv(ScopeFileEnv)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return splitLines(string(data))
}

// EnabledChecks returns the enabled check names for this run, in order.
func EnabledChecks() []string {
	return splitLines(os.Getenv("FITNESS_ENABLED_CHECKS"))
}

// CtxMessage returns the context-inline commit message and whether the
// runner provided one at all (present-but-empty differs from absent).
func CtxMessage() (string, bool) {
	return os.LookupEnv("FITNESS_CTX_MESSAGE")
}

func splitLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// Main is the entry point every check binary delegates to: it parses
// --root/--describe, runs the check, and emits the protocol JSON.
func Main(c Check) {
	shareCPUs()
	os.Exit(runMain(c, os.Args[1:], os.Stdout))
}

func runMain(c Check, argv []string, stdout *os.File) int {
	root, args, describe := parseArgs(argv)
	enc := json.NewEncoder(stdout)
	if describe {
		_ = enc.Encode(c.Describe)
		return 0
	}
	if root == "" {
		root = mustGetwd()
	}
	result, err := safeRun(c, root, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", c.Describe.Name, err)
		_ = enc.Encode(Info(-1, err.Error()))
		return ExitInternal
	}
	return emitResult(enc, withExplain(result, c.Describe.Name))
}

// withExplain points each finding that names no explain command at the
// check's own.
func withExplain(result Result, check string) Result {
	for i := range result.Findings {
		if result.Findings[i].Explain == "" {
			result.Findings[i].Explain = ExplainCommand(check)
		}
	}
	return result
}

// safeRun runs the check, turning a panic into an internal error.
func safeRun(c Check, root string, args []string) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Internal(fmt.Errorf("check panicked: %v", r))
		}
	}()
	return c.Run(root, args)
}

// emitResult normalizes a nil Errors slice to empty, encodes the Result on
// enc, and returns the process exit code (0 passed, 1 failed).
func emitResult(enc *json.Encoder, result Result) int {
	if result.Errors == nil {
		result.Errors = []string{}
	}
	_ = enc.Encode(result)
	if result.Ok {
		return 0
	}
	return 1
}

// parseArgs pulls --root <dir> / --root=<dir> and --describe out of argv;
// everything else passes through to the check in order.
func parseArgs(argv []string) (root string, args []string, describe bool) {
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "--describe" {
			describe = true
			continue
		}
		if value, consumed, ok := rootFlagValue(argv, i); ok {
			root = value
			i += consumed - 1
			continue
		}
		args = append(args, a)
	}
	return root, args, describe
}

// rootFlagValue interprets argv[i] as a --root flag in either form
// (--root <dir> or --root=<dir>): it returns the root value, how many argv
// entries the flag spanned, and whether argv[i] was a root flag at all.
// A trailing --root with no value is not a root flag (it passes through).
func rootFlagValue(argv []string, i int) (value string, consumed int, ok bool) {
	a := argv[i]
	if a == "--root" && i+1 < len(argv) {
		return argv[i+1], 2, true
	}
	if strings.HasPrefix(a, "--root=") {
		return strings.TrimPrefix(a, "--root="), 1, true
	}
	return "", 0, false
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
