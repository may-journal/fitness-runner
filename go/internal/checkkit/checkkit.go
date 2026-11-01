// Package checkkit is the contract between the fitness runner and a check
// binary, which prints one JSON Result on stdout.
//
//fitness:topic protocol
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
	// Ok is true when the check found no problem.
	Ok bool `json:"ok"`
	// Errors lists each problem found, one line each.
	Errors []string `json:"errors"`
	// FilesChecked counts the files judged; -1 when the check judged none.
	FilesChecked int `json:"filesChecked"`
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

// Found builds a Result from findings: it fails with one error line for each
// finding, or passes when there are none.
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

// The scopes a check declares: what it reads when it judges a repo.
const (
	// ScopeFiles judges the repo's tracked files.
	ScopeFiles = "files"
	// ScopeBlob judges one text it is given, such as an issue or PR body.
	ScopeBlob = "blob"
	// ScopeGit judges commit messages and the branch's history.
	ScopeGit = "git"
	// ScopeGitHub judges the branch's live GitHub metadata.
	ScopeGitHub = "github"
)

// The comment treatments a check declares: how code comments count when it
// judges a repo.
const (
	// CommentsJudged means comment text can change the check's verdict.
	CommentsJudged = "judged"
	// CommentsStripped means the check drops comments before it judges code.
	// Tool directives in comment form, such as `//go:build`, still count as
	// code.
	CommentsStripped = "stripped"
	// CommentsNotApplicable means the check reads no code comments, only
	// Markdown, manifests, history or GitHub.
	CommentsNotApplicable = "not applicable"
)

// Describe is the metadata a check prints for `--describe`.
type Describe struct {
	// Name is the check name, as `fitness <name>` runs it.
	Name string `json:"name"`
	// Summary says in one line what the check enforces. A built-in check
	// leaves it empty: its package doc comment's first sentence is
	// generated into the runner, which serves it here.
	Summary string `json:"summary,omitempty"`
	// Scope is one of the Scope constants: what the check reads.
	Scope string `json:"scope,omitempty"`
	// Comments is one of the Comments constants: how code comments count.
	// Every built-in check declares it, and `go generate` fails one that
	// does not.
	Comments string `json:"comments,omitempty"`
	// TimeoutMs is the check's execution budget; 0 means the runner default.
	TimeoutMs int `json:"timeoutMs,omitempty"`
	// ContextInlineArg names a flag (e.g. "--message") whose value the runner
	// extracts from single-check argv into FITNESS_CTX_MESSAGE and strips
	// from passthrough. Empty when the check takes no inline context.
	ContextInlineArg string `json:"contextInlineArg,omitempty"`
	// Blob marks a check that judges a text given with `--body-file`, so
	// `fitness --blob` runs it; the rest need files and are skipped there.
	Blob bool `json:"blob,omitempty"`
	// MinFitness is the oldest fitness release, as major.minor.patch, that runs
	// the check as written. An older one stops the run and names the upgrade.
	MinFitness string `json:"minFitness,omitempty"`
}

// Check is one check implementation: its metadata and run function.
type Check struct {
	// Describe is the metadata --describe prints.
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

// ChangedFiles returns the paths the runner scoped this run to, or nil when
// the run is unscoped and every file is judged.
func ChangedFiles() []string {
	if changed := DiffFiles(); changed != nil {
		return changed
	}
	return scopeFiles()
}

// DiffFiles returns the paths a commit or pull request changes, or nil when
// the run changes nothing, so pairing checks judge no pairs.
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

// documented is the summary Document set, which --describe prints for a
// check that declares none.
var documented string

// Document sets the summary --describe prints when the check declares none.
// The runner calls it with the summary generated from the check's package
// doc comment before it starts a built-in check.
func Document(summary string) {
	documented = summary
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
		d := c.Describe
		if d.Summary == "" {
			d.Summary = documented
		}
		_ = enc.Encode(d)
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

// rootFlagValue interprets argv[i] as a --root flag in either form, returning
// its value, span and whether it matched.
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
