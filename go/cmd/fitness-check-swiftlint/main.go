// Command fitness-check-swiftlint runs SwiftLint's strict JSON reporter over
// the repo — the Go port of the swiftlint check. The tool is a system binary
// (brew, never node_modules): it is resolved from PATH at runtime, and a
// missing binary fails with the same install hint the TypeScript check
// printed. The TS check shelled out with `2>&1`; this port feeds one combined
// stdout+stderr buffer to the same judgment: a "command not found" output is
// the install hint, "No lintable files found" passes clean, and otherwise the
// JSON violation array becomes `file:line[:col] - reason (rule_id)` lines
// with a fallback run hint when nothing parses.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	notInstalled    = "SwiftLint not installed: brew install swiftlint"
	fallbackRunHint = "swiftlint reported an error (run: swiftlint lint --strict)"

	commandNotFoundMarker = "command not found"
	noLintableFilesMarker = "No lintable files found"
)

// violation is one entry of SwiftLint's JSON reporter array. Character is a
// pointer because the reporter emits null (or omits it) for file-level rules.
type violation struct {
	Character *int   `json:"character"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Reason    string `json:"reason"`
	RuleID    string `json:"rule_id"`
}

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "swiftlint"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	bin, err := exec.LookPath("swiftlint")
	if err != nil {
		return checkkit.Fail(0, notInstalled), nil
	}
	exitCode, output := execSwiftlint(bin, root)
	return judge(exitCode, output), nil
}

// execSwiftlint runs `swiftlint lint --strict --reporter json --quiet .` in
// root; returns the exit code and combined stdout+stderr, mirroring the TS
// execSyncResult (which redirected with `2>&1`). A failure that carries no
// exit code maps to 1, exactly like the TS catch path.
func execSwiftlint(bin, root string) (exitCode int, output string) {
	cmd := exec.Command(bin, "lint", "--strict", "--reporter", "json", "--quiet", ".")
	cmd.Dir = root
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
			exitCode = exitErr.ExitCode()
		}
	}
	return exitCode, combined.String()
}

// judge ports the TS check verbatim: command-not-found output is the install
// hint, no-lintable-files passes with zero files, and otherwise the exec
// result is judged like buildExecCheckResult — ok only on exit 0 with no
// parsed violations, fallback hint when a failure parsed nothing.
func judge(exitCode int, output string) checkkit.Result {
	if strings.Contains(output, commandNotFoundMarker) {
		return checkkit.Fail(0, notInstalled)
	}
	if strings.Contains(output, noLintableFilesMarker) {
		return checkkit.Pass(0)
	}
	return judgeExecResult(exitCode, parseViolations(output))
}

// judgeExecResult judges the exec outcome like the TS buildExecCheckResult:
// ok only on exit 0 with no parsed violations, the fallback run hint when a
// failure parsed nothing, else the violations themselves.
func judgeExecResult(exitCode int, violations []string) checkkit.Result {
	if exitCode == 0 && len(violations) == 0 {
		return checkkit.Pass(0)
	}
	if len(violations) == 0 {
		return checkkit.Fail(0, fallbackRunHint)
	}
	return checkkit.Fail(0, violations...)
}

// parseViolations parses the JSON reporter's violation array; malformed
// output yields no parsed errors (the caller falls back to the run hint).
func parseViolations(output string) []string {
	var parsed []violation
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return nil
	}
	lines := make([]string, len(parsed))
	for i, v := range parsed {
		lines[i] = formatViolation(v)
	}
	return lines
}

// formatViolation renders one violation as file:line[:col] - reason (rule_id).
func formatViolation(v violation) string {
	col := ""
	if v.Character != nil {
		col = fmt.Sprintf(":%d", *v.Character)
	}
	return fmt.Sprintf("%s:%d%s - %s (%s)", v.File, v.Line, col, v.Reason, v.RuleID)
}
