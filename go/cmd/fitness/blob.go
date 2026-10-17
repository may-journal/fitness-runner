package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// blobFlag runs every blob-capable check on a text instead of the repo.
const blobFlag = "--blob"

// usageExit is the exit code for a command line the runner cannot run.
const usageExit = 2

// blobRequest is a parsed `fitness --blob` command line.
type blobRequest struct {
	text   string
	checks []string
	// values holds each --blob value seen; exactly one is valid.
	values []string
}

// hasBlobFlag reports whether argv asks for blob mode.
func hasBlobFlag(argv []string) bool {
	for _, a := range argv {
		if a == blobFlag || strings.HasPrefix(a, blobFlag+"=") {
			return true
		}
	}
	return false
}

// runBlob runs the blob-capable checks on the text argv names, listing the
// checks that need files as skipped. Beyond the repo's config, it never
// reads repo files, the staged diff, or a PR, so findings come from the text.
func runBlob(argv []string) int {
	req, err := parseBlob(argv, os.Stdin)
	if err != nil {
		report.Error("fitness --blob", err)
		return usageExit
	}
	root, err := os.Getwd()
	if err != nil {
		report.Error("fitness --blob", err)
		return 1
	}
	run, skipped, err := blobChecks(req.checks)
	if err != nil {
		report.Error("fitness --blob", err)
		return usageExit
	}
	cfg, err := loadConfig(root)
	if err != nil {
		report.Error("fitness --blob", err)
		return 1
	}
	run = withBudgets(cfg, run)
	return judgeBlob(root, req.text, run, skipped, blobEnv(run, req.checks != nil))
}

// judgeBlob saves text where the checks read it with --body-file, runs them
// with env, and prints the usual summary with the skipped checks named.
func judgeBlob(root, text string, checks []resolved, skipped, env []string) int {
	file, err := metadata.WriteBody(text)
	if err != nil {
		report.Error("fitness --blob", err)
		return 1
	}
	defer os.Remove(file)
	start := time.Now()
	fmt.Fprintln(os.Stderr, "Running checks:")
	outcomes := runPool(root, checks, []string{"--body-file", file}, env)
	checks, outcomes, notApplying := applying(checks, outcomes)
	rows, success, failure, files := summarize(checks, outcomes)
	elapsed := time.Since(start).Milliseconds()
	printSummary(rows, success, failure, files, elapsed)
	out := os.Stdout
	if failure > 0 {
		out = os.Stderr
	}
	listSkipped(out, "needs files", skipped)
	listSkipped(out, "does not apply to this text", notApplying)
	return finish(rows, success, failure, files, elapsed)
}

// applying drops the checks that passed inert with zero files, such as
// plan-structure on a PR description, and returns their names.
func applying(checks []resolved, outcomes []outcome) ([]resolved, []outcome, []string) {
	var ran []resolved
	var results []outcome
	var inert []string
	for i, c := range checks {
		if inertOutcome(outcomes[i]) {
			inert = append(inert, c.name)
			continue
		}
		ran, results = append(ran, c), append(results, outcomes[i])
	}
	return ran, results, inert
}

// inertOutcome reports a clean pass that judged no file.
func inertOutcome(o outcome) bool {
	return o.res.Ok && o.res.FilesChecked == 0 && !o.timedOut && o.crashed == 0
}

// listSkipped names the skipped checks and why, when there are any.
func listSkipped(out *os.File, why string, names []string) {
	if len(names) > 0 {
		fmt.Fprintf(out, "Skipped (%s): %s\n", why, strings.Join(names, ", "))
	}
}

// parseBlob reads `--blob <text>`, `--blob=<text>`, or `--blob -` (stdin),
// plus an optional `--checks=a,b`. Anything else, such as --all or a check
// name, is a usage error, and so is an empty text.
func parseBlob(argv []string, stdin io.Reader) (blobRequest, error) {
	var req blobRequest
	for i := 0; i < len(argv); i++ {
		consumed, err := req.take(argv, i)
		if err != nil {
			return req, err
		}
		i += consumed - 1
	}
	if len(req.values) != 1 {
		return req, errors.New("use --blob once")
	}
	return withText(req, req.values[0], stdin)
}

