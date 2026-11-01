package requirements

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// manifest0094 is a package.json with one runtime dependency, zod at zod,
// and the dev dependency parser at parser.
func manifest0094(zod, parser string) map[string]any {
	return map[string]any{
		"name": "example", "version": "1.0.0",
		"dependencies":    map[string]any{"zod": zod},
		"devDependencies": map[string]any{"@typescript-eslint/parser": parser, "vitest": "^3.2.0"},
	}
}

// lock0094 is the lockfile for manifest, with packages its installed
// packages beside the root entry.
func lock0094(manifest map[string]any, packages map[string]any) map[string]any {
	all := map[string]any{"": manifest}
	for name, entry := range packages {
		all[name] = entry
	}
	return map[string]any{"name": "example", "version": "1.0.0", "lockfileVersion": 3, "requires": true, "packages": all}
}

// installed0094 is a lockfile package entry at version, dev-only when dev.
func installed0094(version string, dev bool) map[string]any {
	if dev {
		return map[string]any{"version": version, "dev": true}
	}
	return map[string]any{"version": version}
}

// side0094 is one commit's package.json and lockfile.
type side0094 struct{ manifest, lock map[string]any }

// at0094 is a commit with zod and parser in package.json, installed as
// zodLock and parserLock, plus dev-only tiny 0.3.5 and extra packages.
func at0094(zod, parser, zodLock, parserLock string, extra map[string]any) side0094 {
	m := manifest0094(zod, parser)
	packages := map[string]any{
		"node_modules/zod":                       installed0094(zodLock, false),
		"node_modules/@typescript-eslint/parser": installed0094(parserLock, true),
		"node_modules/tiny":                      installed0094("0.3.5", true),
	}
	for name, entry := range extra {
		packages[name] = entry
	}
	return side0094{m, lock0094(m, packages)}
}

// base0094 is the base commit every bump starts from.
var base0094 = at0094("^3.23.0", "^8.71.2", "3.23.8", "8.71.2", nil)

// contents0094 is GitHub's answer reading file at commit ref as raw JSON.
func contents0094(t *testing.T, file, ref string, body map[string]any) response {
	t.Helper()
	raw, err := json.Marshal(body)
	mustDo(t, err)
	return response{Match: []string{"contents/" + file + "?ref=" + ref}, Stdout: string(raw)}
}

// bump0094 runs the dependency check on PR 23, titled title, at head abc123
// on base tip000 with merge base base000, listing files.
func bump0094(t *testing.T, title string, files response, head side0094) (replays, string, int, string) {
	t.Helper()
	view := response{Match: []string{"pr view 23 --json title"},
		Stdout: `{"title":"` + title + `","state":"OPEN","isDraft":false,"headRefOid":"abc123","baseRefOid":"tip000"}` + "\n"}
	mergeBase := response{Match: []string{"compare/tip000...abc123 --jq .merge_base_commit.sha"}, Stdout: "base000\n"}
	return run0093(t, "Check the dependency bump", []response{
		view, files, mergeBase,
		contents0094(t, "package.json", "base000", base0094.manifest), contents0094(t, "package-lock.json", "base000", base0094.lock),
		contents0094(t, "package.json", "abc123", head.manifest), contents0094(t, "package-lock.json", "abc123", head.lock),
	})
}

// both0094 lists package.json and its lockfile as PR 23's files.
var both0094 = files0093(`[{"filename":"package.json"},{"filename":"package-lock.json"}]`)

// title0094 is a dependency bump's title.
const title0094 = "build(deps): update @typescript-eslint/parser to 8.72.0"

func Test0094_1(t *testing.T) {
	t.Parallel()
	types := map[string]any{"node_modules/@typescript-eslint/types": installed0094("8.72.0", true)}
	_, out, code, written := bump0094(t, title0094, both0094, at0094("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", types))
	sees(t, out, code, 0)
	if written != "sha=abc123\n" {
		t.Errorf("a minor dev-dependency bump was not passed on: output %q\n%s", written, out)
	}
	lockOnly := files0093(`[{"filename":"package-lock.json"}]`)
	_, out, code, written = bump0094(t, "build(deps): update rollup to 4.64.9", lockOnly, at0094("^3.23.0", "^8.71.2", "3.23.8", "8.71.4", nil))
	sees(t, out, code, 0)
	if written != "sha=abc123\n" {
		t.Errorf("a lockfile-only dev bump was not passed on: output %q\n%s", written, out)
	}
	legacy := at0094("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", nil)
	legacy.lock["dependencies"] = map[string]any{"@typescript-eslint/parser": installed0094("8.72.0", true)}
	_, out, code, written = bump0094(t, title0094, both0094, legacy)
	sees(t, out, code, 0)
	if written != "sha=abc123\n" {
		t.Errorf("a bump in a lockfile with the legacy dependencies tree was not passed on: output %q\n%s", written, out)
	}
}

