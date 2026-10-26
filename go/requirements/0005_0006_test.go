package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// judged runs the full suite on happyRepo plus files and returns the output.
func judged(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil)
}

func Test0005_8(t *testing.T) {
	t.Parallel()
	two := with(goRepo(), map[string]string{
		"adder_test.go":                          testFile("adder", "Test0001_1", "Test0001_2"),
		"docs/requirements/0001-adds-numbers.md": requirementDoc("0001.1", "0001.2"),
	})
	repo := example(t, "happyRepo", two)
	commit(t, repo, goRepo(), "docs(requirements): drop 0001.2")
	write(t, repo, two)
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "0001.2 was deleted before")
}

func Test0005_9(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, pairedRepo())
	write(t, repo, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "test(reqs): check sums twice")
	sees(t, out, code, 1, pairedTest+" changed without its requirement doc "+pairedDoc+"; change both together")
	out, code = prRun(t, pairedRepo(), map[string]string{pairedDoc: reworded(requirementDoc("0001.1"))}, "docs(requirements): reword 0001")
	sees(t, out, code, 1, pairedDoc+" changed without its requirement test "+pairedTest+"; change both together")
}

func Test0005_10(t *testing.T) {
	t.Parallel()
	doc := "docs/requirements/0002-adds-again.md"
	test := map[string]string{twoTest: edited(twoRepo()[twoTest])}
	out, code := stagedRun(t, twoRepo(), test)
	sees(t, out, code, 1, twoTest+" changed without its requirement doc "+pairedDoc+" or "+doc)
	out, code = stagedRun(t, twoRepo(), with(test, map[string]string{doc: reworded(twoRepo()[doc])}))
	sees(t, out, code, 0)
}

func Test0005_11(t *testing.T) {
	t.Parallel()
	out, code := stagedRun(t, pairedRepo(), map[string]string{"reqs/tools_test.go": edited(pairedRepo()["reqs/tools_test.go"])})
	sees(t, out, code, 0)
}

func Test0005_14(t *testing.T) {
	t.Parallel()
	keep := []string{"--no-edit"}
	for _, r := range [][][]string{
		{{"merge", "--no-edit", "side"}, keep, {"--amend", "-m", "chore(reqs): merge the reword"}},
		{{"cherry-pick", "side"}, keep},
		{{"revert", "--no-edit", "side"}, {"-m", "revert(reqs): undo the reword"}},
	} {
		out, code := replayed(t, r[0], r[1:]...)
		sees(t, out, code, 0)
	}
	out, code := fitness(t, mergedBase(t), prEnv, "requirements")
	sees(t, out, code, 0)
}

func Test0005_15(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, pairedRepo())
	write(t, repo, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))})
	userSteps(t, repo, env, []string{"add", "-A"}, []string{"commit", "-q", "--no-verify", "-m", "test(reqs): check sums twice"})
	write(t, repo, map[string]string{pairedDoc: reworded(requirementDoc("0001.1"))})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "--amend", "-m", "test(reqs): check sums twice")
	sees(t, out, code, 0, "test(reqs): check sums twice")
}

func Test0005_16(t *testing.T) {
	t.Parallel()
	failing := with(swiftRepo(), map[string]string{swiftTest: xcTestFile("test0001_1")})
	fixed := map[string]string{swiftTest: swiftTestingFile("test0001_1")}
	out, code := stagedRun(t, failing, fixed)
	sees(t, out, code, 0)
	out, code = prRun(t, failing, fixed, "test(adder): move to Swift Testing")
	sees(t, out, code, 0)
	repo, env := hooked(t, with(failing, map[string]string{".fitnessrc.json": `{"policy": "external", "checks": ["requirements"]}` + "\n"}))
	write(t, repo, fixed)
	userSteps(t, repo, env, []string{"add", "-A"}, []string{"commit", "-q", "--no-verify", "-m", "test(adder): move to Swift Testing"})
	write(t, repo, map[string]string{swiftTest: edited(fixed[swiftTest])})
	user(t, repo, env, "add", "-A")
	out, code = user(t, repo, env, "commit", "--amend", "-m", "test(adder): move to Swift Testing")
	sees(t, out, code, 0, "test(adder): move to Swift Testing")
	out, code = stagedRun(t, swiftRepo(), map[string]string{swiftTest: edited(swiftTestingFile("test0001_1"))})
	sees(t, out, code, 1, swiftTest+" changed without its requirement doc "+pairedDoc+"; change both together")
}

// unformatted is requirementDoc as it read before a formatter rewrote it:
// other quotes in its front matter, its Why wrapped across two lines, the
// Measurement's - with no blank lines, and four-space nesting.
var unformatted = strings.NewReplacer(
	"two numbers without", "two numbers\nwithout",
	`['../../.fitnessrc.json']`, `["../../.fitnessrc.json"]`,
	"\n\n-\n\n", "\n-\n",
	"\n  - Given", "\n    - Given",
	"\n    - When", "\n        - When",
	"\n      - Then", "\n            - Then",
)

