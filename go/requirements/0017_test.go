package requirements

import "testing"

// changelogRun runs `fitness-install -- changelog` on happyRepo with files
// written over it.
func changelogRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "changelog")
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

// upstreamChangelog is a CHANGELOG.md as an upstream project keeps it:
// undated, typed sections of numbered items under a version heading.
const upstreamChangelog = "# Changelog\n\n## v2.0.0\n\n### Added\n\n1. Catalog-level publishing\n2. Plugin hooks\n\n### Removed\n1. Old lint tools\n"

func Test0017_21(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, withEntries(map[string]string{
		"undated-change":               entryFileText("2026.10.11.0900", "Feat: undated change."),
		"2026-10-11-0800-moved-change": entryFileText("2026.10.11.1000", "Fix: moved change."),
	}))
	sees(t, out, code, 1,
		"changelog/undated-change.md must be named changelog/2026-10-11-0900-undated-change.md: its heading's date, then the change",
		"changelog/2026-10-11-0800-moved-change.md must be named changelog/2026-10-11-1000-moved-change.md")
}

func Test0017_22(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, withEntries(map[string]string{
		"2026-10-11-0900-good-change":  entryFileText("2026.10.11.0900", "Feat: good change."),
		"2026-10-11-1000-two-headings": entryFileText("2026.10.11.1000", "Fix: one.") + "\n" + entryFileText("2026.10.11.1100", "Fix: two."),
		"2026-10-11-1200-undated":      "### Added\n\n- Feat: one.\n- Fix: two.\n- Docs: three.\n",
	}))
	sees(t, out, code, 1,
		"changelog/2026-10-11-1000-two-headings.md must hold one ### yyyy.mm.dd.HHMM heading and its bullets, nothing else",
		"changelog/2026-10-11-1200-undated.md must hold one ### yyyy.mm.dd.HHMM heading")
}

func Test0017_23(t *testing.T) {
	t.Parallel()
	files := withEntries(map[string]string{
		"2026-10-10-0900-older-change": entryFileText("2026.10.10.0900", "Fix: older change."),
		"2026-10-11-0900-newer-change": entryFileText("2026.10.11.0900", "Feat: newer change."),
	})
	files["CHANGELOG.md"] = upstreamChangelog
	files["package.json"] = `{"name": "app", "private": true, "version": "2.0.1"}` + "\n"
	files["package-lock.json"] = `{"name": "app", "version": "2.0.0"}` + "\n"
	out, code := changelogRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "changelog", "2")
}

func Test0017_24(t *testing.T) {
	t.Parallel()
	for _, changelog := range []string{upstreamChangelog, ""} {
		out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelog})
		sees(t, out, code, 0, "All 1 checks passed")
		passedWithNoFiles(t, out, "changelog")
	}
}
