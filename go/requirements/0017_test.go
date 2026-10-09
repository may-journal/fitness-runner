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

func Test0017_3(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelogWith("### 2026.10.08.1400", "### 2026.10.07")})
	sees(t, out, code, 1, `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026.10.07")`)
}

func Test0017_4(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, map[string]string{"CHANGELOG.md": changelogWith("### 2026.10.08.1400 hotfix", "### 2026.10.07.0900")})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0017_5(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(`{"version": "0.1.0-2026.10.08.1500"}`+"\n", ""))
	sees(t, out, code, 1, "package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)")
}

func Test0017_6(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(changelogVersion, `{"version": "0.1.0-2026.10.07.0900"}`+"\n"))
	sees(t, out, code, 1, "package-lock.json version must match package.json version")
}

func Test0017_7(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages("not json\n", ""))
	sees(t, out, code, 1, "package.json is invalid JSON")
	out, code = changelogRun(t, changelogPackages(changelogVersion, "not json\n"))
	sees(t, out, code, 1, "package-lock.json is invalid JSON")
}

func Test0017_8(t *testing.T) {
	t.Parallel()
	out, code := changelogRun(t, changelogPackages(changelogVersion, changelogVersion))
	sees(t, out, code, 0, "All 1 checks passed")
}
