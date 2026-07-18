// Package checkkit is the contract between the fitness runner and a check
// binary, plus the main-function scaffolding every check builds on.
//
// Protocol: the runner execs `fitness-check-<name> --root <dir> [args…]` with
// cwd = root and context in FITNESS_* environment variables. The check prints
// exactly one JSON Result on stdout; anything meant for humans (banners, tool
// passthrough) goes to stderr. Exit code 0 when the check ran (ok true or
// false — the Result carries the judgment); non-zero means the check crashed
// and the runner synthesizes a failure. `--describe` prints a Describe JSON
// and exits without running.
package checkkit

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Result is the single JSON object a check prints on stdout.
type Result struct {
	Ok           bool     `json:"ok"`
	Errors       []string `json:"errors"`
	FilesChecked int      `json:"filesChecked"`
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
}

// Check is one check implementation: its metadata and run function.
type Check struct {
	Describe Describe
	// Run receives the repo root and passthrough args; the returned Result
	// is emitted verbatim. An error return means the check crashed.
	Run func(root string, args []string) (Result, error)
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

// EnabledChecks returns the enabled check names for this run, in order.
func EnabledChecks() []string {
	return splitLines(os.Getenv("FITNESS_ENABLED_CHECKS"))
}

// CheckName returns the name the runner resolved this binary as — flavor
// packages branch on it. Empty when run standalone.
func CheckName() string {
	return os.Getenv("FITNESS_CHECK_NAME")
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
	result, err := c.Run(root, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", c.Describe.Name, err)
		return 2
	}
	return emitResult(enc, result)
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
