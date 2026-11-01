// Package swiftlint runs SwiftLint's strict JSON reporter over a repo for the
// swiftlint check, resolving the binary from PATH.
package swiftlint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

const (
	notInstalled = "SwiftLint not installed: brew install swiftlint"

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

// Run lints the repo at root, passing clean when it holds no Swift source.
// A scoped run, such as a pull request, lints only its changed Swift files,
// and passes without starting SwiftLint when it changed none.
func Run(root string, _ []string) (checkkit.Result, error) {
	all := walkfs.FilesByExt(root, swift.Ext)
	if len(all) == 0 {
		return checkkit.Pass(0), nil
	}
	paths, count := lintPaths(root, all)
	if len(paths) == 0 {
		return checkkit.Pass(0), nil
	}
	bin, err := exec.LookPath("swiftlint")
	if err != nil {
		return checkkit.Result{}, checkkit.Internal(errors.New(notInstalled))
	}
	return judge(execSwiftlint(bin, root, paths), count)
}

// lintPaths is what SwiftLint lints and how many Swift files that covers: the
// whole tree, or a scoped run's changed Swift files.
func lintPaths(root string, all []string) ([]string, int) {
	if checkkit.ChangedFiles() == nil || includes(root) {
		return []string{"."}, len(all)
	}
	changed := walkfs.InScope(all)
	if len(changed) == 0 {
		return nil, 0
	}
	// --force-exclude drops a named path the config excludes.
	return append([]string{"--force-exclude"}, changed...), len(changed)
}

// includedKey is a top-level `included:` list in a SwiftLint config.
var includedKey = regexp.MustCompile(`(?m)^included\s*:`)

// includes reports whether the repo's .swiftlint.yml lists `included:`
// paths. SwiftLint lints a path named on its command line even outside
// that list, so such a repo keeps the whole-tree lint.
func includes(root string) bool {
	config, err := os.ReadFile(filepath.Join(root, ".swiftlint.yml"))
	return err == nil && includedKey.Match(config)
}

// execSwiftlint runs `swiftlint lint --strict --reporter json --quiet` on paths
// in root, returning its outcome with stdout and stderr in one buffer.
func execSwiftlint(bin, root string, paths []string) checkkit.Outcome {
	cmd := exec.Command(bin, append([]string{"lint", "--strict", "--reporter", "json", "--quiet"}, paths...)...)
	cmd.Dir = root
	return checkkit.ExecCombined(cmd)
}

// judge turns a run that could not start into an internal error with the
// install hint, and otherwise judges the exec result.
func judge(o checkkit.Outcome, count int) (checkkit.Result, error) {
	if err := o.EnvErr("swiftlint"); err != nil {
		return checkkit.Result{}, err
	}
	if strings.Contains(o.Stdout, commandNotFoundMarker) {
		return checkkit.Result{}, checkkit.Internal(errors.New(notInstalled))
	}
	if strings.Contains(o.Stdout, noLintableFilesMarker) {
		return checkkit.Pass(0), nil
	}
	return judgeExecResult(o.Code, parseViolations(o.Stdout), o.Stdout, count)
}

// judgeExecResult judges the exec outcome like the TS buildExecCheckResult:
// ok only on exit 0 with no parsed violations, else the violations
// themselves. A failure that parsed nothing is an internal error quoting
// what SwiftLint printed, since it named nothing to fix.
func judgeExecResult(exitCode int, violations []string, output string, count int) (checkkit.Result, error) {
	if exitCode == 0 && len(violations) == 0 {
		return checkkit.Pass(count), nil
	}
	if len(violations) == 0 {
		return checkkit.Result{}, checkkit.NoVerdict("swiftlint lint --strict", exitCode, output)
	}
	return checkkit.Fail(count, violations...), nil
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
