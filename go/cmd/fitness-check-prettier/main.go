package check

import (
	"encoding/json"
	"errors"
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
	notInstalled = "Prettier not installed: npm install --save-dev prettier"

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

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "prettier", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	bin, res, done := preflight(root, args)
	if done {
		return res, nil
	}
	if bin == "" {
		return checkkit.Result{}, checkkit.Internal(errors.New(notInstalled))
	}
	configPath := resolveConfigPath(root)
	paths, ok := scopedPaths(root, args)
	if !ok {
		return checkkit.Pass(0), nil
	}
	o := execPrettier(bin, root, prettierEnv(root, configPath), prettierArgv(configPath, args, paths))
	if err := o.EnvErr("prettier"); err != nil {
		return checkkit.Result{}, err
	}
	output := ansiCodes.ReplaceAllString(o.Stdout, "")
	parsed := append(parsePrettierOutput(output), unparsable(root, output)...)
	return buildExecResult(o.Code, parsed, prettierErrors(output), filesChecked(parsed, paths))
}

// preflight resolves the Prettier binary, or an early verdict for a
// .prettierignore, a missing package.json, or Biome.
//
// Without Prettier installed it returns no binary and no verdict.
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
	bin, _ = node.FindTool(root, "prettier")
	return bin, checkkit.Result{}, false
}

// formatWithBiome runs `biome format` in a repo that chose Biome (ADR 0006),
// translating Prettier args and naming any flag Biome lacks.
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

// prettierEnv is the environment Prettier runs in: this process's, with
// root's node_modules folders on NODE_PATH when the shared config applies.
func prettierEnv(root, configPath string) []string {
	if configPath == "" {
		return os.Environ()
	}
	return withNodePath(os.Environ(), root)
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

// withNodePath prepends root's node_modules folders to NODE_PATH, so the
// shared config outside the repo loads its plugins.
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

// resolveConfigPath returns "" for a consumer's own config, else the shared
// one with symlinks resolved so its globs count the same ../ steps.
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

// scopedPaths returns the paths to hand Prettier, and false to skip a scoped
// run with no args and no changed file on disk.
func scopedPaths(root string, args []string) ([]string, bool) {
	changed := checkkit.ChangedFiles()
	paths := pathsToCheck(root, changed)
	return paths, changed == nil || len(paths) > 0 || len(args) > 0
}

// pathsToCheck returns every tracked file when unscoped, else the changed
// regular files, since Prettier refuses a named symlink.
func pathsToCheck(root string, changed []string) []string {
	if len(changed) == 0 {
		return walkfs.FilesByExt(root, "")
	}
	var out []string
	for _, p := range changed {
		if walkfs.RegularFile(root, p) {
			out = append(out, p)
		}
	}
	return out
}

// prettierArgv builds Prettier's argv to ignore nothing it can parse, ending
// with passthrough args or --check and the paths.
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

// execPrettier runs the binary in root with stdout and stderr in one buffer,
// as the TS check's `2>&1` did.
func execPrettier(bin, root string, env, argv []string) checkkit.Outcome {
	cmd := exec.Command(bin, argv...)
	cmd.Dir = root
	cmd.Env = env
	return checkkit.ExecCombined(cmd)
}

// errorPrefix opens each error line Prettier prints.
const errorPrefix = "[error] "

// unparsable returns Prettier's error lines that name a file under root,
// such as `[error] src/a.ts: SyntaxError: …`: a file Prettier could not
// parse is a finding. An error line naming no file, such as a plugin the
// config cannot load, is not.
func unparsable(root, output string) []string {
	var errs []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, errorPrefix)
		if !ok {
			continue
		}
		if namesFile(root, rest) {
			errs = append(errs, line)
		}
	}
	return errs
}

// namesFile reports whether an error line's text opens with a path to a
// file under root, then ": ".
func namesFile(root, text string) bool {
	file, _, found := strings.Cut(text, ": ")
	if !found {
		return false
	}
	info, err := os.Stat(filepath.Join(root, file))
	return err == nil && !info.IsDir()
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
		if strings.HasPrefix(line, errorPrefix) && len(errs) < maxPrettierErrors {
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
// parsed errors, failing with the parsed errors otherwise. A failure that
// parsed nothing is an internal error quoting Prettier's own [error] lines
// or last lines, which name the cause.
func buildExecResult(exitCode int, parsed, prettierErrs []string, files int) (checkkit.Result, error) {
	if exitCode == 0 && len(parsed) == 0 {
		return checkkit.Pass(files), nil
	}
	if len(parsed) == 0 {
		return checkkit.Result{}, checkkit.NoVerdict("prettier", exitCode, strings.Join(prettierErrs, "\n"))
	}
	return checkkit.Fail(files, parsed...), nil
}
