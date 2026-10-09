// Command fitness-check-prettier runs the real Prettier over the repo — the
// Go port of the prettier check. The binary stays zero-dependency: Prettier
// is resolved at runtime as a peer of the repo under check (node_modules/.bin
// walking up from root, then PATH — never npx), and a missing binary fails
// with a one-line install hint. Prettier checks every tracked file (or every
// changed file in a scoped run) with --ignore-unknown, so a file type it has
// no parser for passes instead of being skipped by name. It ignores nothing
// else: --ignore-path points at an empty file and --with-node-modules is on,
// and a repo with a .prettierignore fails. Passthrough args replace --check
// mode, [warn] lines become per-file errors, and a non-zero exit that parsed
// nothing reports the fallback message. When the repo has no Prettier config of its own, the
// shared prettier.config.cjs is passed via --config: an installed
// @mayjournal/fitness-shared wins (the npm-era node_modules walk, mirroring
// the TS resolveFitnessConfigPath), else the copy embedded in this binary is
// materialized to the cache (internal/sharedconf). The shared config resolves
// its two plugins with createRequire from its own location, which never has
// them, so the check puts the repo's node_modules folders on NODE_PATH.
// Installing the plugins stays the check's peer contract, unchanged from the
// npm era. When Prettier fails without parsing a file, its own [error] lines
// are reported, so a missing plugin names itself.
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
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
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

// prettierIgnoreFile is the ignore file Prettier would honor; the check
// fails a repo that has one.
const prettierIgnoreFile = ".prettierignore"

// ignoreNothing is Prettier's --ignore-path: an empty file, so neither
// .gitignore nor .prettierignore hides a tracked file.
const ignoreNothing = "/dev/null"

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "prettier"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	bin, res, done := preflight(root)
	if done {
		return res, nil
	}
	configPath := resolveConfigPath(root)
	paths, ok := scopedPaths(root, args)
	if !ok {
		return checkkit.Pass(0), nil
	}
	env := os.Environ()
	if configPath != "" {
		env = withNodePath(env, root)
	}
	exitCode, output := execPrettier(bin, root, env, prettierArgv(configPath, args, paths))
	parsed := parsePrettierOutput(output)
	return buildExecResult(exitCode, parsed, prettierErrors(output), filesChecked(parsed, paths)), nil
}

// preflight resolves the Prettier binary, or the verdict that ends the run
// first: a failure for any repo with a .prettierignore, then a pass for a
// repo with no package.json, then a failure without Prettier installed.
func preflight(root string) (bin string, res checkkit.Result, done bool) {
	if _, err := os.Stat(filepath.Join(root, prettierIgnoreFile)); err == nil {
		return "", checkkit.Fail(0, prettierIgnoreFile+" exists; Prettier checks every tracked file, so delete it and fix the findings instead"), true
	}
	if len(walkfs.FilesByExt(root, packageJSONName)) == 0 {
		return "", checkkit.Pass(0), true
	}
	if bin = resolvePrettierBin(root); bin == "" {
		return "", checkkit.Fail(0, notInstalled), true
	}
	return bin, checkkit.Result{}, false
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

// nodeModulesDirs lists every node_modules folder from root up to the
// filesystem root, nearest first, the same walk that finds the binary.
func nodeModulesDirs(root string) []string {
	var dirs []string
	walkUp(root, func(dir string) string {
		p := filepath.Join(dir, "node_modules")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			dirs = append(dirs, p)
		}
		return ""
	})
	return dirs
}

