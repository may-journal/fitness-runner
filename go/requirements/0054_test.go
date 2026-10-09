package requirements

import "testing"

// releaseChangelog runs `fitness-install -- release-changelog` on happyRepo
// with files written over it.
func releaseChangelog(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "release-changelog")
}

// releaseChangelogNotes is a CHANGELOG with one release section for 1.2.3
// above the timestamped development entries.
const releaseChangelogNotes = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n" +
	"## [1.2.3] (2026-10-04)\n\n* Fix publishing.\n\n## Changes\n\n### 2026.10.08.1400\n\n- Docs: add the readme.\n"

// releaseChangelogRelease is a repo releasing 1.2.3 with its changelog.
func releaseChangelogRelease(changelog string) map[string]string {
	return map[string]string{
		".release-please-manifest.json": `{".":"1.2.3"}` + "\n",
		"version.txt":                   "1.2.3\n",
		"CHANGELOG.md":                  changelog,
	}
}

// releaseChangelogCase is one Given a test runs, with the text it must show.
type releaseChangelogCase struct {
	files map[string]string
	want  string
}

// releaseChangelogFails runs each case and asserts it fails showing its text.
func releaseChangelogFails(t *testing.T, cases []releaseChangelogCase) {
	t.Helper()
	for _, c := range cases {
		out, code := releaseChangelog(t, c.files)
		sees(t, out, code, 1, c.want)
	}
}

func Test0054_1(t *testing.T) {
	t.Parallel()
	out, code := releaseChangelog(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "release-changelog")
}

func Test0054_2(t *testing.T) {
	t.Parallel()
	out, code := releaseChangelog(t, map[string]string{".release-please-manifest.json": "{}\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0054_3(t *testing.T) {
	t.Parallel()
	out, code := releaseChangelog(t, releaseChangelogRelease(releaseChangelogNotes))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0054_4(t *testing.T) {
	t.Parallel()
	releaseChangelogFails(t, []releaseChangelogCase{
		{map[string]string{".release-please-manifest.json": "{\n"}, ".release-please-manifest.json is invalid JSON"},
		{map[string]string{".release-please-manifest.json": "null\n"}, ".release-please-manifest.json must be a JSON object"},
		{map[string]string{".release-please-manifest.json": `{"go":"1.2.3"}`}, `.release-please-manifest.json must contain a root "." release version`},
		{map[string]string{".release-please-manifest.json": `{".":"v1.2.3"}`}, `.release-please-manifest.json root version must be numeric major.minor.patch (got "v1.2.3")`},
	})
}

func Test0054_5(t *testing.T) {
	t.Parallel()
	files := releaseChangelogRelease(releaseChangelogNotes)
	files["version.txt"] = "1.2.4\n"
	out, code := releaseChangelog(t, files)
	sees(t, out, code, 1, `version.txt version "1.2.4" must match .release-please-manifest.json version "1.2.3"`)
}

func Test0054_6(t *testing.T) {
	t.Parallel()
	out, code := releaseChangelog(t, map[string]string{
		".release-please-manifest.json": `{".":"1.2.3"}` + "\n",
		"CHANGELOG.md":                  "",
	})
	sees(t, out, code, 1, "version.txt is missing; it must match release 1.2.3", "CHANGELOG.md is missing; add a release section for 1.2.3")
}

func Test0054_7(t *testing.T) {
	t.Parallel()
	releaseChangelogFails(t, []releaseChangelogCase{
		{releaseChangelogRelease("# Changelog\n\n## Changes\n\n- Work.\n"), "CHANGELOG.md must contain a `## 1.2.3` release section"},
		{releaseChangelogRelease("# Changelog\n\n## 1.2.3\n\nText only.\n"), "CHANGELOG.md release section 1.2.3 must contain at least one list item describing what ships"},
		{releaseChangelogRelease("# Changelog\n\n## 1.2.3\n\n- One.\n\n## 1.2.3 (today)\n\n- Two.\n"), "CHANGELOG.md contains duplicate release sections for 1.2.3"},
	})
}

func Test0054_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", releaseChangelogRelease(releaseChangelogNotes))
	write(t, repo, map[string]string{".release-please-manifest.json": `{".": "1.2.3"}` + "\n"})
	git(t, repo, "add", ".release-please-manifest.json")
	out, code := fitness(t, repo, nil, "release-changelog")
	sees(t, out, code, 1, ".release-please-manifest.json changed without CHANGELOG.md; add the release section reviewers will approve")
}
