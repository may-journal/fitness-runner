package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
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

// checkVersions is root's checkVersions with a "*" pin spread over every
// released check, or every released check at currentMajor when unset or empty.
func checkVersions(root string) (map[string]string, error) {
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}
	if cfg == nil || len(cfg.CheckVersions) == 0 {
		return defaultPins()
	}
	return withEveryCheck(cfg.CheckVersions), badPin(cfg.CheckVersions)
}

// everyCheck is the checkVersions name that pins every released check.
const everyCheck = "*"

// withEveryCheck gives every released check the "*" pin, when pins has one.
// A check pinned by its own name keeps its own pin.
func withEveryCheck(pins map[string]string) map[string]string {
	every, ok := pins[everyCheck]
	if !ok {
		return pins
	}
	out := map[string]string{}
	for name := range defaultPins() {
		out[name] = every
	}
	for name, pin := range pins {
		out[name] = pin
	}
	delete(out, everyCheck)
	return out
}

// defaultPins pins every released check to currentMajor, but the ones the
// engine runs itself. The engine holds no check list, so the fitness-checks
// release index names them.
func defaultPins() (map[string]string, error) {
	installer, err := distribution.New("", "", os.Stderr)
	if err != nil {
		return nil, err
	}
	released, err := installer.ReleasedChecks(context.Background())
	if err != nil {
		return nil, err
	}
	pins := map[string]string{}
	for _, name := range released {
		if builtins[name] == nil {
			pins[name] = currentMajor
		}
	}
	return pins, nil
}

// checkName matches a check's name, lowercase words joined by dashes.
var checkName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// pinPattern matches a pin: a major such as v1, or a release such as v1.2.3.
var pinPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))?$`)

// badPin fails the first pin, in name order, that is malformed or names a
// check the engine runs itself. A name with no release fails at install.
func badPin(pins map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		if err := pinError(name, pins[name]); err != nil {
			return err
		}
	}
	return nil
}

// pinError says what is wrong with one pin, or nil.
func pinError(name, pin string) error {
	switch {
	case name != everyCheck && !checkName.MatchString(name):
		return fmt.Errorf("checkVersions names %q, which is no check name; `fitness list` names them", name)
	case engineJudges[name] != nil:
		return fmt.Errorf("checkVersions pins %s, which runs inside the engine, so remove it", name)
	case !pinPattern.MatchString(pin):
		return fmt.Errorf("checkVersions pins %s to %q; use a major such as v1 or a release such as v1.2.3", name, pin)
	}
	return nil
}
