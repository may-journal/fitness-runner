// Command fitness-check-vitest-coverage-full ensures the consumer repo and
// the fitness-runner package both hold Vitest coverage thresholds of 100,
// then runs `vitest run --coverage` in root — the Go port of the
// vitest-coverage-full check. Two departures from the TypeScript original,
// both mechanical: coverage thresholds are judged textually via
// internal/vitestconf instead of evaluating the config as JavaScript, and
// vitest is executed directly — resolved from node_modules/.bin walking up
// from root, then PATH, never npx — so a missing binary fails with an
// install hint where npx would have installed it on demand. Threshold
// gating order, config fallback, last-line failure extraction, error
// strings, and filesChecked all match the TS check. Node's execSync capped
// captured output at 1 MiB (maxBuffer); this port does not.
//
// The fitness-runner root — the directory whose vitest.config.mjs backs both
// the threshold fallback and the --config fallback — resolves through
// internal/sharedconf: an installed node_modules/@mayjournal/fitness-shared
// walking up from root, else the config directory embedded in this binary,
// materialized to the cache.
package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/vitestconf"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
)

// Error strings ported verbatim from the TS check's enUS.
const (
	thresholdsNot100 = "Vitest coverage thresholds must be 100 for " +
		"branches, functions, lines, and statements."
	fitnessRunnerThresholdsNot100 = "Fitness-runner package must have Vitest coverage " +
		"thresholds set to 100 for branches, functions, lines, and statements."
	fallbackRunHint = "Vitest coverage did not meet 100% thresholds. Run: npm run test"
)

// missingVitestHint is this port's own failure: the TS check ran through
// npx, which fetched vitest on demand; the direct exec asks for an install.
const missingVitestHint = "vitest not found in node_modules/.bin (walking up from root) " +
	"or on PATH. Install it (npm install -D vitest) to run the vitest-coverage-full check"

// fitnessVitestConfig is the TS FITNESS_VITEST_CONFIG: the config filename
// resolved from the fitness-runner root when the consumer has none.
const fitnessVitestConfig = "vitest.config.mjs"

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "vitest-coverage-full", TimeoutMs: 120000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !node.Applies(root) {
		return checkkit.Pass(0), nil
	}
	return runVitest(root), nil
}

// runVitest applies the coverage rule to a JS project: full thresholds here and
// in the shared config, then a passing vitest run.
func runVitest(root string) checkkit.Result {
	frRoot := sharedconf.ResolveDir(root)
	if !vitestconf.HasFullThresholds(root, frRoot) {
		return checkkit.Fail(1, thresholdsNot100)
	}
	if !vitestconf.HasFullThresholds(frRoot, "") {
		return checkkit.Fail(1, fitnessRunnerThresholdsNot100)
	}
	bin, found := node.FindTool(root, "vitest")
	if !found {
		return checkkit.Fail(1, missingVitestHint)
	}
	exitCode, output := execVitest(bin, root, vitestArgs(root, frRoot))
	if exitCode == 0 {
		return checkkit.Pass(1)
	}
	return checkkit.Fail(1, failureMessage(output))
}

// vitestArgs is the TS buildVitestCoverageCmd: plain `run --coverage` when
// the consumer has a local vitest.config.*, else with --config pointing at
// the fitness-runner fallback — frRoot, the sharedconf-resolved config
// directory (the TS resolveFitnessConfigPath: a local vitest.config.mjs
// would win, but by then no local config exists). The TS shell-quoted the
// path for execSync; args exec directly here, unquoted.
func vitestArgs(root, frRoot string) []string {
	args := []string{"run", "--coverage"}
	for _, name := range vitestconf.ConfigNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return args
		}
	}
	configPath := filepath.Join(root, fitnessVitestConfig)
	if _, err := os.Stat(configPath); err != nil {
		configPath = filepath.Join(frRoot, fitnessVitestConfig)
	}
	return append(args, "--config", configPath)
}

// execVitest runs the vitest binary in root, mirroring the TS
// execSyncResult: exit 0 returns stdout alone; a failure returns the exit
// code with stdout and stderr joined by a newline, empty streams dropped.
// stderr also passes through live, as Node's execSync default stdio did.
func execVitest(bin, root string, args []string) (int, string) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	o := checkkit.Exec(cmd)
	if o.Err != nil {
		return o.Code, combineStreams(o.Stdout, o.Stderr)
	}
	return 0, o.Stdout
}

// combineStreams is the TS `[stdout, stderr].filter(Boolean).join('\n')`.
func combineStreams(stdout, stderr string) string {
	var parts []string
	for _, s := range []string{stdout, stderr} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}

// failureMessage extracts the last line of the combined output, exactly like
// the TS coverageFailureResult; an all-whitespace output falls back to the
// run hint.
func failureMessage(output string) string {
	trimmed := strings.TrimSpace(output)
	lines := strings.Split(trimmed, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return fallbackRunHint
	}
	return last
}
