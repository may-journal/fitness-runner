package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Fixtures use requirement 9999 so they never collide with this repo's own.
const (
	doc1     = "docs/requirements/9999-offline-sync.md"
	testFile = "sync/sync_test.go"
	source   = "Source: #12"
	chain    = "    - Given no network\n        - When I edit an entry\n            - Then it is saved locally\n"
)

// reqDoc1 builds requirement 9999 with the given Source line and
// Requirements body.
func reqDoc1(src, reqs string) string {
	return "# 9999 Offline Sync\n\n## Why\n\n<!-- one sentence -->\nI can keep writing offline\n\n## Measurement\n\n" +
		"users who save locally\n-\nusers who have lost connection\n\n" + src + "\n\n## Requirements\n\n" + reqs
}

// validDoc is requirement 9999 with one acceptance, 9999.1.
var validDoc = reqDoc1(source, "- 9999.1\n"+chain)

// testsNamed is a Go test file declaring one test per name.
func testsNamed(names ...string) string {
	var b strings.Builder
	b.WriteString("package sync\n\nimport \"testing\"\n")
	for _, n := range names {
		b.WriteString("\nfunc " + n + "(t *testing.T) {}\n")
	}
	return b.String()
}

// owned is a repo whose valid doc's one acceptance has its one test.
func owned() map[string]string {
	return map[string]string{doc1: validDoc, testFile: testsNamed("Test9999_1")}
}

// with returns base plus overrides.
func with(base map[string]string, overrides map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// fixture is one repo layout, the run's changed files ("" for a full run),
// and the error it must produce ("" for a pass).
type fixture struct {
	name    string
	files   map[string]string
	changed string
	want    string
}

// judge runs the check on each fixture in a fresh directory.
func judge(t *testing.T, cases []fixture) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, c.files)
			expect(t, runIn(t, root, c.changed), c.want)
		})
	}
}

// runIn runs the check on root with the given changed files.
func runIn(t *testing.T, root, changed string) []string {
	t.Helper()
	t.Setenv("FITNESS_CHANGED_FILES", changed)
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok != (len(res.Errors) == 0) {
		t.Fatalf("ok=%v disagrees with errors %v", res.Ok, res.Errors)
	}
	return res.Errors
}

// expect asserts the errors contain want, or are empty when want is "".
func expect(t *testing.T, errs []string, want string) {
	t.Helper()
	joined := strings.Join(errs, "\n")
	if want == "" && len(errs) > 0 {
		t.Fatalf("want pass, got:\n%s", joined)
	}
	if !strings.Contains(joined, want) {
		t.Fatalf("want an error containing %q, got:\n%s", want, joined)
	}
}

// docWith is the owned repo with requirement 9999's Requirements replaced.
func docWith(reqs string) map[string]string {
	return with(owned(), map[string]string{doc1: reqDoc1(source, reqs)})
}

// skips asserts the check passes on files with nothing checked.
func skips(t *testing.T, files map[string]string, changed string) {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, files)
	t.Setenv("FITNESS_CHANGED_FILES", changed)
	if res, err := run(root, nil); err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("want a clean pass with nothing checked, got %+v, %v", res, err)
	}
}

func Test0005_1(t *testing.T) {
	judge(t, []fixture{
		{"owned acceptance passes", owned(), "", ""},
		{"unowned acceptance fails", with(owned(), map[string]string{testFile: testsNamed("TestOther")}), "", "9999.1 has no test; name exactly one test Test9999_1"},
	})
}

func Test0005_2(t *testing.T) {
	judge(t, []fixture{
		{"two packages owning one acceptance fail", with(owned(), map[string]string{"other/x_test.go": testsNamed("Test9999_1")}), "", "9999.1 is owned by 2 tests"},
	})
}

func Test0005_3(t *testing.T) {
	judge(t, []fixture{
		{"a test for a removed acceptance fails", with(owned(), map[string]string{testFile: testsNamed("Test9999_1", "Test9999_2")}), "", "Test9999_2 names 9999.2, which no requirement defines"},
	})
}

func Test0005_4(t *testing.T) {
	stray := with(owned(), map[string]string{"other/y_test.go": testsNamed("TestHelperParses")})
	judge(t, []fixture{
		{"a full run fails a test that proves nothing", stray, "", "TestHelperParses proves no requirement"},
		{"a scoped run fails it when its file changed", stray, "other/y_test.go", "TestHelperParses proves no requirement"},
		{"a scoped run leaves unchanged files to their own PR", stray, testFile, ""},
		{"TestMain and helpers are not tests", with(owned(), map[string]string{"other/y_test.go": "package y\n\nimport \"testing\"\n\nfunc TestMain(m *testing.M) {}\n\nfunc testsHelper(t *testing.T) {}\n"}), "", ""},
	})
}

func Test0005_5(t *testing.T) {
	judge(t, []fixture{
		{"a Go repo without docs fails", map[string]string{testFile: testsNamed("TestX")}, "", "has no requirement docs"},
		{"even when a scoped run touches neither", map[string]string{testFile: testsNamed("TestX")}, "README.md", "has no requirement docs"},
	})
}