func Test0005_17(t *testing.T) {
	t.Parallel()
	given := with(pairedRepo(), map[string]string{pairedDoc: unformatted.Replace(requirementDoc("0001.1"))})
	formatted := map[string]string{pairedDoc: requirementDoc("0001.1")}
	out, code := stagedRun(t, given, formatted)
	sees(t, out, code, 0)
	out, code = prRun(t, given, formatted, "style(docs): format requirement docs with Biome")
	sees(t, out, code, 0)
	out, code = stagedRun(t, given, with(formatted, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))}))
	sees(t, out, code, 0)
	sourced := with(pairedRepo(), map[string]string{pairedDoc: withSource(requirementDoc("0001.1"), "`adder.go:5`")})
	for _, change := range []string{withSource(requirementDoc("0001.1"), "`adder.go:6`"), requirementDoc("0001.1")} {
		out, code = stagedRun(t, sourced, map[string]string{pairedDoc: change})
		sees(t, out, code, 0)
		out, code = prRun(t, sourced, map[string]string{pairedDoc: change}, "docs(requirements): drop the Source line")
		sees(t, out, code, 0)
	}
	out, code = stagedRun(t, sourced, map[string]string{pairedDoc: strings.Replace(requirementDoc("0001.1"), "sums asked for", "sums tried", 1)})
	sees(t, out, code, 1, pairedDoc+" changed without its requirement test")
}

// withSource is doc with the Source line older docs end their Measurement
// with, citing cite. It sits on line 19.
func withSource(doc, cite string) string {
	return strings.Replace(doc, "## Requirements", "Source: "+cite+"\n\n## Requirements", 1)
}

// docOf is goRepo with requirement 0001 replaced by body at path.
func docOf(path, body string) map[string]string {
	return with(goRepo(), map[string]string{"docs/requirements/0001-adds-numbers.md": "", path: body})
}

// reqs is goRepo with requirement 0001's Requirements replaced.
func reqs(body string) map[string]string {
	return docOf("docs/requirements/0001-adds-numbers.md", docWith(body))
}

func Test0006_1(t *testing.T) {
	t.Parallel()
	out, code := judged(t, docOf("docs/requirements/0002-adds-numbers.md", requirementDoc("0001.1")))
	sees(t, out, code, 1, "name the file NNNN-kebab-title.md and title it `# NNNN Title` with the same ID")
}

func Test0006_2(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"docs/requirements/0001-other.md": requirementDoc("0001.1")}))
	sees(t, out, code, 1, "requirement 0001 is used by")
}

func Test0006_3(t *testing.T) {
	t.Parallel()
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", requirementDoc("0001.1")+"\n## Out of scope\n\n- Subtraction.\n"))
	sees(t, out, code, 1, "sections must be exactly ## Why, ## Measurement, ## Requirements")
}

func Test0006_4(t *testing.T) {
	t.Parallel()
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", strings.Replace(requirementDoc("0001.1"), "\n-\n", "\nover\n", 1)))
	sees(t, out, code, 1, "Measurement needs three lines: numerator, `-`, denominator")
}

func Test0006_11(t *testing.T) {
	t.Parallel()
	doc := requirementDoc("0001.1")
	sourced := withSource(doc, "`gone.go:99`")
	repo := example(t, "happyRepo", with(pairedRepo(), map[string]string{pairedDoc: sourced}))
	out, code := fitness(t, repo, nil, "requirements")
	sees(t, out, code, 0, pairedDoc+":15: delete this `Source:` line, or run `fitness requirements --fix`")
	out, code = fitness(t, repo, nil, "requirements", "--fix")
	sees(t, out, code, 0)
	if got, err := os.ReadFile(filepath.Join(repo, pairedDoc)); err != nil || string(got) != doc {
		t.Errorf("--fix must leave the doc without its Source line, got %v:\n%s", err, got)
	}
	git(t, repo, "add", "-A")
	out, code = fitness(t, repo, nil, "requirements")
	sees(t, out, code, 0)
	if strings.Contains(out, "Source") {
		t.Errorf("a doc without a Source line gets no note, got:\n%s", out)
	}
}

func Test0006_8(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1", "0001.1")))
	sees(t, out, code, 1, "0001.1 appears twice")
}

func Test0006_9(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1")+"        - When I add them again\n"))
	sees(t, out, code, 1, "needs exactly one Given, one When, and one Then")
	out, code = judged(t, reqs("- 0001.1\n    - When I add them\n        - Given two numbers\n            - Then I get their sum\n"))
	sees(t, out, code, 1, "each acceptance is one Given, then one When, then one Then, each nested under the line above")
}

func Test0006_10(t *testing.T) {
	t.Parallel()
	other := "docs/requirements/0002-adds-again.md"
	adds := "- 0001.1\n    - Given Writer has two whole numbers\n        - When I add them\n            - Then I get their sum\n"
	again := "- 0002.1\n    - Given Writer has two whole numbers that carry\n        - When I add them\n            - Then I get their sum\n"
	out, code := judged(t, with(goRepo(), map[string]string{
		"adder_test.go": testFile("adder", "Test0001_1", "Test0002_1"),
		pairedDoc:       docWith(adds),
		other:           strings.Replace(docWith(again), "0001 Adds Numbers", "0002 Adds Again", 1),
	}))
	sees(t, out, code, 1, pairedDoc+" and "+other+` both cover "When I add them"`, "0004-one-requirement-doc-per-area")
}
