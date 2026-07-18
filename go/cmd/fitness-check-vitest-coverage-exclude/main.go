// Command fitness-check-vitest-coverage-exclude validates that Vitest
// coverage exclude patterns stay within the conventional allowed set — the
// Go port of the vitest-coverage-exclude check. The TS check evaluated the
// vitest config as JavaScript; this port scans it textually via
// internal/vitestconf, so exclude entries built from variables, spreads, or
// imports are invisible — the same judgment the TS applied to non-string
// entries at runtime, one level earlier.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/vitestconf"
)

// allowedCoverageExcludePatterns is the TS ALLOWED_COVERAGE_EXCLUDE_PATTERNS
// list: declaration, type-only, test, spec, bench, barrel index, worker
// entry — Vitest excludes tests by default.
var allowedCoverageExcludePatterns = []string{
	"**/*.bench.ts",
	"**/*.d.ts",
	"**/*.types.ts",
	"**/*.test.ts",
	"**/*.spec.ts",
	"**/index.ts",
	"**/run-one-check-worker.ts",
}

// allowedSuffixes is the TS ALLOWED_SUFFIXES regex: the conventional endings
// a .ts exclude pattern may carry.
var allowedSuffixes = regexp.MustCompile(
	`(\.(bench\.ts|d\.ts|types\.ts|test\.ts|spec\.ts)|index\.ts|run-one-check-worker\.ts)$`)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "vitest-coverage-exclude"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return judge(vitestconf.LoadExclude(root)), nil
}

// judge applies the allowed-pattern rule to the exclude list; filesChecked
// is always 1 — the one config source — exactly like the TS check.
func judge(exclude []string) checkkit.Result {
	var errs []string
	for _, pattern := range exclude {
		if !isDisallowedTsPattern(pattern) {
			continue
		}
		errs = append(errs, fmt.Sprintf(
			"Vitest coverage exclude only allows %s; disallowed: %s",
			strings.Join(allowedCoverageExcludePatterns, ", "), jsonQuote(pattern)))
	}
	if len(errs) == 0 {
		return checkkit.Pass(1)
	}
	return checkkit.Fail(1, errs...)
}

// isDisallowedTsPattern reports whether an exclude pattern falls outside the
// conventional set: directory globs (…/**) always fail, non-.ts patterns
// always pass, .ts patterns must end in an allowed suffix.
func isDisallowedTsPattern(pattern string) bool {
	t := strings.TrimSpace(pattern)
	if strings.HasSuffix(t, "/**") {
		return true
	}
	if !strings.Contains(t, ".ts") {
		return false
	}
	return !allowedSuffixes.MatchString(t)
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
