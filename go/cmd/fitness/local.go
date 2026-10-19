package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/selection"
)

// localBins maps this run's local checks, from .fitnessrc.json's
// localChecks directory, to their programs. loadLocalChecks sets it once per
// run, before any check resolves.
var localBins map[string]string

// turnedOff reports whether an external policy's disabledChecks names name,
// so a hook skips it rather than fail on an empty selection.
func turnedOff(cfg *conf.Config, name string) bool {
	return cfg != nil && cfg.Policy == selection.External && slices.Contains(cfg.DisabledChecks, name)
}

// loadLocalChecks finds the repo's local checks, so every later lookup
// searches them before the bundled programs.
func loadLocalChecks(cfg *conf.Config, root string) error {
	bins, err := cfg.LocalCheckBinaries(root)
	if err != nil {
		return err
	}
	for name := range bins {
		if _, taken := subcommands[name]; taken || name == "help" {
			return fmt.Errorf("local check %s shares a fitness subcommand's name, so `fitness %s` could not run it; rename it", name, name)
		}
	}
	localBins = bins
	return nil
}

// findCheckBinary returns the program for a check name: the repo's local
// check when one has that name, else the check bundled beside fitness.
func findCheckBinary(name string) (string, error) {
	if bin, ok := localBins[name]; ok {
		return bin, nil
	}
	return checkkit.Binary(name)
}

// checkCatalog is the built-in check list followed by the local checks that add
// a new name, in name order. A local check with a built-in name keeps that
// check's place.
func checkCatalog() []string {
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

// allowLocalChecks enforces ADR 0009 for the resolved policy and names each
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
// under ADR 0009: local checks that replace a bundled one, then
// disabledChecks.
func repoChecks(cfg *conf.Config, policy string, names []string) ([]string, error) {
	if err := allowLocalChecks(policy); err != nil {
		return nil, err
	}
	var disabled []string
	if cfg != nil {
		disabled = cfg.DisabledChecks
	}
	return selection.Disable(policy, names, disabled, append(checkCatalog(), lang.Ignored()...))
}

// configPolicy is the repo's configured policy, org unless it says external.
func configPolicy(cfg *conf.Config) string {
	if cfg != nil && cfg.Policy == selection.External {
		return selection.External
	}
	return selection.Org
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
	return slices.Contains(allChecks, name) || slices.Contains(lang.Ignored(), name)
}

// runGH runs a GitHub command with the repo's check settings, so an issue or
// PR body meets the same local and disabled checks as the repo's files.
func runGH(name string) int {
	if err := useRepoChecks(); err != nil {
		report.Error("fitness "+name, err)
		return 1
	}
	return metadata.RunGH(name)
}

// useRepoChecks checks the repo's settings under ADR 0009, as a suite run
// does, then points the GitHub commands at its local and disabled checks.
func useRepoChecks() error {
	_, cfg, _, err := blobConfig()
	if err != nil {
		return err
	}
	metadata.RepoChecks.Find = findCheckBinary
	metadata.RepoChecks.Off = func(name string) bool { return turnedOff(cfg, name) }
	return nil
}
