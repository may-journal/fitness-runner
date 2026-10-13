package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/selection"
)

// optionalChecks are bundled checks outside the default list, which a repo
// selects by name.
var optionalChecks = []string{"go-test-coverage"}

// localBins maps this run's local checks, from .fitnessrc.json's
// localChecks directory, to their programs. loadLocalChecks sets it once per
// run, before any check resolves.
var localBins map[string]string

// loadLocalChecks finds the repo's local checks, so every later lookup
// searches them before the bundled programs.
func loadLocalChecks(cfg *conf.Config, root string) error {
	bins, err := cfg.LocalCheckBinaries(root)
	localBins = bins
	return err
}

// findCheckBinary returns the program for a check name: the repo's local
// check when one has that name, else the check bundled beside fitness.
func findCheckBinary(name string) (string, error) {
	if bin, ok := localBins[name]; ok {
		return bin, nil
	}
	return checkkit.Binary(name)
}

// catalog is the built-in check list followed by the local checks that add
// a new name, in name order. A local check with a built-in name keeps that
// check's place.
func catalog() []string {
	names := append([]string{}, allChecks...)
	for _, name := range localNames() {
		if !builtIn(name) {
			names = append(names, name)
		}
	}
	return names
}

// replacedChecks lists the local checks that share a built-in check's name.
func replacedChecks() []string {
	var replaced []string
	for _, name := range localNames() {
		if builtIn(name) {
			replaced = append(replaced, name)
		}
	}
	return replaced
}

// allowLocalChecks enforces ADR 0006 for the resolved policy and names each
// replaced built-in, so a run never swaps one in silently.
func allowLocalChecks(policy string) error {
	replaced := replacedChecks()
	if err := selection.AllowReplacements(policy, replaced); err != nil {
		return err
	}
	if len(replaced) > 0 {
		fmt.Fprintln(os.Stderr, "fitness: local checks replace built-in "+strings.Join(replaced, ","))
	}
	return nil
}

// repoChecks applies the repo's own check settings to the resolved names,
// under ADR 0006: local checks that replace a bundled one, then
// disabledChecks.
func repoChecks(cfg *conf.Config, policy string, names []string) ([]string, error) {
	if err := allowLocalChecks(policy); err != nil {
		return nil, err
	}
	var disabled []string
	if cfg != nil {
		disabled = cfg.DisabledChecks
	}
	return selection.Disable(policy, names, disabled, append(catalog(), optionalChecks...))
}

// localNames is the local check names in order.
func localNames() []string {
	names := make([]string, 0, len(localBins))
	for name := range localBins {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// builtIn reports whether a bundled check has the name.
func builtIn(name string) bool {
	return slices.Contains(allChecks, name) || slices.Contains(optionalChecks, name)
}
