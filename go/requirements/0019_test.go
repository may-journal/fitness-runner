package requirements

import (
	"strings"
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

// changelogBase is happyRepo's changelog.
const changelogBase = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n" +
	"### 2026.10.08.1400\n\n- Docs: add the readme.\n- Docs: add the changelog.\n- Docs: add the license note.\n"

// changelogEntry is happyRepo's changelog with a new section, holding one
// bullet item, on top.
func changelogEntry(stamp, item string) string {
	return strings.Replace(changelogBase, "## Changes\n\n", "## Changes\n\n### "+stamp+"\n\n- "+item+"\n\n", 1)
}

// changelogNow is the stamp for a section written minutes ago.
func changelogNow(minutes int) string {
	return time.Now().Add(-time.Duration(minutes) * time.Minute).Format("2006.01.02.1504")
}

// changelogFeature is a staged source change the entry should mention.
const changelogFeature = "Added runner feature for validation.\n"

// changelogStagedEntry is happyRepo's staged feature with its entry file
// under stamp, its first bullet item.
func changelogStagedEntry(stamp, item string) map[string]string {
	files := withEntries(map[string]string{"runner-feature": entryFileText(stamp, item)})
	files["src.txt"] = changelogFeature
	return files
}

func Test0019_13(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{"src.txt": changelogFeature})
	sees(t, out, code, 1, "Stage a changelog/<name>.md entry file that mentions your staged changes")
	files := changelogStagedEntry(changelogNow(1), "Feat: added runner feature for validation.")
	files["CHANGELOG.md"] = ""
	out, code = changelogStaged(t, files)
	sees(t, out, code, 1, "CHANGELOG.md missing; add it, and record your staged changes in a changelog/<name>.md entry file")
}

func Test0019_16(t *testing.T) {
	t.Parallel()
	files := changelogStagedEntry(changelogNow(1), "Feat: added runner feature for validation.")
	out, code := changelogStaged(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	repo := example(t, "happyRepo", map[string]string{"CHANGELOG.md": ""})
	files["CHANGELOG.md"] = changelogBase
	write(t, repo, files)
	git(t, repo, "add", "-A")
	out, code = fitness(t, repo, nil, "changelog-updated")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0019_15(t *testing.T) {
	t.Parallel()
	want := "CHANGELOG.md is staged; only the release PR writes it, so record this change in a changelog/<name>.md entry file instead"
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(0), "Feat: added runner feature for validation."),
	})
	sees(t, out, code, 1, want)
	files := changelogStagedEntry(changelogNow(1), "Feat: added runner feature for validation.")
	files["CHANGELOG.md"] = strings.Replace(changelogBase, "- Docs: add the readme.", "- Docs: add the readme file.", 1)
	out, code = changelogStaged(t, files)
	sees(t, out, code, 1, want)
}

func Test0019_3(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, changelogStagedEntry(changelogNow(0), "Fix: minor tweak."))
	sees(t, out, code, 1,
		"Changelog entry file additions should mention at least 3 words from your staged changes (found 0: )",
		"e.g. use words like: added, runner, feature, for, validation")
}

func Test0019_4(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, changelogStagedEntry(changelogNow(10), "Feat: added runner feature for validation."))
	sees(t, out, code, 1, "Changelog entry file new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time (expected ### ")
}

func Test0019_5(t *testing.T) {
	t.Parallel()
	files := changelogStagedEntry(changelogNow(0), "Feat: added runner feature for validation.")
	files["package.json"] = `{"name": "app", "private": true, "version": "1.0.0-2026.10.08.1500"}` + "\n"
	out, code := changelogStaged(t, files)
	sees(t, out, code, 1, "not a guessed time (expected ### 2026.10.08.1500)")
}

func Test0019_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"src.txt": "base\n"})
	git(t, repo, "checkout", "-q", "-b", "side")
	commit(t, repo, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry("2026.07.07.0850", "Feat: added runner feature for validation."),
	}, "feat: add runner")
	git(t, repo, "checkout", "-q", "-")
	commit(t, repo, map[string]string{"README.md": readme + "\nMore.\n"}, "docs: more")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-commit", "--no-ff", "side")
	out, code := fitness(t, repo, nil, "changelog-updated")
	sees(t, out, code, 0, "All 1 checks passed")
}

// changelogBranch is happyRepo, with files over it, on a branch off
// origin/main whose first commit added first for the staged feature.
func changelogBranch(t *testing.T, files, first map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", files)
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "work")
	first["src.txt"] = changelogFeature
	commit(t, repo, first, "feat: add runner")
	return repo
}

// changelogFirst is the stamp of the branch's first entry.
const changelogFirst = "2026.01.02.0900"

// changelogSecond stages a second change in repo, with entry, and runs
// changelog-updated on it.
func changelogSecond(t *testing.T, repo string, entry map[string]string) (string, int) {
	t.Helper()
	entry["src.txt"] = changelogFeature + "Added runner option for validation.\n"
	write(t, repo, entry)
	git(t, repo, "add", "-A")
	return fitness(t, repo, nil, "changelog-updated")
}

func Test0019_14(t *testing.T) {
	t.Parallel()
	first := entryFileText(changelogFirst, "Feat: added runner feature for validation.")
	repo := changelogBranch(t, nil, map[string]string{"changelog/runner-feature.md": first})
	second := withEntries(map[string]string{"runner-option": entryFileText(changelogNow(0), "Feat: added runner option for validation.")})
	out, code := changelogSecond(t, repo, second)
	sees(t, out, code, 1, "this branch already has its entry changelog/runner-feature.md",
		"instead of a new entry: changelog/runner-option.md")
}