// withNodePath returns env with root's node_modules folders prepended to
// NODE_PATH, so the shared config, materialized outside the repo, can load
// the plugins the repo installs.
func withNodePath(env []string, root string) []string {
	dirs := nodeModulesDirs(root)
	if len(dirs) == 0 {
		return env
	}
	out := make([]string, 0, len(env)+1)
	existing := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "NODE_PATH="); ok {
			existing = v
			continue
		}
		out = append(out, kv)
	}
	if existing != "" {
		dirs = append(dirs, existing)
	}
	return append(out, "NODE_PATH="+strings.Join(dirs, string(os.PathListSeparator)))
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
// shared prettier.config.cjs — an installed @mayjournal/fitness-shared when
// present, the embedded copy materialized from this binary otherwise
// (sharedconf.Resolve; its repo-local step never fires here because
// hasPrettierConfig already ruled a root-level prettier.config.cjs out). ""
// again only when even the fallback fails to materialize, running Prettier on
// its defaults.
func resolveConfigPath(root string) string {
	if hasPrettierConfig(root) {
		return ""
	}
	return sharedconf.Resolve(root, prettierConfigCjs)
}

// scopedPaths returns the paths to hand Prettier, and false when a scoped run
// changed no file still on disk, so it skips instead of checking the whole
// repo. Passthrough args always run.
func scopedPaths(root string, args []string) ([]string, bool) {
	changed := checkkit.ChangedFiles()
	paths := pathsToCheck(root, changed)
	return paths, changed == nil || len(paths) > 0 || len(args) > 0
}

// pathsToCheck returns the changed paths still on disk, or every tracked
// file when the run is unscoped.
func pathsToCheck(root string, changed []string) []string {
	if len(changed) == 0 {
		return walkfs.FilesByExt(root, "")
	}
	var out []string
	for _, p := range changed {
		if info, err := os.Stat(filepath.Join(root, p)); err == nil && !info.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

// prettierArgv builds Prettier's argv: --config when a shared config applies,
// the flags that make Prettier ignore nothing it can parse, then either the
// passthrough args verbatim (no --check) or --check plus the paths.
func prettierArgv(configPath string, passthrough, paths []string) []string {
	var argv []string
	if configPath != "" {
		argv = append(argv, "--config", configPath)
	}
	argv = append(argv, "--ignore-unknown", "--ignore-path", ignoreNothing, "--with-node-modules")
	if len(passthrough) > 0 {
		return append(argv, passthrough...)
	}
	return append(append(argv, "--check"), paths...)
}

// execPrettier runs the resolved binary in root; returns the exit code and
// one combined stdout+stderr buffer, mirroring the TS execSyncResult (which
// redirected with `2>&1`). A failure carrying no exit code maps to 1.
func execPrettier(bin, root string, env, argv []string) (exitCode int, output string) {
	cmd := exec.Command(bin, argv...)
	cmd.Dir = root
	cmd.Env = env
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

// maxPrettierErrors caps the [error] lines reported when Prettier fails
// without parsing a file; the first few name the cause.
const maxPrettierErrors = 5

// prettierErrors returns Prettier's own [error] lines, up to
// maxPrettierErrors, such as a plugin the shared config cannot load. When it
// printed none, its last lines stand in, so a failure always shows its cause.
func prettierErrors(output string) []string {
	var errs []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[error] ") && len(errs) < maxPrettierErrors {
			errs = append(errs, line)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return lastLines(output, maxPrettierErrors)
}

// lastLines returns the last n non-blank lines of output, trimmed.
func lastLines(output string, n int) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines[max(0, len(lines)-n):]
}

// filesChecked mirrors the TS getFilesChecked: error count when files
// failed, else the path count.
func filesChecked(parsed, paths []string) int {
	if len(parsed) > 0 {
		return len(parsed)
	}
	return len(paths)
}

// buildExecResult is the TS buildExecCheckResult: ok only on exit 0 with no
// parsed errors; a failure that parsed nothing gets the fallback message,
// followed by Prettier's own [error] lines when it printed any.
func buildExecResult(exitCode int, parsed, prettierErrs []string, files int) checkkit.Result {
	if exitCode == 0 && len(parsed) == 0 {
		return checkkit.Pass(files)
	}
	if len(parsed) == 0 {
		return checkkit.Fail(files, append([]string{prettierFallbackMessage}, prettierErrs...)...)
	}
	return checkkit.Fail(files, parsed...)
}
