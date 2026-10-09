package requirements

import (
	"testing"
	"time"
)

// changelogStaged commits happyRepo, stages staged over it, and runs
// `fitness-install -- changelog-updated` as a pre-commit hook would.
func changelogStaged(t *testing.T, staged map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	write(t, repo, staged)
	git(t, repo, "add", "-A")
	return fitness(t, repo, nil, "changelog-updated")
}

// changelogEntry is happyRepo's changelog with a new section on top.
func changelogEntry(stamp, item string) string {
	return "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n### " +
		stamp + "\n\n- " + item + "\n\n### 2026.10.08.1400\n\n- Docs: add the readme.\n" +
		"- Docs: add the changelog.\n- Docs: add the license note.\n"
}

// changelogNow is the stamp for a section written minutes ago.
func changelogNow(minutes int) string {
	return time.Now().Add(-time.Duration(minutes) * time.Minute).Format("2006.01.02.1504")
}

// changelogFeature is a staged source change the entry should mention.
const changelogFeature = "Added runner feature for validation.\n"

func Test0019_1(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(2), "Feat: added runner feature for validation.\n\n### 2026.01.02.0900\n\n- Docs: an older entry."),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0019_2(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{"src.txt": changelogFeature})
	sees(t, out, code, 1, "Stage CHANGELOG.md and add an entry that mentions your staged changes")
}

func Test0019_3(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(0), "Fix: minor tweak."),
	})
	sees(t, out, code, 1,
		"CHANGELOG.md additions should mention at least 3 words from your staged changes (found 0: )",
		"e.g. use words like: added, runner, feature, for, validation")
}

func Test0019_4(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(10), "Feat: added runner feature for validation."),
	})
	sees(t, out, code, 1, "CHANGELOG.md new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time (expected ### ")
}

func Test0019_5(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{
		"package.json": `{"name": "app", "private": true, "version": "1.0.0-2026.10.08.1500"}` + "\n",
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(0), "Feat: added runner feature for validation."),
	})
	sees(t, out, code, 1, "not a guessed time (expected ### 2026.10.08.1500)")
}

func Test0019_6(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{"src.txt": "base\n"})
	git(t, repo, "checkout", "-q", "-b", "side")
	commit(t, repo, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry("2026.07.07.0850", "Feat: added runner feature for validation."),
	}, "feat: add runner")
	git(t, repo, "checkout", "-q", "-")
	commit(t, repo, map[string]string{"README.md": readme + "\nMore.\n"}, "docs: more")
	git(t, repo, "merge", "-q", "--no-commit", "--no-ff", "side")
	out, code := fitness(t, repo, nil, "changelog-updated")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0019_7(t *testing.T) {
	out, code := changelogStaged(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "changelog-updated")
}

func Test0019_8(t *testing.T) {
	out, code := changelogStaged(t, map[string]string{"src.txt": changelogFeature, "CHANGELOG.md": ""})
	sees(t, out, code, 1, "CHANGELOG.md missing; add it and mention your staged changes")
}
