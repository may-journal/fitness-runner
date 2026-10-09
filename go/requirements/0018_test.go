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

func Test0018_1(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, changelogBulletsDoc("Docs: add the readme.", "Docs: add the changelog."))
	sees(t, out, code, 1, `CHANGELOG.md section "2026.10.08.1400" has 2 bullets; keep entries to 3-5 bullets`)
}

func Test0018_2(t *testing.T) {
	t.Parallel()
	long := "Feat: " + strings.Repeat("x", 359)
	out, code := changelogBullets(t, changelogBulletsDoc(long, "Fix: two.", "Docs: three."))
	sees(t, out, code, 1, "CHANGELOG.md:11: bullet is 365 characters; keep each under 365")
}

func Test0018_3(t *testing.T) {
	t.Parallel()
	wrapped := "Feat: " + strings.Repeat("x", 200) + "\n  " + strings.Repeat("y", 200)
	out, code := changelogBullets(t, changelogBulletsDoc(wrapped, "Fix: two.", "Docs: three."))
	sees(t, out, code, 1, "CHANGELOG.md:11: bullet is 407 characters; keep each under 365")
}

func Test0018_4(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, changelogBulletsDoc("Added the readme.", "Fix: two.", "Docs: three."))
	sees(t, out, code, 1, "CHANGELOG.md:11: bullet must start with a semantic type (Feat:, Fix:, ... or feat(scope):, fix(scope):, ...)")
}

func Test0018_5(t *testing.T) {
	t.Parallel()
	out, code := changelogBullets(t, changelogBulletsDoc("Feat: one.", "fix(release)!: two.", "chore(ci): three.", "docs: four.", "Revert: five."))
	sees(t, out, code, 0, "All 1 checks passed")
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

func Test0018_8(t *testing.T) {
	t.Parallel()
	doc := changelogBulletsDoc("Docs: add the readme.", "Docs: add the changelog.", "Docs: add the license note.") +
		"\n## 1.1.0\n\n- Feat: one.\n- Feat: two.\n- Feat: three.\n- Fix: four.\n- Fix: five.\n- Test: six.\n"
	repo := example(t, "happyRepo", map[string]string{
		"CHANGELOG.md":    doc,
		".fitnessrc.json": `{"proseBudget": {"maxListItems": 5}}` + "\n",
	})
	out, code := fitness(t, repo, nil, "changelog-bullets")
	sees(t, out, code, 1, `section "1.1.0" has 6 bullets; keep releases to 1-5 bullets`)
	if strings.Contains(out, "2026.10.08.1400") {
		t.Errorf("the entry above the release must not fail:\n%s", out)
	}
}
