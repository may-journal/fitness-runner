package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("build-fitness", "Build every fitness command from the checkout into go/bin, so this repo's CI runs the code under test.", nil, buildFitness)
	register("export-checks-dir", "Install the checks the repo pins through githooks/checks-dir and point later steps at them with FITNESS_CHECKS_DIR.", nil, exportChecksDir)
	register("show-performance", "Print each check's time, slowest first, from the performance record the suite kept, when it kept one.", nil, showPerformance)
	register("check-secrets", "Fail with the secret's name when the automation App's APP_ID or APP_KEY did not reach this repo.", nil, checkSecrets)
	register("run-commands", "Run each fitness command in FITNESS_COMMANDS after checking every name, so a typo runs nothing and one run lists every failure.", nil, runCommands)
	register("install-source-checks", "Install the pinned checks beside a source-built engine and point later steps at them with FITNESS_CHECKS_DIR.", nil, installSourceChecks)
	register("cache-key", "Write the Go build cache's prefix, key and save flag for FAMILY, GO_VERSION, SOURCE_HASH and SAVE.", []string{"prefix", "key", "save"}, cacheKey)
}

// performanceRecord is the record the suite keeps for this job, in name.
func performanceRecord(name string) string {
	return filepath.Join(env("RUNNER_TEMP"), "fitness-performance", name)
}

func buildFitness(actions.Step, []string) error {
	return actions.Run("go", "-C", "go", "build", "-o", "bin/", "./cmd/...")
}

func exportChecksDir(actions.Step, []string) error {
	checks, err := actions.Call{}.Capture("githooks/checks-dir")
	if err != nil {
		return err
	}
	return actions.Env("FITNESS_CHECKS_DIR", checks)
}

func showPerformance(actions.Step, []string) error {
	// A suite that never wrote a record already failed, so this stays quiet.
	record := performanceRecord("fitness.json")
	if !exists(record) {
		return nil
	}
	return actions.Run("go/bin/fitness", "perf", record)
}

func checkSecrets(actions.Step, []string) error {
	for _, name := range []string{"APP_ID", "APP_KEY"} {
		if env(name) == "" {
			return actions.Fail("AUTOMATION_" + name + " is not available to this repo")
		}
	}
	return nil
}

// commandName is the shape of every fitness command's name.
var commandName = regexp.MustCompile(`^[a-z-]+$`)

func runCommands(actions.Step, []string) error {
	commands := strings.Fields(env("FITNESS_COMMANDS"))
	if why := badCommands(commands); why != "" {
		return refuse(why)
	}
	// A later command still runs after an earlier one fails, so one run
	// lists every failure.
	failed := false
	for _, command := range commands {
		failed = actions.Run("fitness", command) != nil || failed
	}
	if failed {
		return actions.Failed
	}
	return nil
}

// badCommands says what is wrong with the command list, or "".
func badCommands(commands []string) string {
	if len(commands) == 0 {
		return "needs at least one command"
	}
	if at := slices.IndexFunc(commands, func(c string) bool { return !commandName.MatchString(c) }); at >= 0 {
		return "got " + commands[at] + ", which is not a fitness command name"
	}
	return ""
}

// refuse annotates the bad input once, marks the failure reported, and
// stops before any command runs.
func refuse(why string) error {
	actions.Error("The fitness-command action " + why + ".")
	if err := actions.Env("FITNESS_FAILURE_REPORTED", "true"); err != nil {
		return err
	}
	return actions.Failed
}

func installSourceChecks(actions.Step, []string) error {
	gopath, err := actions.Call{}.Capture("go", "env", "GOPATH")
	if err != nil {
		return err
	}
	printed, err := actions.Call{}.Capture(filepath.Join(gopath, "bin", "fitness-install"), "--version", "latest", "--install-only")
	if err != nil {
		return err
	}
	// The second line names the checks folder; a repo that pins none prints none.
	lines := strings.Split(printed, "\n")
	if len(lines) < 2 || lines[1] == "" {
		return nil
	}
	return actions.Env("FITNESS_CHECKS_DIR", lines[1])
}

func cacheKey(step actions.Step, _ []string) error {
	prefix := fmt.Sprintf("go-build-%s-%s-%s-go%s-", env("FAMILY"), env("RUNNER_OS"), env("RUNNER_ARCH"), env("GO_VERSION"))
	return writeAll(step, "prefix", prefix, "key", prefix+env("SOURCE_HASH"), "save", env("SAVE"))
}
