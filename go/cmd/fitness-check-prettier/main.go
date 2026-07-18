// Command fitness-check-prettier runs the real Prettier over the repo — the
// Go port of the prettier check. The binary stays zero-dependency: Prettier
// is resolved at runtime as a peer of the repo under check (node_modules/.bin
// walking up from root, then PATH — never npx), and a missing binary fails
// with a one-line install hint. Everything else is the TS check verbatim:
// staged paths are filtered (skip-list entries, .mdc, and the githooks/,
// scripts/, go/ trees Prettier cannot parse), the glob "." runs when nothing
// is staged, passthrough args replace --check mode, [warn] lines become
// per-file errors, and a non-zero exit that parsed nothing reports the
// fallback message. When the repo has no Prettier config of its own, the
// shared prettier.config.cjs bundled with @mayjournal/fitness-shared is
// passed via --config (found by the same node_modules walk, mirroring the TS
// resolveFitnessConfigPath); when neither exists the check runs on Prettier
// defaults.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	// prettierCLI names the tool in user-facing messages exactly like the TS
	// check's PRETTIER_CLI (install hints may still say npx; the binary never
	// execs it).
	prettierCLI             = "npx prettier"
	prettierFallbackMessage = "Prettier reported issues. Run: " + prettierCLI + " . --write"
	notInstalled            = "Prettier not installed: npm install --save-dev prettier"

	warnPrefix        = "[warn] "
	codeStyleSummary  = "Code style issues"
	packageJSONName   = "package.json"
	prettierConfigCjs = "prettier.config.cjs"
)

// prettierConfigNames are the consumer config files that suppress the shared
// --config, same list as the TS check.
var prettierConfigNames = []string{
	".prettierrc",
	".prettierrc.json",
	".prettierrc.yml",
	".prettierrc.yaml",
	".prettierrc.js",
	".prettierrc.cjs",
	".prettierrc.mts",
	".prettierrc.cts",
	".prettierrc.ts",
	"prettier.config.js",
	prettierConfigCjs,
	"prettier.config.mjs",
	"prettier.config.mts",
	"prettier.config.cts",
	"prettier.config.ts",
}

// prettierSkipStaged are staged paths never handed to Prettier (no parser or
// covered by ignore files) — the TS PRETTIER_SKIP_STAGED set.
var prettierSkipStaged = map[string]bool{
	".gitignore":          true,
	"githooks/commit-msg": true,
	".npmrc":              true,
	".prettierignore":     true,
	"LICENSE":             true,
	"package-lock.json":   true,
}

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "prettier"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	bin := resolvePrettierBin(root)
	if bin == "" {
		return checkkit.Fail(0, notInstalled), nil
	}
	configPath := resolveConfigPath(root)
	paths := pathsToCheck(root, checkkit.StagedFiles())
	exitCode, output := execPrettier(bin, root, prettierArgv(configPath, args, paths))
	parsed := parsePrettierOutput(output)
	return buildExecResult(exitCode, parsed, filesChecked(parsed, paths)), nil
}