func Test0005_6(t *testing.T) {
	skips(t, map[string]string{"README.md": "# App\n", "App/AppTests.swift": "@Test func a() {}\n"}, "")
}

func Test0005_7(t *testing.T) {
	skips(t, with(owned(), map[string]string{testFile: testsNamed("TestUnowned")}), "README.md")
}

// gitRepo commits each step's files in order and returns the root.
func gitRepo(t *testing.T, steps ...map[string]string) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q")
	for _, files := range steps {
		writeFiles(t, root, files)
		git(t, root, "add", "-A")
		git(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "step")
	}
	return root
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func Test0005_8(t *testing.T) {
	two := with(owned(), map[string]string{doc1: reqDoc1(source, "- 9999.1\n"+chain+"- 9999.2\n"+chain), testFile: testsNamed("Test9999_1", "Test9999_2")})
	root := gitRepo(t, two, owned())
	writeFiles(t, root, two)
	expect(t, runIn(t, root, ""), "9999.2 was deleted before")

	edited := gitRepo(t, owned(), map[string]string{doc1: strings.Replace(validDoc, "no network", "airplane mode", 1)})
	expect(t, runIn(t, edited, ""), "")
}

func Test0006_1(t *testing.T) {
	judge(t, []fixture{
		{"a matching name and title pass", owned(), "", ""},
		{"a file name without an ID fails", map[string]string{"docs/requirements/offline.md": validDoc, testFile: testsNamed("Test9999_1")}, "", "name the file NNNN-kebab-title.md"},
		{"a title ID differing from the file name fails", map[string]string{"docs/requirements/9998-offline.md": validDoc, testFile: testsNamed("Test9999_1")}, "", "the title must be `# 9998 Title`"},
	})
}

func Test0006_2(t *testing.T) {
	judge(t, []fixture{
		{"two docs sharing an ID fail", with(owned(), map[string]string{"docs/requirements/9999-other.md": validDoc}), "", "requirement 9999 is used by"},
	})
}

func Test0006_3(t *testing.T) {
	judge(t, []fixture{
		{"an extra section fails", with(owned(), map[string]string{doc1: validDoc + "\n## Out of scope\n\n- x\n"}), "", "sections must be exactly"},
		{"an empty Why fails", with(owned(), map[string]string{doc1: strings.Replace(validDoc, "I can keep writing offline\n", "", 1)}), "", "section `## Why` is empty"},
		{"a template beside the docs is ignored", with(owned(), map[string]string{"docs/requirements/template.md": "# NNNN Title\n"}), "", ""},
	})
}

func Test0006_4(t *testing.T) {
	judge(t, []fixture{
		{"no dash line fails", with(owned(), map[string]string{doc1: strings.Replace(validDoc, "\n-\n", "\nover\n", 1)}), "", "Measurement needs four lines"},
		{"no Source line fails", with(owned(), map[string]string{doc1: reqDoc1("From: #12", "- 9999.1\n"+chain)}), "", "must start with `Source:`"},
	})
}

func Test0006_5(t *testing.T) {
	src := func(s string) map[string]string {
		return with(owned(), map[string]string{doc1: reqDoc1(s, "- 9999.1\n"+chain)})
	}
	judge(t, []fixture{
		{"an issue passes", src("Source: #12"), "", ""},
		{"an existing code line passes", src("Source: `sync/sync_test.go:3`"), "", ""},
		{"code and an issue together fail", src("Source: `sync/sync_test.go:1` #12"), "", "either one line of code"},
		{"a missing file fails", src("Source: `nope.go:1`"), "", "not a file in this repo"},
		{"a line past the end fails", src("Source: `sync/sync_test.go:90`"), "", "past its end"},
	})
}

func Test0006_6(t *testing.T) {
	judge(t, []fixture{
		{"a second When fails", docWith("- 9999.1\n" + chain + "        - When I relaunch\n"), "", "needs exactly one Given, one When, and one Then"},
	})
}

func Test0006_7(t *testing.T) {
	judge(t, []fixture{
		{"keywords out of order fail", docWith("- 9999.1\n    - When I edit\n        - Given no network\n            - Then saved\n"), "", "must start with `Given `"},
		{"a flat chain fails", docWith("- 9999.1\n    - Given no network\n    - When I edit\n    - Then saved\n"), "", "must be nested one level under"},
	})
}

func Test0006_8(t *testing.T) {
	judge(t, []fixture{
		{"another requirement's prefix fails", docWith("- 9998.1\n" + chain), "", "top-level items must be acceptance IDs `9999.N`"},
		{"a repeated ID fails", docWith("- 9999.1\n" + chain + "- 9999.1\n" + chain), "", "9999.1 appears twice"},
		{"prose in Requirements fails", docWith("Some prose\n- 9999.1\n" + chain), "", "Requirements holds only list items"},
	})
}
