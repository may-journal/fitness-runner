// Package bodycheck wires the body-oriented fitness checks (plan-structure,
// pr-structure) that validate one GitHub Issue or PR body rather than walking
// files. It resolves the body from a `--body-file` path (`-` meaning stdin),
// the runner's context-inline `--body` value, or piped stdin, and turns a
// validator's messages into a checkkit.Result.
package bodycheck

import (
	"io"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Run resolves the target body and validates it. With no input it passes with
// zero files checked, so the check stays inert under the file runner;
// otherwise it reports one file whose errors are validate's messages.
func Run(args []string, validate func(body string) []string) (checkkit.Result, error) {
	body, ok, err := resolveBody(args)
	if err != nil {
		return checkkit.Result{}, err
	}
	if !ok {
		return checkkit.Pass(0), nil
	}
	if errs := validate(body); len(errs) > 0 {
		return checkkit.Fail(1, errs...), nil
	}
	return checkkit.Pass(1), nil
}

// resolveBody returns the body text and whether any input was supplied, in
// priority order: --body-file (- means stdin), the context-inline body
// (FITNESS_CTX_MESSAGE), then piped stdin. A terminal stdin is treated as no
// input so the check stays non-blocking under the file runner and in tests.
func resolveBody(args []string) (string, bool, error) {
	if path, ok := bodyFileArg(args); ok {
		if path == "-" {
			return readAll(os.Stdin)
		}
		raw, err := os.ReadFile(path)
		return string(raw), true, err
	}
	if msg, ok := checkkit.CtxMessage(); ok {
		return msg, true, nil
	}
	if stdinPiped() {
		return readAll(os.Stdin)
	}
	return "", false, nil
}

func readAll(r io.Reader) (string, bool, error) {
	raw, err := io.ReadAll(r)
	return string(raw), true, err
}

// bodyFileArg extracts a --body-file value in either form.
func bodyFileArg(args []string) (string, bool) {
	for i, a := range args {
		if a == "--body-file" && i+1 < len(args) {
			return args[i+1], true
		}
		if v, ok := strings.CutPrefix(a, "--body-file="); ok {
			return v, true
		}
	}
	return "", false
}

// stdinPiped reports whether stdin is a pipe or file rather than a terminal.
func stdinPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}
