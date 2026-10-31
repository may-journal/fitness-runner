// Package biome runs Biome for the eslint and prettier checks in a repo that
// chose it in their place (ADR 0006). A repo chooses Biome with a biome.json
// or biome.jsonc at its root and @biomejs/biome in its package.json. The
// eslint check then runs `biome lint` and the prettier check `biome format`,
// each with the GitHub reporter, whose one-line findings this package reads.
//
//fitness:container cli
//fitness:component kit Its language's checks share it.
package biome

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
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
	return hasConfig(root) && declared(root)
}

func hasConfig(root string) bool {
	for _, name := range configNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

// declared reports whether package.json names @biomejs/biome in its
// dependencies or devDependencies.
func declared(root string) bool {
	raw, err := os.ReadFile(filepath.Join(root, node.Manifest))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]any `json:"dependencies"`
		DevDependencies map[string]any `json:"devDependencies"`
	}
	_ = json.Unmarshal(raw, &pkg)
	_, dev := pkg.DevDependencies[packageName]
	_, dep := pkg.Dependencies[packageName]
	return dev || dep
}

// tooOld is the failure for a Biome before 1.8, whose CLI rejects
// `--reporter=github`, naming the version found.
const tooOld = "Biome %s found; Biome 1.8 or later is required: npm install --save-dev @biomejs/biome@latest"

// version is the resolved binary's answer to `biome --version`, asked
// because package.json may say `latest`, `catalog:`, or an alias. Known is
// false when the answer names no major.minor.
type version struct {
	text         string
	major, minor int
	known        bool
}

func installed(root, bin string) version {
	cmd := exec.Command(bin, "--version")
	cmd.Dir = root
	out, _ := cmd.Output()
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "Version:"))
	parts := strings.SplitN(text, ".", 3)
	if len(parts) < 2 {
		return version{text: text}
	}
	major, errMajor := strconv.Atoi(parts[0])
	minor, errMinor := strconv.Atoi(parts[1])
	return version{text: text, major: major, minor: minor, known: errMajor == nil && errMinor == nil}
}

// before reports whether v is a known version older than major.minor.
func (v version) before(major, minor int) bool {
	return v.known && (v.major < major || v.major == major && v.minor < minor)
}

// maxDiagnostics lifts Biome's default cap of 20 diagnostics, which would
// hide the rest. Biome 1.9 and later take `none`; 1.8 rejects it and takes
// at most 65535.
func (v version) maxDiagnostics() string {
	if v.before(1, 9) {
		return "--max-diagnostics=65535"
	}
	return "--max-diagnostics=none"
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

// Check runs `biome <command>` with flags on paths in root and turns its
// findings into the check's result: each finding rendered by show fails. A
// failure that names no finding fails with fallback and Biome's last lines. A
// missing binary fails with the install hint, and a Biome before 1.8 with its
// version.
func Check(root, command string, flags, paths []string, fallback string, show func(Finding) string) checkkit.Result {
	bin, capFlag, failure := usable(root)
	if failure != "" {
		return checkkit.Fail(0, failure)
	}
	findings, exitCode, tail := run(root, bin, capFlag, command, append(append([]string{}, flags...), paths...))
	errs := make([]string, 0, len(findings))
	for _, f := range findings {
		errs = append(errs, show(f))
	}
	if len(errs) == 0 && exitCode != 0 {
		errs = append([]string{fallback}, tail...)
	}
	return checkkit.Result{Ok: len(errs) == 0, Errors: errs, FilesChecked: len(paths)}
}

// usable resolves the biome binary and the diagnostics cap its version takes,
// or the failure when none resolves or it is older than 1.8.
func usable(root string) (bin, capFlag, failure string) {
	bin, found := node.FindTool(root, "biome")
	if !found {
		return "", "", NotInstalled
	}
	v := installed(root, bin)
	if v.before(1, 8) {
		return "", "", fmt.Sprintf(tooOld, v.text)
	}
	return bin, v.maxDiagnostics(), ""
}

// run execs bin in root as `biome <command>` with the GitHub reporter and
// capFlag, which lifts the diagnostics cap for its version,
// tolerating paths Biome does not handle, then args. It returns every error
// and warning finding, the exit code, and the last lines of output.
func run(root, bin, capFlag, command string, args []string) (findings []Finding, exitCode int, tail []string) {
	argv := append([]string{command, "--reporter=github", capFlag, "--no-errors-on-unmatched", "--files-ignore-unknown=true"}, args...)
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
