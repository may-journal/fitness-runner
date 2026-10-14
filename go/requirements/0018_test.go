package requirements

import (
	"strings"
	"testing"
)

// changelogBullets runs `fitness-install -- changelog-bullets` on happyRepo
// with the given CHANGELOG.md body, or none when it is empty.
func changelogBullets(t *testing.T, changelog string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"CHANGELOG.md": changelog})
	return fitness(t, repo, nil, "changelog-bullets")
}

// changelogBulletsDoc is a changelog whose one section, opened on line 9,
// holds bullets, so its first bullet sits on line 11.
func changelogBulletsDoc(bullets ...string) string {
	doc := "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n### 2026.10.08.1400\n\n"
	for _, b := range bullets {
		doc += "- " + b + "\n"
	}
	return doc
}

func Test0018_16(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, changelogBulletsDoc("Docs: add the readme.", "Docs: add the changelog."))
	sees(t, out, code, 1, `CHANGELOG.md section "2026.10.08.1400" has 2 bullets; keep entries to 3-5 bullets`)
	entry := "### 2026.10.09.1500\n\n* Fix: one.\n* Fix: two.\n* Fix: three.\n\n"
	out, code = notesChecks(t, notesHead+entry, "changelog-bullets")
	sees(t, out, code, 1, `"2026.10.09.1500" has 0 bullets`)
}

func Test0018_17(t *testing.T) {
	t.Parallel()
	files := withEntries(map[string]string{
		"short-entry":   "### 2026.10.11.0900\n\n- Feat: one.\n- Fix: two.\n",
		"untyped-entry": "### 2026.10.11.1000\n\n- Feat: one.\n- Fix: two.\n- no type here.\n",
	})
	out, code := fitness(t, example(t, "happyRepo", files), nil, "changelog-bullets")
	sees(t, out, code, 1, `changelog/short-entry.md section "2026.10.11.0900" has 2 bullets`,
		"changelog/untyped-entry.md:5: bullet must start with a semantic type")
}

func Test0018_6(t *testing.T) {
	t.Parallel()
	old := changelogBulletsDoc("Feat: one.", "Fix: two.", "Docs: three.") + "\n### 2026.05.01.0000\n\n- Docs: one old bullet.\n"
	out, code := changelogBullets(t, old)
	sees(t, out, code, 1, `CHANGELOG.md section "2026.05.01.0000" has 1 bullets; keep entries to 3-5 bullets`)
}

func Test0018_7(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, "")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "changelog-bullets")
}

func Test0018_12(t *testing.T) {
	t.Parallel()
	long := "Feat: " + strings.Repeat("x", 359)
	wrapped := "Fix: " + strings.Repeat("x", 200) + "\n  " + strings.Repeat("y", 200)
	out, code := changelogBullets(t, changelogBulletsDoc(long, wrapped, "Docs: three."))
	sees(t, out, code, 1,
		"CHANGELOG.md:11: bullet is 365 characters; keep each under 365",
		"CHANGELOG.md:12: bullet is 406 characters; keep each under 365")
}

func Test0018_15(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, changelogBulletsDoc("Feat: one.", "fix(release)!: two.", "chore(ci): three.", "Revert: four.", "Added the readme."))
	sees(t, out, code, 1, "CHANGELOG.md:15: bullet must start with a semantic type")
	if n := strings.Count(out, "must start with a semantic type"); n != 1 {
		t.Errorf("only the untyped bullet must fail, got %d failures:\n%s", n, out)
	}
	out, code = notesChecks(t, notesHead+notesRelease(notesTwo)+notesEntry, "changelog-bullets")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0018_14(t *testing.T) {
	t.Parallel()
	doc := changelogBulletsDoc("Docs: add the readme.", "Docs: add the changelog.", "Docs: add the license note.") +
		"\n## 1.1.0\n\n- Feat: one.\n- Feat: two.\n- Feat: three.\n- Fix: four.\n- Fix: five.\n- Test: six.\n"
	repo := example(t, "happyRepo", map[string]string{
		"CHANGELOG.md":    doc,
		".fitnessrc.json": `{"proseBudget": {"maxListItems": 5}}` + "\n",
	})
	out, code := fitness(t, repo, nil, "changelog-bullets")
	sees(t, out, code, 1, `section "1.1.0" has 6 bullets; keep releases to 1-5 bullets`)
	split := notesRelease(map[string][]string{"Features": {"a", "b", "c", "d", "e"}, "Bug Fixes": {"f", "g", "h", "i"}})
	splitOut, splitCode := notesChecks(t, notesHead+notesEntry+split, "changelog-bullets")
	sees(t, splitOut, splitCode, 1, "has 9 bullets; keep releases to 1-8 bullets")
	for _, o := range []string{out, splitOut} {
		if strings.Contains(o, "2026.10.08.1400") || strings.Contains(o, "2026.10.09.1500") {
			t.Errorf("the entry above the release must not fail:\n%s", o)
		}
	}
}
