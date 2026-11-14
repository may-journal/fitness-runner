package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// currentMajor is the major every check releases on today, which an unset
// checkVersions pins each check to.
const currentMajor = "v1"

// runCheckVersions prints each check the repo installs with its pin, one
// `<name> <pin>` line each in name order, for the installer to resolve.
func runCheckVersions(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "fitness check-versions: takes no arguments")
		return 2
	}
	pins, err := checkVersions(".")
	if err != nil {
		report.Error("fitness check-versions", err)
		return 1
	}
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		fmt.Println(name, pins[name])
	}
	return 0
}

// checkVersions is root's checkVersions, or every known check at
// currentMajor when the key is unset. A pin naming no known check fails.
func checkVersions(root string) (map[string]string, error) {
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.CheckVersions == nil {
		return defaultPins(), nil
	}
	return cfg.CheckVersions, unknownPin(cfg.CheckVersions)
}

// defaultPins pins every released check to currentMajor.
func defaultPins() map[string]string {
	pins := map[string]string{}
	for _, name := range releasedChecks() {
		pins[name] = currentMajor
	}
	return pins
}

// unknownPin fails the first pin, in name order, that names no released
// check.
func unknownPin(pins map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		if !slices.Contains(releasedChecks(), name) {
			return fmt.Errorf("checkVersions pins %s, which is no check; `fitness list` names them", name)
		}
	}
	return nil
}

// releasedChecks names every built-in and installed check but the ones the
// engine judges, sorted: the checks fitness-checks releases.
func releasedChecks() []string {
	names := slices.Collect(maps.Keys(builtins))
	for name := range installedPrograms {
		if builtins[name] == nil {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return slices.DeleteFunc(names, func(name string) bool { return engineJudges[name] != nil })
}
