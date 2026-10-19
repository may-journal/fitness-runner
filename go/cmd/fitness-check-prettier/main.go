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
//
// A repo that chose Biome (internal/biome) is checked by `biome format` on
// the same paths instead, and needs no Prettier (ADR 0006). Its passthrough
// args are translated to Biome's, never handed over as paths.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/biome"
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
	bin, res, done := preflight(root, args)
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
	output = ansiCodes.ReplaceAllString(output, "")
	parsed := parsePrettierOutput(output)
	return buildExecResult(exitCode, parsed, prettierErrors(output), filesChecked(parsed, paths)), nil
}

// preflight resolves the Prettier binary, or the verdict that ends the run
// first: a failure for any repo with a .prettierignore, then a pass for a
// repo with no package.json, then Biome's verdict in a repo that chose it,
// then a failure without Prettier installed.
func preflight(root string, args []string) (bin string, res checkkit.Result, done bool) {
	if _, err := os.Stat(filepath.Join(root, prettierIgnoreFile)); err == nil {
		return "", checkkit.Fail(0, prettierIgnoreFile+" exists; Prettier checks every tracked file, so delete it and fix the findings instead"), true
	}
	if !node.Applies(root) {
		return "", checkkit.Pass(0), true
	}
	if biome.Chosen(root) {
		return "", formatWithBiome(root, args), true
	}
	bin, found := node.FindTool(root, "prettier")
	if !found {
		return "", checkkit.Fail(0, notInstalled), true
	}
	return bin, checkkit.Result{}, false
}

// formatWithBiome checks the same paths with `biome format` in a repo that
// chose Biome (ADR 0006), naming each file Biome would reformat. Passthrough
// args are Prettier's, so biomeFormatArgs translates them: the paths among
// them replace the checked paths, and a flag Biome has no counterpart for
// fails by name instead of reaching Biome as a path.
func formatWithBiome(root string, args []string) checkkit.Result {
	flags, given, unknown := biomeFormatArgs(args)
	if len(unknown) > 0 {
		return checkkit.Fail(0, "Biome formats this repo and has no counterpart for Prettier's "+strings.Join(unknown, ", ")+"; pass paths, --write, --check, or --ignore-unknown")
	}
	paths, ok := scopedPaths(root, args)
	if !ok {
		return checkkit.Pass(0)
	}
	if len(given) > 0 {
		paths = given
	}
	return biome.Check(root, "format", flags, paths, "Biome reported issues. Run: npx biome format --write .", func(f biome.Finding) string {
		return f.File
	})
}

// biomeFormatArgs splits Prettier passthrough args into `biome format` flags,
// paths, and the flags Biome has no counterpart for. Everything after "--",
// and every arg not starting with "-", is a path.
func biomeFormatArgs(args []string) (flags, paths, unknown []string) {
	if i := slices.Index(args, "--"); i >= 0 {
		flags, paths, unknown = biomeFormatArgs(args[:i])
		return flags, append(paths, args[i+1:]...), unknown
	}
	for _, a := range args {
		if biomeFlags, ok := prettierToBiome[a]; ok {
			flags = append(flags, biomeFlags...)
		} else if strings.HasPrefix(a, "-") {
			unknown = append(unknown, a)
		} else {
			paths = append(paths, a)
		}
	}
	return flags, paths, unknown
}

// prettierToBiome maps the Prettier flags Biome can honor to Biome's own.
// --write writes in place. --check and --list-different are Biome's default,
// and --ignore-unknown is always on, so they map to nothing.
var prettierToBiome = map[string][]string{
	"--write": {"--write"}, "-w": {"--write"},
	"--check": nil, "-c": nil,
	"--list-different": nil, "-l": nil,
	"--ignore-unknown": nil, "-u": nil,
}

// nodeModulesDirs lists every node_modules folder from root up to the
// filesystem root, nearest first, the same walk that finds the binary.
func nodeModulesDirs(root string) []string {
	var dirs []string
	for _, dir := range node.Ancestors(root) {
		p := filepath.Join(dir, "node_modules")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			dirs = append(dirs, p)
		}
	}
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
// its defaults. The path has its symlinks resolved, as Node resolves the
// config's own __dirname, so the config's override globs, which Prettier
// matches from the --config folder, count the same ../ steps.
func resolveConfigPath(root string) string {
	if hasPrettierConfig(root) {
		return ""
	}
	p := sharedconf.Resolve(root, prettierConfigCjs)
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
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
	o := checkkit.ExecCombined(cmd)
	return o.Code, o.Stdout
}

// ansiCodes matches terminal color codes. Prettier colors its output in CI,
// which would hide the [warn] and [error] prefixes the check reads.
var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
