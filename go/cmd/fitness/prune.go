package main

import (
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// pruneCache marks the running engine's bundle as used, then prunes the cache
// when args run checks, keeping the engine and its check set.
func pruneCache(args []string) {
	markRunning()
	if len(args) > 0 && !runsChecks(args[0]) {
		return
	}
	cache, err := distribution.CacheDir()
	if err != nil {
		return
	}
	removed := distribution.Prune(cache, inUse(), func() bool { return metadata.UnreachableGitHub() == "" })
	if len(removed) > 0 {
		fmt.Fprintf(os.Stderr, "fitness: pruned %d cached engine and check releases unused for 30 days or superseded from %s\n", len(removed), cache)
	}
}

// markRunning marks the running engine's cache bundle as used, so one run
// by its saved path stays.
func markRunning() {
	if exe, err := os.Executable(); err == nil {
		distribution.MarkUsed(exe)
	}
}

// inUse lists the cache paths this process uses: its own executable and the
// check set handed to it.
func inUse() []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, exe)
	}
	if dir := os.Getenv(checkkit.ChecksDirEnv); dir != "" {
		paths = append(paths, dir)
	}
	return paths
}

// runPrefetch downloads and verifies every check the repo pins, at every
// pinned version, so later runs need no network, and prints their folder.
func runPrefetch(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "fitness prefetch: takes no arguments")
		return 2
	}
	pins, err := checkVersions(".")
	var dir string
	if err == nil {
		dir, err = installPins(pins)
	}
	if err != nil {
		report.Error("fitness prefetch", err)
		return 1
	}
	fmt.Println(dir)
	report.Outcome("fitness prefetch", "Every check the repo pins is cached.")
	return 0
}
