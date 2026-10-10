package requirements

import (
	"strings"
	"testing"
)

// notesHead opens a changelog in the house layout.
const notesHead = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n"

// notesEntry is a timestamped entry that follows the house rules.
const notesEntry = "### 2026.10.09.1500\n\n- Fix: one.\n- Fix: two.\n- Fix: three.\n\n"

// notesRelease is a 1.1.1 release as Release Please writes it: a linked
// version heading, then each subheading with one bullet per scope given.
func notesRelease(subheadings map[string][]string) string {
	s := "## [1.1.1](https://github.com/o/r/compare/v1.1.0...v1.1.1) (2026-10-10)\n\n"
	for _, h := range []string{"Features", "Bug Fixes"} {
		if len(subheadings[h]) == 0 {
			continue
		}
		s += "\n### " + h + "\n\n"
		for _, scope := range subheadings[h] {
			s += "* **" + scope + ":** change it ([#239](https://github.com/o/r/issues/239)) ([c9f4729](https://github.com/o/r/commit/c9f4729))\n"
		}
		s += "\n"
	}
	return s
}

// notesChecks runs the named checks on happyRepo with changelog.
func notesChecks(t *testing.T, changelog, checks string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"CHANGELOG.md": changelog})
	return fitness(t, repo, nil, "--policy=external", "--checks="+checks, "--all")
}

// notesTwo is a release with one feature and one fix.
var notesTwo = map[string][]string{"Features": {"install"}, "Bug Fixes": {"release"}}

func Test0064_1(t *testing.T) {
	t.Parallel()
	out, code := notesChecks(t, notesHead+notesRelease(notesTwo)+notesEntry, "changelog")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0064_2(t *testing.T) {
	t.Parallel()
	out, code := notesChecks(t, notesHead+"### Features\n\n- Feat: stray.\n\n"+notesEntry, "changelog")
	sees(t, out, code, 1, `invalid: "### Features"`)
}

func Test0064_3(t *testing.T) {
	t.Parallel()
	out, code := notesChecks(t, notesHead+notesRelease(notesTwo)+notesEntry, "changelog-bullets,markdown-no-bold-italic")
	sees(t, out, code, 0, "All 2 checks passed")
}

func Test0064_4(t *testing.T) {
	t.Parallel()
	release := strings.Replace(notesRelease(notesTwo), "change it", "change **everything**", 1)
	out, code := notesChecks(t, notesHead+release+notesEntry, "markdown-no-bold-italic")
	sees(t, out, code, 1, `"**everything**"`)
}

func Test0064_5(t *testing.T) {
	t.Parallel()
	many := map[string][]string{"Features": {"a", "b", "c", "d", "e"}, "Bug Fixes": {"f", "g", "h", "i"}}
	out, code := notesChecks(t, notesHead+notesRelease(many)+notesEntry, "changelog-bullets")
	sees(t, out, code, 1, `has 9 bullets; keep releases to 1-8 bullets`)
}

func Test0064_6(t *testing.T) {
	t.Parallel()
	entry := "### 2026.10.09.1500\n\n* Fix: one.\n* Fix: two.\n* Fix: three.\n\n"
	out, code := notesChecks(t, notesHead+entry, "changelog-bullets")
	sees(t, out, code, 1, `"2026.10.09.1500" has 0 bullets`)
}

func Test0064_7(t *testing.T) {
	t.Parallel()
	changelog := "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n" +
		"## [1.2.3](https://github.com/o/r/compare/v1.2.2...v1.2.3) (2026-10-10)\n\n\n### Bug Fixes\n\n* **release:** fix publishing\n\n" +
		"### 2026.10.08.1400\n\n- Docs: add the readme.\n"
	out, code := releaseChangelog(t, releaseChangelogRelease(changelog))
	sees(t, out, code, 0, "All 1 checks passed")
}
