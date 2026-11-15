package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// noChecksCommands are the subcommands that run no check, so they fetch none.
var noChecksCommands = []string{"llms", "docs", "help", "--help", "-h", "check-versions", "init", "strip-front-matter", "adr-migrate", "boxes", "perf", "comments"}

// fetchMissingChecks installs the checks the repo pins, or every released
// check at v1 when it pins none, once any of them is missing. The engine holds
// no checks, so a plain `fitness` after `go install` judges as an install does.
func fetchMissingChecks(args []string) {
	if !needsChecks(args) {
		return
	}
	pins, err := checkVersions(".")
	if err != nil || !missingPins(pins) {
		return
	}
	dir, err := installPins(pins)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: could not install the checks the repo pins: %v; install them with `fitness-install --with-checks`\n", err)
		return
	}
	_ = os.Setenv(checkkit.ChecksDirEnv, dir)
	installedPrograms = checkkit.Discover()
}

// needsChecks reports whether a run with args runs checks and has none handed
// to it, as fitness-install hands them through FITNESS_CHECKS_DIR.
func needsChecks(args []string) bool {
	if os.Getenv(checkkit.ChecksDirEnv) != "" || builtinsOff() {
		return false
	}
	return len(args) == 0 || !slices.Contains(noChecksCommands, args[0])
}

// missingPins reports whether any pinned check is not installed.
func missingPins(pins map[string]string) bool {
	return slices.ContainsFunc(slices.Collect(maps.Keys(pins)), func(name string) bool { return installedPrograms[name] == "" })
}

// installPins installs each pinned check, verified by checksum, into the
// user's cache, and returns the folder that holds them.
func installPins(pins map[string]string) (string, error) {
	installer, err := distribution.New("", "", os.Stderr)
	if err != nil {
		return "", err
	}
	list := make([]distribution.CheckPin, 0, len(pins))
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		list = append(list, distribution.CheckPin{Name: name, Pin: pins[name]})
	}
	return installer.InstallChecks(context.Background(), list)
}
