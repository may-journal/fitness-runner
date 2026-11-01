package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/biome"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// eslintFallbackMessage mirrors ESLINT_FALLBACK_MESSAGE in the TS check.
const eslintFallbackMessage = "ESLint reported issues. Run: npx eslint ."

// missingEslintMessage is the install hint when no eslint binary resolves.
const missingEslintMessage = "eslint not found in node_modules/.bin (walking up from the repo root) or on PATH. Run: npm install --save-dev eslint"

// eslintConfigMjs is the shared flat-config filename resolved for --config.
const eslintConfigMjs = "eslint.config.mjs"

// ignoredFileNotice marks results for explicitly-passed ignored files; each
// one fails, since the check judges every tracked file.
const ignoredFileNotice = "File ignored because of a matching ignore pattern"

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "eslint", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !applies(root) {
		return checkkit.Pass(0), nil
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	if biome.Chosen(root) {
		return lintWithBiome(root), nil
	}
	return lintWithEslint(root)
}

// lintWithEslint lints the paths with the resolved eslint binary. A missing
// binary is an internal error carrying the install hint.
func lintWithEslint(root string) (checkkit.Result, error) {
	bin, found := node.FindTool(root, "eslint")
	if !found {
		return checkkit.Result{}, checkkit.Internal(errors.New(missingEslintMessage))
	}
	paths := pathsToLint(root, checkkit.ChangedFiles())
	if len(paths) == 0 {
		return checkkit.Pass(0), nil
	}
	errs, exitCode, filesChecked, err := runEslint(root, bin, paths)
	if err != nil {
		return checkkit.Result{}, err
	}
	return buildExecCheckResult(exitCode, errs, filesChecked)
}

// buildExecCheckResult ports the shared TS helper: ok when exit 0 and no
// errors, failing with the errors otherwise. A failing exit that parsed no
// error is an internal error, since ESLint named nothing to fix.
func buildExecCheckResult(exitCode int, errs []string, filesChecked int) (checkkit.Result, error) {
	if exitCode == 0 && len(errs) == 0 {
		return checkkit.Pass(filesChecked), nil
	}
	if len(errs) == 0 {
		return checkkit.Result{}, checkkit.Internal(errors.New(eslintFallbackMessage))
	}
	return checkkit.Fail(filesChecked, errs...), nil
}

// applies reports whether ESLint has anything to judge: a JS project plus,
// when scoped, at least one changed file it lints.
func applies(root string) bool {
	if !node.Applies(root) || !node.HasSource(root) {
		return false
	}
	changed := checkkit.ChangedFiles()
	return changed == nil || anyLintable(root, changed)
}

// anyLintable reports whether any changed path is one ESLint would lint.
func anyLintable(root string, changed []string) bool {
	for _, p := range changed {
		if lintableStaged(root, p) {
			return true
		}
	}
	return false
}

// pathsToLint returns the changed lintable paths still under root, or every
// tracked lintable file when the run is unscoped.
func pathsToLint(root string, changed []string) []string {
	if changed == nil {
		return walkfs.FilesByExt(root, node.LintableExts...)
	}
	var out []string
	for _, p := range changed {
		if lintableStaged(root, p) {
			out = append(out, p)
		}
	}
	return out
}

// lintableStaged reports whether a changed path should be linted: a lintable
// extension, still existing under root.
func lintableStaged(root, p string) bool {
	return hasLintableExt(p) && exists(filepath.Join(root, p))
}

// hasLintableExt reports whether p ends in one of the endings ESLint lints.
func hasLintableExt(p string) bool {
	for _, ext := range node.LintableExts {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

// runEslint execs the CLI twin of the TS Node-API call, forcing the resolved
// shared config via --config when one materializes.
// A run that could not start, was killed, crashed, or printed no JSON is an
// internal error quoting what ESLint printed.
func runEslint(root, bin string, paths []string) (errs []string, exitCode, filesChecked int, err error) {
	var args []string
	if cfg := sharedconf.Resolve(root, eslintConfigMjs); cfg != "" {
		args = append(args, "--config", cfg)
	}
	args = append(args, "--no-error-on-unmatched-pattern", "--format", "json")
	args = append(args, paths...)
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	o := checkkit.Exec(cmd)
	if envErr := o.EnvErr("eslint"); envErr != nil {
		return nil, 0, 0, envErr
	}
	if results, parsed := parseResults([]byte(o.Stdout)); parsed && lintExit(o) {
		errs, exitCode, filesChecked = summarize(results)
		return errs, exitCode, filesChecked, nil
	}
	return nil, 0, 0, checkkit.Internal(fmt.Errorf("%s (exit %d) Output: %s", eslintFallbackMessage, o.Code, snippet(o)))
}

// lintExit reports whether the process ended with a lint verdict (exit 0 or
// 1), else a tool crash.
func lintExit(o checkkit.Outcome) bool {
	return o.EnvErr("eslint") == nil && o.Code <= 1
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

// summarize folds parsed results into the TS shape, exiting 1 when any file
// has errors or was ignored.
func summarize(results []eslintResult) (errs []string, exitCode, filesChecked int) {
	for _, r := range results {
		if r.ErrorCount > 0 {
			exitCode = 1
		}
		for _, m := range r.Messages {
			errs = append(errs, formatMessage(r.FilePath, m))
		}
	}
	return errs, exitCode, len(results)
}

// formatMessage renders one finding as file:line:col - message (rule); an
// ignored-file notice says to remove the ignore pattern.
func formatMessage(filePath string, m eslintMessage) string {
	if strings.Contains(m.Message, ignoredFileNotice) {
		return filePath + " - ESLint ignores this tracked file; remove the ignore pattern so it is linted"
	}
	rule := ""
	if m.RuleID != "" {
		rule = " (" + m.RuleID + ")"
	}
	return fmt.Sprintf("%s:%d:%d - %s%s", filePath, m.Line, m.Column, m.Message, rule)
}

// snippet is the tool output quoted in the fallback error: stdout then
// stderr (the shared execSyncResult order), else the exec error itself.
func snippet(o checkkit.Outcome) string {
	var parts []string
	for _, s := range []string{o.Stdout, o.Stderr} {
		if t := strings.TrimSpace(s); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 && o.Err != nil {
		return o.Err.Error()
	}
	return strings.Join(parts, "\n")
}

// biomeFallbackMessage stands in when Biome fails without a finding.
const biomeFallbackMessage = "Biome reported issues. Run: npx biome lint ."

// lintWithBiome lints the same paths with `biome lint` in a repo that chose
// Biome (ADR 0006), reporting each finding as path:line:col - message (rule).
func lintWithBiome(root string) checkkit.Result {
	paths := pathsToLint(root, checkkit.ChangedFiles())
	if len(paths) == 0 {
		return checkkit.Pass(0)
	}
	return biome.Check(root, "lint", nil, paths, biomeFallbackMessage, func(f biome.Finding) string {
		return fmt.Sprintf("%s:%d:%d - %s (%s)", f.File, f.Line, f.Col, f.Message, f.Rule)
	})
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
