package release

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Release Please writes a release as a `## [x.y.z](compare) (date)` heading,
// `### Features`-style subheadings, and `* **scope:** text` bullets. The
// house CHANGELOG format, which the changelog checks require, is a
// `## x.y.z (date)` heading over `- Feat: scope: text` bullets.
var (
	pleaseHeading = regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)\]\([^)]*\) \(([^)]+)\)\s*$`)
	pleaseBullet  = regexp.MustCompile(`^\* (?:\*\*([^*]+):\*\* )?(.+)$`)
	houseEntry    = regexp.MustCompile(`^(### \d{4}\.\d{2}\.\d{2}\.\d{4}|## )`)
)

// noteTypes maps Release Please's subheadings to house bullet types.
var noteTypes = map[string]string{
	"Features":                 "Feat",
	"Bug Fixes":                "Fix",
	"Performance Improvements": "Perf",
	"Reverts":                  "Revert",
	"Documentation":            "Docs",
	"Miscellaneous Chores":     "Chore",
	"Code Refactoring":         "Refactor",
	"Tests":                    "Test",
	"Build System":             "Build",
	"Continuous Integration":   "Ci",
	"Styles":                   "Style",
}

// HouseChangelog rewrites the top Release Please section of CHANGELOG.md
// into the house format and leaves every other line as it was. A changelog
// whose top release is already in the house format is left untouched.
func (c Config) HouseChangelog() error {
	path := filepath.Join(c.Root, "CHANGELOG.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	start := pleaseStart(lines)
	if start < 0 {
		return nil
	}
	section, end, err := houseSection(lines, start)
	if err != nil {
		return err
	}
	out := append(append(append([]string{}, lines[:start]...), section...), lines[end:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// pleaseStart is the index of the first Release Please release heading, or
// -1 when the changelog has none.
func pleaseStart(lines []string) int {
	for i, line := range lines {
		if pleaseHeading.MatchString(line) {
			return i
		}
	}
	return -1
}

// houseSection converts the release starting at lines[start] and returns
// the new lines and the index just past the old section, which ends at the
// next timestamped entry or release heading.
func houseSection(lines []string, start int) ([]string, int, error) {
	m := pleaseHeading.FindStringSubmatch(lines[start])
	out := []string{fmt.Sprintf("## %s (%s)", m[1], m[2]), ""}
	kind := ""
	end := start + 1
	for ; end < len(lines) && !houseEntry.MatchString(lines[end]); end++ {
		var err error
		if out, kind, err = houseLine(out, kind, strings.TrimSpace(lines[end])); err != nil {
			return nil, 0, err
		}
	}
	return append(out, ""), end, nil
}

// houseLine folds one Release Please line into the house section: a
// subheading sets the bullet type, a bullet becomes a typed house bullet,
// and blank lines are dropped.
func houseLine(out []string, kind, line string) ([]string, string, error) {
	if heading, ok := strings.CutPrefix(line, "### "); ok {
		next, known := noteTypes[heading]
		if !known {
			return nil, "", fmt.Errorf("CHANGELOG.md: unknown release subheading %q", heading)
		}
		return out, next, nil
	}
	if b := pleaseBullet.FindStringSubmatch(line); b != nil && kind != "" {
		return append(out, houseBullet(kind, b[1], b[2])), kind, nil
	}
	return out, kind, nil
}

// houseBullet is one `- Type: scope: text` bullet; the scope is left out
// when Release Please gave none.
func houseBullet(kind, scope, text string) string {
	if scope == "" {
		return "- " + kind + ": " + text
	}
	return "- " + kind + ": " + scope + ": " + text
}