// walkUp calls fn on root (absolute) and each ancestor, returning fn's first
// non-empty answer; "" when the filesystem root is passed without a hit.
func walkUp(root string, fn func(dir string) string) string {
	dir := root
	if abs, err := filepath.Abs(root); err == nil {
		dir = abs
	}
	for {
		if p := fn(dir); p != "" {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// resolvePrettierBin finds the Prettier executable: node_modules/.bin walking
// up from root, then PATH. Empty when not installed anywhere.
func resolvePrettierBin(root string) string {
	if bin := walkUp(root, func(dir string) string {
		p := filepath.Join(dir, "node_modules", ".bin", "prettier")
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return p
		}
		return ""
	}); bin != "" {
		return bin
	}
	bin, err := exec.LookPath("prettier")
	if err != nil {
		return ""
	}
	return bin
}

// hasPrettierInPackageJSON reports a non-null "prettier" field in
// package.json; unreadable or invalid JSON is false, like the TS check.
func hasPrettierInPackageJSON(root string) bool {
	raw, err := os.ReadFile(filepath.Join(root, packageJSONName))
	if err != nil {
		return false
	}
	var pkg struct {
		Prettier any `json:"prettier"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return false
	}
	return pkg.Prettier != nil
}

// hasPrettierConfig reports a Prettier config file or package.json
// "prettier" field at root.
func hasPrettierConfig(root string) bool {
	for _, name := range prettierConfigNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return hasPrettierInPackageJSON(root)
}

// resolveConfigPath returns "" when the consumer has its own config, else the
// shared prettier.config.cjs from the installed @mayjournal/fitness-shared
// package ("" again when that is not installed either).
func resolveConfigPath(root string) string {
	if hasPrettierConfig(root) {
		return ""
	}
	return walkUp(root, func(dir string) string {
		p := filepath.Join(dir, "node_modules", "@mayjournal", "fitness-shared", "config", prettierConfigCjs)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	})
}

// isUnparseableTree is true for staged paths in trees Prettier cannot parse
// (githooks/, scripts/, go/ — Go sources and go.mod have no parser).
func isUnparseableTree(p string) bool {
	return strings.Contains(p, "githooks/") || strings.HasPrefix(p, "scripts/") || strings.HasPrefix(p, "go/")
}

// pathsToCheck returns the staged paths to hand Prettier — existing files
// minus the skip list, .mdc files, and unparseable trees — or ["."] when
// nothing is staged.
func pathsToCheck(root string, staged []string) []string {
	if len(staged) == 0 {
		return []string{"."}
	}
	var out []string
	for _, p := range staged {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			continue
		}
		if prettierSkipStaged[p] || strings.HasSuffix(p, ".mdc") || isUnparseableTree(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// prettierArgv builds Prettier's argv: --config when a shared config applies,
// then either the passthrough args verbatim (no --check) or --check plus the
// paths ("." when the filtered list is empty) — the TS buildPrettierCmd order.
func prettierArgv(configPath string, passthrough, paths []string) []string {
	var argv []string
	if configPath != "" {
		argv = append(argv, "--config", configPath)
	}
	if len(passthrough) > 0 {
		return append(argv, passthrough...)
	}
	argv = append(argv, "--check")
	if len(paths) == 0 {
		return append(argv, ".")
	}
	return append(argv, paths...)
}

// execPrettier runs the resolved binary in root; returns the exit code and
// one combined stdout+stderr buffer, mirroring the TS execSyncResult (which
// redirected with `2>&1`). A failure carrying no exit code maps to 1.
func execPrettier(bin, root string, argv []string) (exitCode int, output string) {
	cmd := exec.Command(bin, argv...)
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

// parsePrettierOutput extracts file paths from [warn] lines, skipping the
// "Code style issues" summary line.
func parsePrettierOutput(output string) []string {
	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, warnPrefix) && !strings.Contains(line, codeStyleSummary) {
			files = append(files, strings.TrimSpace(line[len(warnPrefix):]))
		}
	}
	return files
}

// filesChecked mirrors the TS getFilesChecked: error count when files
// failed, 0 for the "." glob, else the staged-path count.
func filesChecked(parsed, paths []string) int {
	if len(parsed) > 0 {
		return len(parsed)
	}
	if len(paths) == 1 && paths[0] == "." {
		return 0
	}
	return len(paths)
}

// buildExecResult is the TS buildExecCheckResult: ok only on exit 0 with no
// parsed errors; a failure that parsed nothing gets the fallback message.
func buildExecResult(exitCode int, parsed []string, files int) checkkit.Result {
	if exitCode == 0 && len(parsed) == 0 {
		return checkkit.Pass(files)
	}
	if len(parsed) == 0 {
		return checkkit.Fail(files, prettierFallbackMessage)
	}
	return checkkit.Fail(files, parsed...)
}
