// Command fitness-check-eslint lints the repo with the shared flat config —
// the Go port of the eslint check. The TypeScript check ran ESLint in-process
// through its Node API; this binary execs the real eslint CLI instead,
// resolved from node_modules/.bin walking up from the root, then PATH — never
// npx. The judgment is ported verbatim: staged-path filtering, JSON result
// parsing, message formatting, ignore-notice filtering, and the fallback
// error strings. Syntactic parse only (see ADR 0001), so the runner-default
// timeout stands.
//
// The shared eslint.config.mjs forced via --config resolves through
// internal/sharedconf: the repo's own file wins, then an installed
// node_modules/@mayjournal/fitness-shared, then the copy embedded in this
// binary, materialized to the cache. The shared config imports its plugins
// (typescript-eslint, jsdoc, perfectionist, prettier) as bare specifiers, so
// the materialized copy only loads in repos where Node can resolve those
// packages — installing them alongside eslint stays the check's peer
// contract, exactly as it was in the npm era.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
)

// eslintFallbackMessage mirrors ESLINT_FALLBACK_MESSAGE in the TS check.
const eslintFallbackMessage = "ESLint reported issues. Run: npx eslint ."

// missingEslintMessage is the install hint when no eslint binary resolves.
const missingEslintMessage = "eslint not found in node_modules/.bin (walking up from the repo root) or on PATH. Run: npm install --save-dev eslint"

// eslintConfigMjs is the shared flat-config filename resolved for --config.
const eslintConfigMjs = "eslint.config.mjs"

// ignoredFileNotice marks results for explicitly-passed ignored files; the TS
// check dropped these messages while still counting the file.
const ignoredFileNotice = "File ignored because of a matching ignore pattern"

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "eslint"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	bin, found := findEslint(root)
	if !found {
		return checkkit.Fail(0, missingEslintMessage), nil
	}
	paths := pathsToLint(root, checkkit.StagedFiles())
	errs, exitCode, filesChecked := runEslint(root, bin, paths)
	return buildExecCheckResult(exitCode, errs, filesChecked), nil
}

// buildExecCheckResult ports the shared TS helper: ok when exit 0 and no
// errors; the fallback message stands in when a failure parsed no errors.
func buildExecCheckResult(exitCode int, errs []string, filesChecked int) checkkit.Result {
	if exitCode == 0 && len(errs) == 0 {
		return checkkit.Pass(filesChecked)
	}
	if len(errs) == 0 {
		errs = []string{eslintFallbackMessage}
	}
	return checkkit.Fail(filesChecked, errs...)
}

var (
	lintableExt     = regexp.MustCompile(`\.(cjs|js|mjs|tsx?)$`)
	ignoredByEslint = regexp.MustCompile(`\.(test|spec)\.(cjs|js|mjs|ts|tsx)$`)
)

// pathsToLint ports getPathsToLint: staged lintable existing paths under
// root — skipping declaration and test/spec files — or ["."] when none.
func pathsToLint(root string, staged []string) []string {
	var out []string
	for _, p := range staged {
		if lintableExt.MatchString(p) && !strings.HasSuffix(p, ".d.ts") &&
			!ignoredByEslint.MatchString(p) && exists(filepath.Join(root, p)) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"."}
	}
	return out
}

// runEslint execs the CLI twin of the TS Node-API invocation: cwd root, the
// resolved shared config forced via --config (repo-local file, installed
// @mayjournal/fitness-shared, or the embedded copy materialized on demand —
// sharedconf.Resolve), unmatched patterns tolerated, JSON out. In the
// vanishingly rare case nothing materializes, eslint runs on its own config
// discovery instead.
func runEslint(root, bin string, paths []string) (errs []string, exitCode, filesChecked int) {
	var args []string
	if cfg := sharedconf.Resolve(root, eslintConfigMjs); cfg != "" {
		args = append(args, "--config", cfg)
	}
	args = append(args, "--no-error-on-unmatched-pattern", "--format", "json")
	args = append(args, paths...)
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if results, parsed := parseResults(stdout.Bytes()); parsed && lintExit(runErr) {
		return summarize(results)
	}
	return []string{eslintFallbackMessage, "Output: " + snippet(&stdout, &stderr, runErr)}, 1, 0
}

// lintExit reports whether the process ended with a lint verdict (exit 0 or
// 1); anything else is a tool crash and takes the fallback path.
func lintExit(err error) bool {
	if err == nil {
		return true
	}
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}

// eslintMessage and eslintResult mirror the JSON-formatter fields the TS
// check read; a null ruleId decodes as the empty string.
type eslintMessage struct {
	Column  int    `json:"column"`
	Line    int    `json:"line"`
	Message string `json:"message"`
	RuleID  string `json:"ruleId"`
}

type eslintResult struct {
	ErrorCount int             `json:"errorCount"`
	FilePath   string          `json:"filePath"`
	Messages   []eslintMessage `json:"messages"`
}

// parseResults ports tryParseJsonArray: a JSON array parses, anything else
// (including null or an object) does not.
func parseResults(out []byte) ([]eslintResult, bool) {
	trimmed := bytes.TrimSpace(out)
	if !bytes.HasPrefix(trimmed, []byte("[")) {
		return nil, false
	}
	var results []eslintResult
	if json.Unmarshal(trimmed, &results) != nil {
		return nil, false
	}
	return results, true
}

// summarize folds parsed results into the TS shape: formatted messages minus
// explicit-ignore notices, exit 1 when any file has errors, files = results.
func summarize(results []eslintResult) (errs []string, exitCode, filesChecked int) {
	for _, r := range results {
		if r.ErrorCount > 0 {
			exitCode = 1
		}
		for _, m := range r.Messages {
			if strings.Contains(m.Message, ignoredFileNotice) {
				continue
			}
			errs = append(errs, formatMessage(r.FilePath, m))
		}
	}
	return errs, exitCode, len(results)
}

// formatMessage renders one finding as file:line:col - message (rule).
func formatMessage(filePath string, m eslintMessage) string {
	rule := ""
	if m.RuleID != "" {
		rule = " (" + m.RuleID + ")"
	}
	return fmt.Sprintf("%s:%d:%d - %s%s", filePath, m.Line, m.Column, m.Message, rule)
}

// snippet is the tool output quoted in the fallback error: stdout then
// stderr (the shared execSyncResult order), else the exec error itself.
func snippet(stdout, stderr *bytes.Buffer, runErr error) string {
	var parts []string
	for _, s := range []string{stdout.String(), stderr.String()} {
		if t := strings.TrimSpace(s); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 && runErr != nil {
		return runErr.Error()
	}
	return strings.Join(parts, "\n")
}

// findEslint resolves the eslint binary: node_modules/.bin walking up from
// root, then PATH — never npx.
func findEslint(root string) (string, bool) {
	for d := root; ; {
		bin := filepath.Join(d, "node_modules", ".bin", "eslint")
		if isExecutable(bin) {
			return bin, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	if bin, err := exec.LookPath("eslint"); err == nil {
		return bin, true
	}
	return "", false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