func Test0094_2(t *testing.T) {
	t.Parallel()
	for want, head := range map[string]side0094{
		"package.json changes more than devDependencies": at0094("^3.24.0", "^8.71.2", "3.24.1", "8.71.2", nil),
		"node_modules/zod is not a dev-only package":     at0094("^3.23.0", "^8.71.2", "3.23.9", "8.71.2", nil),
		"node_modules/@typescript-eslint/parser resolves outside the npm registry": at0094("^3.23.0", "^8.72.0", "3.23.8", "8.72.0",
			map[string]any{"node_modules/@typescript-eslint/parser": map[string]any{"version": "8.72.0", "dev": true, "resolved": "https://example.com/parser.tgz"}}),
		"node_modules/@typescript-eslint/parser changes without a new version": at0094("^3.23.0", "^8.71.2", "3.23.8", "8.71.2",
			map[string]any{"node_modules/@typescript-eslint/parser": map[string]any{"version": "8.71.2", "dev": true, "integrity": "sha512-other"}}),
	} {
		_, out, code, written := bump0094(t, title0094, both0094, head)
		sees(t, out, code, 0, "::notice::PR #23 is not a patch or minor dev-dependency bump, so a person approves it:", want)
		if written != "" {
			t.Errorf("a runtime dependency change was passed on (%s): output %q", want, written)
		}
	}
}

func Test0094_3(t *testing.T) {
	t.Parallel()
	_, out, code, written := bump0094(t, title0094, both0094, at0094("^3.23.0", "^9.0.0", "3.23.8", "9.0.0", nil))
	sees(t, out, code, 0, "@typescript-eslint/parser moves from ^8.71.2 to ^9.0.0, not a patch or minor bump",
		"node_modules/@typescript-eslint/parser moves from 8.71.2 to 9.0.0, not a patch or minor bump")
	if written != "" {
		t.Errorf("a major dev-dependency bump was passed on: output %q", written)
	}
	tiny := map[string]any{"node_modules/tiny": installed0094("0.4.0", true)}
	_, out, code, written = bump0094(t, title0094, both0094, at0094("^3.23.0", "^8.71.2", "3.23.8", "8.71.2", tiny))
	sees(t, out, code, 0, "node_modules/tiny moves from 0.3.5 to 0.4.0, not a patch or minor bump")
	if written != "" {
		t.Errorf("a 0.x minor bump, which semver counts as breaking, was passed on: output %q", written)
	}
}

func Test0094_4(t *testing.T) {
	t.Parallel()
	head := at0094("^3.23.0", "^8.72.0", "3.23.8", "8.72.0", nil)
	rp, out, code, written := bump0094(t, "feat: add a grid", both0094, head)
	sees(t, out, code, 0, "::notice::PR #23 is not titled build(deps), so a person approves it.")
	if written != "" || rp.called("gh", "contents/") {
		t.Errorf("a PR not titled build(deps) went on: output %q, gh calls %q", written, rp.calls("gh"))
	}
	workflow := files0093(`[{"filename":"package.json"},{"filename":".github/workflows/ci.yml"}]`)
	rp, out, code, written = bump0094(t, title0094, workflow, head)
	sees(t, out, code, 0, "::notice::PR #23 changes more than package.json and package-lock.json, so a person approves it: .github/workflows/ci.yml")
	if written != "" || rp.called("gh", "contents/") {
		t.Errorf("a bump that also changes a workflow went on: output %q, gh calls %q", written, rp.calls("gh"))
	}
}

func Test0094_5(t *testing.T) {
	t.Parallel()
	_, out, code, _ := run0093(t, "Check the approver token", nil, "KIND=dependencies", "TOKEN=token")
	sees(t, out, code, 1, "::error::Release approve got kind 'dependencies'; pass release or dependency.")
	// Actions fills in the input's default when a caller passes none.
	for kind, want := range map[string]string{"dependency": "Check the dependency bump", default0094(t): "Check the release PR"} {
		for _, step := range []string{"Check the dependency bump", "Check the release PR"} {
			if got := runs0093(t, step, map[string]string{"secrets.ready": "true", "inputs.kind": kind}); got != (step == want) {
				t.Errorf("with kind %q, step %q runs = %v", kind, step, got)
			}
		}
	}
}

// default0094 returns the default of the workflow's kind input.
func default0094(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(workflow0093)
	mustDo(t, err)
	_, kind, ok := strings.Cut(string(raw), "      kind:\n")
	if ok {
		_, kind, ok = strings.Cut(kind, "default: ")
	}
	if !ok {
		t.Fatalf("%s has no default for its kind input", workflow0093)
	}
	value, _, _ := strings.Cut(kind, "\n")
	return value
}
