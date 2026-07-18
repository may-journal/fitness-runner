// Command fitness-check-changelog validates the root CHANGELOG.md — the Go
// port of the changelog check. Every ### heading must be
// ### yyyy.mm.dd.HHMM, the package.json version suffix must match the first
// heading's timestamp, and package-lock.json (when present) must carry the
// same version as package.json. Version files that fail to parse report
// "<file> is invalid JSON", exactly like the TypeScript original's
// JSON.parse path.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	changelogMD     = "CHANGELOG.md"
	packageJSON     = "package.json"
	packageLockJSON = "package-lock.json"

	errMissing          = "missing root CHANGELOG.md"
	msgVersionChangelog = "package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)"
	msgVersionLock      = "package-lock.json version must match package.json version"
)

var (
	// datedSectionRe is deliberately not end-anchored: trailing text after
	// the timestamp is allowed, matching the TS regex.
	datedSectionRe = regexp.MustCompile(`^###\s+(\d{4}\.\d{2}\.\d{2}\.\d{4})`)
	// versionSuffixRe IS end-anchored: the timestamp must terminate the
	// package version string.
	versionSuffixRe = regexp.MustCompile(`-(\d{4}\.\d{2}\.\d{2}\.\d{4})$`)
	h3Re            = regexp.MustCompile(`^###\s`)
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	raw, err := os.ReadFile(filepath.Join(root, changelogMD))
	if err != nil {
		if os.IsNotExist(err) {
			return checkkit.Fail(1, errMissing), nil
		}
		return checkkit.Result{}, err
	}
	return judge(root, string(raw)), nil
}

// judge applies the gates in the TS order: heading format first (its errors
// short-circuit the version checks), then version correlation. filesChecked
// is always 1 — the one CHANGELOG.md.
func judge(root, content string) checkkit.Result {
	if errs := formatErrors(content); len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	if errs := versionErrors(root, firstTimestamp(content)); len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}

// h3Lines returns the lines that are ### headings (level-3 only).
func h3Lines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		if h3Re.MatchString(line) {
			out = append(out, line)
		}
	}
	return out
}

// formatErrors returns the heading-format errors for the CHANGELOG content:
// at least one ### heading must exist, and every one must carry the
// yyyy.mm.dd.HHMM timestamp.
func formatErrors(content string) []string {
	h3s := h3Lines(content)
	if len(h3s) == 0 {
		return []string{"CHANGELOG.md must have at least one ### yyyy.mm.dd.HHMM section"}
	}
	var errs []string
	for _, line := range h3s {
		if !datedSectionRe.MatchString(line) {
			errs = append(errs, fmt.Sprintf(
				`every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "%s")`, strings.TrimSpace(line)))
		}
	}
	return errs
}

// firstTimestamp extracts the timestamp of the first dated ### heading. Only
// called after formatErrors passed, so a match always exists.
func firstTimestamp(content string) string {
	for _, line := range h3Lines(content) {
		if m := datedSectionRe.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// jsonVersion is the "version" property of a parsed JSON file, preserving
// the JavaScript distinction between an absent property (undefined) and an
// explicit null — the TS check compared the raw values with ===.
type jsonVersion struct {
	value   any
	present bool
}

// versionErrors returns the version-correlation errors for root: package.json
// suffix vs the changelog timestamp, and package-lock.json vs package.json.
// A repo without a package.json has nothing to correlate.
func versionErrors(root, changelogTS string) []string {
	pkgPath := filepath.Join(root, packageJSON)
	if _, err := os.Stat(pkgPath); err != nil {
		return nil
	}
	pkgVersion, ok := readJSONVersion(pkgPath)
	if !ok {
		return []string{packageJSON + " is invalid JSON"}
	}
	var errs []string
	if msg := packageChangelogMismatch(pkgVersion, changelogTS); msg != "" {
		errs = append(errs, msg)
	}
	if msg := lockVersionError(root, pkgVersion); msg != "" {
		errs = append(errs, msg)
	}
	return errs
}

// readJSONVersion parses the file and pulls its top-level "version". ok is
// false when the TS check's JSON.parse-and-dereference would have thrown:
// unreadable file, invalid JSON, or a null document (null.version throws).
// Non-object documents parse fine and simply have no version.
func readJSONVersion(path string) (v jsonVersion, ok bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return jsonVersion{}, false
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return jsonVersion{}, false
	}
	if doc == nil {
		return jsonVersion{}, false
	}
	obj, isObj := doc.(map[string]any)
	if !isObj {
		return jsonVersion{}, true
	}
	value, present := obj["version"]
	return jsonVersion{value: value, present: present}, true
}

// packageChangelogMismatch returns the mismatch error unless the package
// version's timestamp suffix equals the changelog's first-heading timestamp.
func packageChangelogMismatch(pkgVersion jsonVersion, changelogTS string) string {
	if versionTimestamp(pkgVersion) == changelogTS {
		return ""
	}
	return msgVersionChangelog
}

// versionTimestamp extracts yyyy.mm.dd.HHMM from a version like
// "0.1.0-2026.02.16.1900". Only strings are inspected: no other JSON value
// can stringify to something the end-anchored suffix regex matches, so this
// is outcome-equivalent to the TS String() coercion.
func versionTimestamp(v jsonVersion) string {
	s, isString := v.value.(string)
	if !isString {
		return ""
	}
	m := versionSuffixRe.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}

// lockVersionError returns the lock error when package-lock.json exists and
// its version is not strictly equal to package.json's, or the lock cannot be
// parsed.
func lockVersionError(root string, pkgVersion jsonVersion) string {
	lockPath := filepath.Join(root, packageLockJSON)
	if _, err := os.Stat(lockPath); err != nil {
		return ""
	}
	lockVersion, ok := readJSONVersion(lockPath)
	if !ok {
		return packageLockJSON + " is invalid JSON"
	}
	if strictlyEqual(lockVersion, pkgVersion) {
		return ""
	}
	return msgVersionLock
}

// strictlyEqual mirrors JavaScript === on the two version values: absent
// (undefined) equals only absent, null equals only null, primitives compare
// by value, and objects/arrays from two separate parses are never equal.
func strictlyEqual(a, b jsonVersion) bool {
	if a.present != b.present {
		return false
	}
	if !a.present {
		return true
	}
	if !comparableJSON(a.value) || !comparableJSON(b.value) {
		return false
	}
	return a.value == b.value
}

// comparableJSON reports whether the decoded JSON value is a Go-comparable
// primitive (nil, bool, float64, string) rather than a map or slice.
func comparableJSON(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return false
	}
	return true
}
