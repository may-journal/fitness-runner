// Package bodycheck wires the body-oriented fitness checks (plan-structure,
// epic-structure, pr-structure) that validate one GitHub Issue or PR body
// rather than walking files. It resolves the body from a `--body-file` path (`-` meaning stdin),
// the runner's context-inline `--body` value, or piped stdin, and turns a
// validator's messages into a checkkit.Result.
//
//fitness:container cli
//fitness:component kit Checks share it as one engine, so no check keeps its own copy.
package bodycheck

import (
	"io"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/epic"
)

// Run resolves the target body and validates it. With no input it passes with
// zero files checked, so the check stays inert under the file runner;
// otherwise it reports one file whose errors are validate's messages.
func Run(args []string, validate func(body string) []string) (checkkit.Result, error) {
	return RunIf(args, func(string) bool { return true }, validate)
}

// BlobEnv is set by `fitness --blob`, which judges a draft text of any kind.
const BlobEnv = "FITNESS_BLOB"

// RunIf is Run for a check that judges one kind of body. Under `fitness
// --blob`, a text applies rejects passes inert with zero files, so the runner
// lists the check as not applying. Every other caller names the body's kind
// itself, so the check always judges it.
func RunIf(args []string, applies func(body string) bool, validate func(body string) []string) (checkkit.Result, error) {
	body, ok, err := resolveBody(args)
	if err != nil {
		return checkkit.Result{}, err
	}
	if inert(ok, body, applies) {
		return checkkit.Pass(0), nil
	}
	if errs := validate(body); len(errs) > 0 {
		return checkkit.Fail(1, errs...), nil
	}
	return checkkit.Pass(1), nil
}

// inert reports whether there is no body to judge, or a blob the check does
// not apply to.
func inert(ok bool, body string, applies func(string) bool) bool {
	return !ok || os.Getenv(BlobEnv) != "" && !applies(body)
}

// PlanBody reports whether body is a Plan issue: it has the Plan template's
// checklist section, and its steps are not an epic's issue references.
func PlanBody(body string) bool {
	return hasHeading(body, epic.Section) && !epic.Looks(body)
}

// EpicBody reports whether body is an Epic issue: its What needs to happen
// section opens with an issue reference.
func EpicBody(body string) bool {
	return epic.Looks(body)
}

// PRBody reports whether body is a pull request description: it has the PR
// template's changelog heading, at any level.
func PRBody(body string) bool {
	return hasHeading(body, "Changelog")
}

// Closing reports whether body is an issue being closed. A draft never is:
// only close-check and pr-check judge an issue at close time.
func Closing(string) bool {
	return false
}

// hasHeading reports whether body has a heading naming title outside a
// fenced code block, at any level, in any case, with or without a trailing
// colon, so the structure check names a wrong level or colon. A text with no
// such heading is of another kind, and the check does not apply.
func hasHeading(body, title string) bool {
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		}
		if !fenced && strings.EqualFold(headingText(line), title) {
			return true
		}
	}
	return false
}

// headingText returns a heading line's text without its # marks and a
// trailing colon, or "" for any other line.
func headingText(line string) string {
	text, isHeading := strings.CutPrefix(line, "#")
	if !isHeading {
		return ""
	}
	return strings.Trim(text, "# :")
}

// RunDoc runs a file-walking doc check in body mode when an explicit
// `--body-file` document is supplied, resolving config from root exactly as
// the file walk would. It returns handled=false when no `--body-file` is
// present, so the caller falls back to its normal file walk — the trigger is
// explicit only, never the ambient stdin or context message the suite may
// carry, so a normal suite run keeps walking files.
func RunDoc(root string, args []string, validate func(root, content string) []string) (result checkkit.Result, handled bool, err error) {
	content, ok, err := BodyFile(args)
	if err != nil {
		return checkkit.Result{}, false, err
	}
	if !ok {
		return checkkit.Result{}, false, nil
	}
	if errs := validate(root, content); len(errs) > 0 {
		return checkkit.Fail(1, errs...), true, nil
	}
	return checkkit.Pass(1), true, nil
}

// BodyFile resolves a document only from an explicit `--body-file` path (`-`
// meaning stdin), returning ok=false when the flag is absent. Unlike
// resolveBody it never falls back to the context message or ambient stdin, so
// doc checks enter body mode only when a caller deliberately points them at a
// file.
func BodyFile(args []string) (content string, ok bool, err error) {
	path, has := bodyFileArg(args)
	if !has {
		return "", false, nil
	}
	if path == "-" {
		return readAll(os.Stdin)
	}
	raw, readErr := os.ReadFile(path)
	return string(raw), true, readErr
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
