package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// checksCommands are the subcommands that run, describe or judge GitHub
// bodies with checks, so they fetch the missing ones. A run, a flag or a
// check name fetches too.
var checksCommands = []string{
	"check", "hook", "list", "explain", "branch-check", "pr-check", "dependabot-pr",
	"issue-check", "plan-check", "close-check", "dupe-check", "comment-check",
}

// fetchMissingChecks installs the checks the repo pins, or every released
// check at v1 when it pins none, once any of them is missing. The engine holds
// no checks, so a plain `fitness` after `go install` judges as an install does.
func fetchMissingChecks(args []string) {
	if !needsChecks(args) {
		return
	}
	pins, err := checkVersions(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: could not read the checks the repo pins: %v; install them with `fitness-install --with-checks`\n", err)
		return
	}
	missing := hookPins(args, missingPins(pins))
	if !fetchesLater(args, missing) {
		installMissing(missing)
	}
}

// fetchesLater reports whether nothing installs now. Either none is missing,
// or the command fetches each check once it reaches it, as defersFetch says.
func fetchesLater(args []string, missing map[string]string) bool {
	return len(missing) == 0 || defersFetch(args) && planFetches(missing)
}

// installMissing installs every missing pin and hands the checks to the run.
func installMissing(missing map[string]string) {
	dir, err := installPins(missing)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: could not install the checks the repo pins: %v; install them with `fitness-install --with-checks`\n", err)
		return
	}
	addFetched(dir)
}

// addFetched adds each program in dir that no installed program provides,
// and hands dir to the checks this run starts, such as local-branch.
func addFetched(dir string) {
	_ = os.Setenv(checkkit.ChecksDirEnv, dir)
	if installedPrograms == nil {
		installedPrograms = map[string]string{}
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		name, ok := strings.CutPrefix(entry.Name(), checkkit.BinPrefix)
		if ok && installedPrograms[name] == "" {
			installedPrograms[name] = filepath.Join(dir, entry.Name())
		}
	}
}

// needsChecks reports whether a run with args runs checks and has none handed
// to it, as fitness-install hands them through FITNESS_CHECKS_DIR.
func needsChecks(args []string) bool {
	// --no-builtins tests a clean install, so it judges only what is installed.
	if os.Getenv(checkkit.ChecksDirEnv) != "" || builtinsOff() || listsBuiltins(args) {
		return false
	}
	return len(args) == 0 || runsChecks(args[0])
}

// listsBuiltins reports whether args are `check --list`, which names only
// the checks the engine runs itself. An install runs it on every new
// bundle, so it fetches nothing.
func listsBuiltins(args []string) bool {
	return len(args) > 1 && args[0] == "check" && args[1] == "--list"
}

// runsChecks reports whether a command line starting with first runs or
// describes checks: a flag, a check name or one of checksCommands.
func runsChecks(first string) bool {
	_, subcommand := subcommands[first]
	return strings.HasPrefix(first, "-") || !subcommand || slices.Contains(checksCommands, first)
}

// missingPins keeps the pins whose check is not installed.
func missingPins(pins map[string]string) map[string]string {
	missing := map[string]string{}
	for name, pin := range pins {
		if installedPrograms[name] == "" {
			missing[name] = pin
		}
	}
	return missing
}

// installPins installs each pinned check, verified by checksum, into the
// user's cache, and returns the folder that holds them.
func installPins(pins map[string]string) (string, error) {
	installer, err := distribution.New("", "", os.Stderr)
	if err != nil {
		return "", err
	}
	return installer.InstallChecks(context.Background(), pinList(pins))
}
