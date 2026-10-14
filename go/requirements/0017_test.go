package requirements

import "testing"

// changelogRun runs `fitness-install -- changelog` on happyRepo with files
// written over it.
func changelogRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "changelog")
}

// changelogWith is a changelog whose sections carry the given headings.
func changelogWith(headings ...string) string {
	body := "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n"
	for _, h := range headings {
		body += "\n" + h + "\n\n- Docs: add the readme.\n"
	}
	return body
}

// changelogPackages is a changelog dated 2026.10.08.1400 with the given
// package.json and package-lock.json bodies.
func changelogPackages(pkg, lock string) map[string]string {
	return map[string]string{
		"CHANGELOG.md":      changelogWith("### 2026.10.08.1400"),
		"package.json":      pkg,
		"package-lock.json": lock,
	}
}

const changelogVersion = `{"name": "app", "private": true, "version": "0.1.0-2026.10.08.1400"}` + "\n"

func Test0017_1(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": ""})
	sees(t, out, code, 1, "missing root CHANGELOG.md")
}

func Test0017_2(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelogWith("## 2026.10.08.1400")})
	sees(t, out, code, 1, "CHANGELOG.md must have at least one ### yyyy.mm.dd.HHMM section")
}

func Test0017_5(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(`{"version": "0.1.0-2026.10.08.1500"}`+"\n", ""))
	sees(t, out, code, 1, "package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)")
}

func Test0017_18(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(changelogVersion, `{"version": "0.1.0-2026.10.07.0900"}`+"\n"))
	sees(t, out, code, 1, "package-lock.json version must match package.json version")
}

func Test0017_19(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages("not json\n", ""))
	sees(t, out, code, 1, "package.json is invalid JSON")
	out, code = changelogRun(t, changelogPackages(changelogVersion, "not json\n"))
	sees(t, out, code, 1, "package-lock.json is invalid JSON")
}

// entryFileText is an entry file under stamp with three typed bullets, the
// first one item.
func entryFileText(stamp, item string) string {
	return "---\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n\n### " + stamp + "\n\n- " + item + "\n- Test: cover it.\n- Docs: describe it.\n"
}

// withEntries is files over happyRepo holding each named entry file.
func withEntries(entries map[string]string) map[string]string {
	files := map[string]string{}
	for name, body := range entries {
		files["changelog/"+name+".md"] = body
	}
	return files
}

func Test0017_15(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelogWith("### 2026.10.08.1400 hotfix", "### 2026.10.07.0900")})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = notesChecks(t, notesHead+notesRelease(notesTwo)+notesEntry, "changelog")
	sees(t, out, code, 0, "All 1 checks passed")
	files := withEntries(map[string]string{
		"older-change": entryFileText("2026.10.10.0900", "Fix: older change."),
		"newer-change": entryFileText("2026.10.11.0900", "Feat: newer change."),
	})
	files["CHANGELOG.md"] = notesHead + notesRelease(notesTwo) + notesEntry
	files["package.json"] = `{"name": "app", "private": true, "version": "0.1.0-2026.10.11.0900"}` + "\n"
	out, code = changelogRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0017_8(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(changelogVersion, changelogVersion))
	sees(t, out, code, 0, "All 1 checks passed")
}

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

func Test0017_17(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelogWith("### 2026.10.08.1400", "### 2026.10.07")})
	sees(t, out, code, 1, `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026.10.07")`)
	out, code = notesChecks(t, notesHead+"### Features\n\n- Feat: stray.\n\n"+notesEntry, "changelog")
	sees(t, out, code, 1, `invalid: "### Features"`)
	out, code = changelogRun(t, withEntries(map[string]string{
		"good-change":  entryFileText("2026.10.11.0900", "Feat: good change."),
		"two-headings": entryFileText("2026.10.11.1000", "Fix: one.") + "\n" + entryFileText("2026.10.11.1100", "Fix: two."),
	}))
	sees(t, out, code, 1, "changelog/two-headings.md must hold one ### yyyy.mm.dd.HHMM heading and its bullets, nothing else")
}
