package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// noChecksCommands are the subcommands that fetch no check: those that run
// none, and claude-hook, which must answer an edit within 300ms.
var noChecksCommands = []string{"llms", "docs", "help", "--help", "-h", "check-versions", "init", "strip-front-matter", "adr-migrate", "boxes", "perf", "comments", "claude-hook"}

// fetchMissingChecks installs the checks the repo pins, or every released
// check at v1 when it pins none, once any of them is missing. The engine holds
// no checks, so a plain `fitness` after `go install` judges as an install does.
func fetchMissingChecks(args []string) {
	if !needsChecks(args) {
		return
	}
	pins, err := checkVersions(".")
	if err != nil {
		return
	}
	missing := missingPins(pins)
	if len(missing) == 0 {
		return
	}
	dir, err := installPins(missing)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: could not install the checks the repo pins: %v; install them with `fitness-install --with-checks`\n", err)
		return
	}
	addFetched(dir)
}

// addFetched adds each program in dir whose check no installed program
// provides, so a fetched check never shadows one already installed.
func addFetched(dir string) {
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
	if os.Getenv(checkkit.ChecksDirEnv) != "" || builtinsOff() {
		return false
	}
	return len(args) == 0 || !slices.Contains(noChecksCommands, args[0])
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
	list := make([]distribution.CheckPin, 0, len(pins))
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		list = append(list, distribution.CheckPin{Name: name, Pin: pins[name]})
	}
	return installer.InstallChecks(context.Background(), list)
}
