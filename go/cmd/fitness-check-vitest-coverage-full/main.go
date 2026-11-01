package check

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
)

// missingVitestHint is this port's own info. The TS check ran through npx,
// which fetched vitest on demand. The direct exec asks for an install.
const missingVitestHint = "vitest not found in node_modules/.bin (walking up from root) " +
	"or on PATH. Install it (npm install -D vitest) to run the vitest-coverage-full check"

// fitnessVitestConfig is the TS FITNESS_VITEST_CONFIG: the config filename
// resolved from the fitness-runner root when the consumer has none.
const fitnessVitestConfig = "vitest.config.mjs"

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "vitest-coverage-full", Scope: checkkit.ScopeFiles, TimeoutMs: 120000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !node.Applies(root) || !node.HasSource(root) {
		return checkkit.Pass(0), nil
	}
	return runVitest(root)
}

// runVitest applies the coverage rule to a JS project: full thresholds here and
// in the shared config, then a passing vitest run. A missing, crashed or
// killed vitest is an internal error, since it never judged the code.
func runVitest(root string) (checkkit.Result, error) {
	frRoot := sharedconf.ResolveDir(root)
	if !vitestconf.HasFullThresholds(root, frRoot) {
		return checkkit.Fail(1, thresholdsNot100), nil
	}
	if !vitestconf.HasFullThresholds(frRoot, "") {
		return checkkit.Fail(1, fitnessRunnerThresholdsNot100), nil
	}
	bin, found := node.FindTool(root, "vitest")
	if !found {
		return checkkit.Result{}, checkkit.Internal(errors.New(missingVitestHint))
	}
	return judgeRun(execVitest(bin, root, vitestArgs(root, frRoot)))
}

// judgeRun judges a finished vitest run. It passes on exit 0 and reports the
// last line of output when Vitest judged the code.
// It reports an internal error when Vitest could not start, was killed, or
// failed without a verdict.
func judgeRun(o checkkit.Outcome) (checkkit.Result, error) {
	if err := o.EnvErr("vitest"); err != nil {
		return checkkit.Result{}, err
	}
	output := combineStreams(o.Stdout, o.Stderr)
	switch {
	case o.Code == 0:
		return checkkit.Pass(1), nil
	case !verdict.MatchString(output):
		return checkkit.Result{}, checkkit.NoVerdict("vitest run --coverage", o.Code, output)
	}
	return checkkit.Fail(1, failureMessage(output)), nil
}

// verdict matches what Vitest prints when it judged the code: a coverage
// threshold miss, or a failing test or test file. A failure without either,
// such as a missing coverage provider, never judged it.
var verdict = regexp.MustCompile(`does not meet (global )?threshold|\b\d+ failed\b`)

// vitestArgs is the TS buildVitestCoverageCmd. It builds plain `run --coverage`
// when the consumer has a local vitest.config.*. Otherwise it adds --config
// pointing at the fitness-runner fallback.
//
// That fallback is frRoot, the sharedconf-resolved config directory (the TS
// resolveFitnessConfigPath). A local vitest.config.mjs would win, but by then
// no local config exists. The TS shell-quoted the path for execSync.
// Args exec directly here, unquoted.
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

// execVitest runs the vitest binary in root and returns its outcome. stderr
// also passes through live, as Node's execSync default stdio did.
func execVitest(bin, root string, args []string) checkkit.Outcome {
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	return checkkit.Exec(cmd)
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
// the TS coverageFailureResult.
func failureMessage(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
