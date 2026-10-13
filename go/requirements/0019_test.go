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

// changelogEntry is happyRepo's changelog with a new section on top.
func changelogEntry(stamp, item string) string {
	return changelogOnTop(changelogBase, stamp, item)
}

// changelogOnTop puts a dated section with one bullet above changelog's
// first section.
func changelogOnTop(changelog, stamp, item string) string {
	return strings.Replace(changelog, "## Changes\n\n", "## Changes\n\n### "+stamp+"\n\n- "+item+"\n\n", 1)
}

// changelogNow is the stamp for a section written minutes ago.
func changelogNow(minutes int) string {
	return time.Now().Add(-time.Duration(minutes) * time.Minute).Format("2006.01.02.1504")
}

// changelogFeature is a staged source change the entry should mention.
const changelogFeature = "Added runner feature for validation.\n"

func Test0019_1(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(2), "Feat: added runner feature for validation.\n\n### 2026.01.02.0900\n\n- Docs: an older entry."),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0019_10(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{"src.txt": changelogFeature})
	sees(t, out, code, 1, "Stage a changelog/<name>.md entry file, or a CHANGELOG.md section, that mentions your staged changes")
	out, code = changelogStaged(t, map[string]string{"src.txt": changelogFeature, "CHANGELOG.md": ""})
	sees(t, out, code, 1, "CHANGELOG.md missing; add it and mention your staged changes")
}

func Test0019_11(t *testing.T) {
	t.Parallel()
	files := withEntries(map[string]string{"runner-feature": entryFileText(changelogNow(1), "Feat: added runner feature for validation.")})
	files["src.txt"] = changelogFeature
	out, code := changelogStaged(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0019_3(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(0), "Fix: minor tweak."),
	})
	sees(t, out, code, 1,
		"CHANGELOG.md additions should mention at least 3 words from your staged changes (found 0: )",
		"e.g. use words like: added, runner, feature, for, validation")
}

func Test0019_4(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(10), "Feat: added runner feature for validation."),
	})
	sees(t, out, code, 1, "CHANGELOG.md new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time (expected ### ")
}

func Test0019_5(t *testing.T) {
	t.Parallel()
	out, code := changelogStaged(t, map[string]string{
		"package.json": `{"name": "app", "private": true, "version": "1.0.0-2026.10.08.1500"}` + "\n",
		"src.txt":      changelogFeature,
		"CHANGELOG.md": changelogEntry(changelogNow(0), "Feat: added runner feature for validation."),
	})
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

func Test0019_12(t *testing.T) {
	t.Parallel()
	entry := changelogEntry(changelogFirst, "Feat: added runner feature for validation.")
	repo := changelogBranch(t, nil, map[string]string{"CHANGELOG.md": entry})
	stamp := changelogNow(0)
	out, code := changelogSecond(t, repo, map[string]string{"CHANGELOG.md": changelogOnTop(entry, stamp, "Feat: added runner option for validation.")})
	sees(t, out, code, 1, "this branch already has its entry ### "+changelogFirst,
		"instead of a new entry: ### "+stamp)
	first := entryFileText(changelogFirst, "Feat: added runner feature for validation.")
	repo = changelogBranch(t, nil, map[string]string{"changelog/runner-feature.md": first})
	second := map[string]string{"CHANGELOG.md": changelogEntry(stamp, "Feat: added runner option for validation.")}
	out, code = changelogSecond(t, repo, second)
	sees(t, out, code, 1, "this branch already has its entry changelog/runner-feature.md",
		"instead of a new entry: ### "+stamp)
}
