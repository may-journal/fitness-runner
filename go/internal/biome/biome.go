// Package biome runs Biome for the eslint and prettier checks in a repo that
// chose it in their place (ADR 0006). A repo chooses Biome with a biome.json
// or biome.jsonc at its root and @biomejs/biome in its package.json. The
// eslint check then runs `biome lint` and the prettier check `biome format`,
// each with the GitHub reporter, whose one-line findings this package reads.
package biome

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// NotInstalled is the failure a check reports when the repo chose Biome but
// no biome binary resolves.
const NotInstalled = "Biome not installed: npm install --save-dev @biomejs/biome"

// packageName is the npm package that ships the biome binary.
const packageName = "@biomejs/biome"

// configNames are the root config files that mark a repo as using Biome.
var configNames = []string{"biome.json", "biome.jsonc"}

// Chosen reports whether the repo at root lints and formats with Biome: a
// Biome config at the root and @biomejs/biome among its package.json
// dependencies or devDependencies.
func Chosen(root string) bool {
	_, ok := declared(root)
	return hasConfig(root) && ok
}

func hasConfig(root string) bool {
	for _, name := range configNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

// declared returns the version range package.json gives @biomejs/biome, in
// devDependencies or dependencies, and whether it names the package at all.
func declared(root string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return "", false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	_ = json.Unmarshal(raw, &pkg)
	if v, ok := pkg.DevDependencies[packageName]; ok {
		return v, true
	}
	v, ok := pkg.Dependencies[packageName]
	return v, ok
}

// maxDiagnostics lifts Biome's default cap of 20 diagnostics, which would
// hide the rest. Biome 2 and 1.9 take `none`; earlier 1.x rejects it and
// takes at most 65535, which every 1.x accepts.
func maxDiagnostics(root string) string {
	version, _ := declared(root)
	major, _, _ := strings.Cut(strings.TrimLeft(version, "^~>=v "), ".")
	if major == "1" {
		return "--max-diagnostics=65535"
	}
	return "--max-diagnostics=none"
}

// Bin resolves the biome binary: node_modules/.bin walking up from root,
// then PATH, never npx. Empty when none resolves.
func Bin(root string) string {
	for dir := absolute(root); ; {
		if p := filepath.Join(dir, "node_modules", ".bin", "biome"); executable(p) {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	p, err := exec.LookPath("biome")
	if err != nil {
		return ""
	}
	return p
}

func absolute(root string) string {
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func executable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// Finding is one diagnostic from Biome's GitHub reporter: the rule (or
// "format"), the file, its position, and the message.
type Finding struct {
	Rule    string
	File    string
	Line    int
	Col     int
	Message string
}

// Check runs `biome <command>` on paths in root and turns its findings into
// the check's result: each finding rendered by show fails. A failure that
// names no finding fails with fallback and Biome's last lines. A missing
// binary fails with the install hint.
func Check(root, command string, paths []string, fallback string, show func(Finding) string) checkkit.Result {
	bin := Bin(root)
	if bin == "" {
		return checkkit.Fail(0, NotInstalled)
	}
	findings, exitCode, tail := run(root, bin, command, paths)
	errs := make([]string, 0, len(findings))
	for _, f := range findings {
		errs = append(errs, show(f))
	}
	if len(errs) == 0 && exitCode != 0 {
		errs = append([]string{fallback}, tail...)
	}
	return checkkit.Result{Ok: len(errs) == 0, Errors: errs, FilesChecked: len(paths)}
}

// run execs bin in root as `biome <command>` with the GitHub reporter and
// the diagnostics cap lifted for the declared major (maxDiagnostics),
// tolerating paths Biome does not handle, then args. It returns every error
// and warning finding, the exit code, and the last lines of output.
func run(root, bin, command string, args []string) (findings []Finding, exitCode int, tail []string) {
	argv := append([]string{command, "--reporter=github", maxDiagnostics(root), "--no-errors-on-unmatched", "--files-ignore-unknown=true"}, args...)
	cmd := exec.Command(bin, argv...)
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		exitCode = 1
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			exitCode = exit.ExitCode()
		}
	}
	return parse(out.String()), exitCode, lastLines(out.String(), 5)
}

// parse reads the GitHub reporter's `::error` and `::warning` lines:
// `::error title=<rule>,file=<path>,line=<n>,...,col=<n>,...::<message>`.
func parse(output string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(output, "\n") {
		if f, ok := parseLine(strings.TrimSpace(line)); ok {
			findings = append(findings, f)
		}
	}
	return findings
}

func parseLine(line string) (Finding, bool) {
	rest, ok := cutLevel(line)
	if !ok {
		return Finding{}, false
	}
	props, message, ok := strings.Cut(rest, "::")
	if !ok {
		return Finding{}, false
	}
	kv := map[string]string{}
	for _, pair := range strings.Split(props, ",") {
		key, value, _ := strings.Cut(pair, "=")
		kv[key] = value
	}
	n, _ := strconv.Atoi(kv["line"])
	col, _ := strconv.Atoi(kv["col"])
	return Finding{Rule: kv["title"], File: kv["file"], Line: n, Col: col, Message: message}, kv["file"] != ""
}

// cutLevel strips the `::error ` or `::warning ` prefix from a line.
func cutLevel(line string) (string, bool) {
	for _, prefix := range []string{"::error ", "::warning "} {
		if rest, ok := strings.CutPrefix(line, prefix); ok {
			return rest, true
		}
	}
	return "", false
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