// take consumes argv[i], a --blob value or a --checks list, and returns how
// many tokens it used; anything else is a usage error.
func (r *blobRequest) take(argv []string, i int) (int, error) {
	if v, consumed, ok := blobValue(argv, i); ok {
		r.values = append(r.values, v)
		return consumed, nil
	}
	if list, ok := strings.CutPrefix(argv[i], "--checks="); ok && r.checks == nil {
		r.checks = strings.Split(list, ",")
		return 1, nil
	}
	if argv[i] == blobFlag {
		return 0, errors.New("--blob needs a text, or - to read stdin")
	}
	return 0, fmt.Errorf("--blob runs alone or with --checks=name,name; remove %q", argv[i])
}

// blobValue reads `--blob=<text>` or `--blob <text>` at argv[i], with how
// many tokens it spans.
func blobValue(argv []string, i int) (string, int, bool) {
	if v, ok := strings.CutPrefix(argv[i], blobFlag+"="); ok {
		return v, 1, true
	}
	if argv[i] == blobFlag && textFollows(argv, i) {
		return argv[i+1], 2, true
	}
	return "", 0, false
}

// textFollows reports whether a text follows argv[i]. A next token of
// --checks=… means the text was left out; any other token, front matter's
// --- included, is the text.
func textFollows(argv []string, i int) bool {
	return i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--checks=")
}

// withText fills in the text, reading stdin for "-", and rejects an empty
// one.
func withText(req blobRequest, value string, stdin io.Reader) (blobRequest, error) {
	if value == "-" {
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return req, err
		}
		value = string(raw)
	}
	if strings.TrimSpace(value) == "" {
		return req, errors.New("--blob needs a nonempty text")
	}
	req.text = value
	return req, nil
}

// blobChecks resolves the catalog, or the --checks selection, and splits it
// into the checks that declare blob input and the names of those that need
// files.
func blobChecks(selected []string) (run []resolved, skipped []string, err error) {
	names, err := blobSelection(selected)
	if err != nil {
		return nil, nil, err
	}
	checks, err := resolveList(names)
	if err != nil {
		return nil, nil, err
	}
	run, skipped = splitBlob(checks)
	if len(run) == 0 {
		return nil, nil, errors.New("--checks names no check that can judge a text")
	}
	return run, skipped, nil
}

// splitBlob splits checks into those that declare blob input and the names
// of those that need files.
func splitBlob(checks []resolved) (run []resolved, skipped []string) {
	for _, c := range checks {
		if c.desc.Blob {
			run = append(run, c)
		} else {
			skipped = append(skipped, c.name)
		}
	}
	return run, skipped
}

// blobSelection returns the names to consider: every check, or the
// --checks list when it names only known checks, each once.
func blobSelection(selected []string) ([]string, error) {
	if selected == nil {
		return allChecks, nil
	}
	for i, name := range selected {
		if !slices.Contains(allChecks, name) || slices.Contains(selected[:i], name) {
			return nil, fmt.Errorf("--checks names an unknown, empty, or repeated check %q", name)
		}
	}
	return selected, nil
}

// blobEnv is the environment blob checks get: no changed or staged files and
// no commit message, so nothing but the text reaches them. Without --checks
// it sets FITNESS_BLOB, so a body check skips text of another kind; a check
// named in --checks always judges the text.
func blobEnv(checks []resolved, chosen bool) []string {
	env := os.Environ()
	for _, name := range []string{"FITNESS_CHANGED_FILES", "FITNESS_STAGED_FILES", "FITNESS_CTX_MESSAGE", bodycheck.BlobEnv} {
		env = withoutVar(env, name)
	}
	names := make([]string, len(checks))
	for i, c := range checks {
		names[i] = c.name
	}
	env = append(env, "FITNESS_ENABLED_CHECKS="+strings.Join(names, "\n"))
	if chosen {
		return env
	}
	return append(env, bodycheck.BlobEnv+"=1")
}
