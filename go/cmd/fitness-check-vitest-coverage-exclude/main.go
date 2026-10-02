// Command fitness-check-vitest-coverage-exclude validates that Vitest
// coverage excludes nothing — the Go port of the vitest-coverage-exclude
// check, with its allowed set of test, type, and barrel patterns removed. The TS check evaluated the
// vitest config as JavaScript; this port scans it textually via
// internal/vitestconf, so exclude entries built from variables, spreads, or
// imports are invisible — the same judgment the TS applied to non-string
// entries at runtime, one level earlier.
//
// A root without any config source is judged against the shared config
// directory: an installed node_modules/@mayjournal/fitness-shared when
// present, else the vitest.config.mjs embedded in this binary, materialized
// to the cache (internal/sharedconf).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/vitestconf"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "vitest-coverage-exclude"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if len(walkfs.FilesByExt(root, "package.json")) == 0 {
		return checkkit.Pass(0), nil
	}
	return judge(loadExclude(root)), nil
}

// loadExclude resolves the coverage exclude list the way the TS check's
// loadVitestConfig(root, getFitnessRunnerRoot()) call did: from root when it
// has a config source, else from the shared config directory
// (sharedconf.ResolveDir — installed package first, embedded copy last).
func loadExclude(root string) []string {
	if exclude, found := vitestconf.LoadExcludeFromRoot(root); found {
		return exclude
	}
	exclude, _ := vitestconf.LoadExcludeFromRoot(sharedconf.ResolveDir(root))
	return exclude
}

// judge fails every coverage exclude entry; filesChecked is always 1 — the
// one config source — exactly like the TS check.
func judge(exclude []string) checkkit.Result {
	var errs []string
	for _, pattern := range exclude {
		errs = append(errs, fmt.Sprintf(
			"Vitest coverage exclude must be empty, so coverage judges every file; remove: %s",
			jsonQuote(pattern)))
	}
	if len(errs) == 0 {
		return checkkit.Pass(1)
	}
	return checkkit.Fail(1, errs...)
}

// jsonQuote renders a pattern exactly as the TS JSON.stringify did —
// double-quoted, without Go's default HTML escaping.
func jsonQuote(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}
