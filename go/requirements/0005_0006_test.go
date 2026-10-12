package requirements

import (
	"strings"
	"testing"
)

// judged runs the full suite on happyRepo plus files and returns the output.
func judged(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil)
}

func Test0005_6(t *testing.T) {
	t.Parallel()
	out, _ := judged(t, nil)
	passedWithNoFiles(t, out, "requirements")
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
	sees(t, out, code, 1, pairedTest+" changed without its requirement doc "+pairedDoc)
	out, code = prRun(t, pairedRepo(), map[string]string{pairedDoc: requirementDoc("0001.1") + "\n"}, "", "docs(requirements): reword 0001")
	sees(t, out, code, 1, pairedDoc+" changed without its requirement test "+pairedTest)
}

func Test0005_10(t *testing.T) {
	t.Parallel()
	doc := "docs/requirements/0002-adds-again.md"
	test := map[string]string{twoTest: edited(twoRepo()[twoTest])}
	out, code := stagedRun(t, twoRepo(), test)
	sees(t, out, code, 1, twoTest+" changed without its requirement doc "+pairedDoc+" or "+doc)
	out, code = stagedRun(t, twoRepo(), with(test, map[string]string{doc: twoRepo()[doc] + "\n"}))
	sees(t, out, code, 0)
}

func Test0005_11(t *testing.T) {
	t.Parallel()
	out, code := stagedRun(t, pairedRepo(), map[string]string{"reqs/tools_test.go": edited(pairedRepo()["reqs/tools_test.go"])})
	sees(t, out, code, 0)
}

func Test0005_12(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, pairedRepo())
	write(t, repo, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "test(reqs): check sums twice", "-m", exempt)
	sees(t, out, code, 0, "test(reqs): check sums twice")
}

func Test0005_13(t *testing.T) {
	t.Parallel()
	change := map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))}
	out, code := prRun(t, pairedRepo(), change, "Rewords one test.", "test(reqs): reword", exempt)
	sees(t, out, code, 1, `"`+exempt+`" is not listed in the pull request body`)
	out, code = prRun(t, pairedRepo(), change, "Rewords one test.\n\n"+exempt+"\n", "test(reqs): reword", exempt)
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
	out, code := fitness(t, mergedBase(t), prEnv(t, ""), "requirements")
	sees(t, out, code, 0)
}

// docOf is goRepo with requirement 0001 replaced by body at path.
func docOf(path, body string) map[string]string {
	return with(goRepo(), map[string]string{"docs/requirements/0001-adds-numbers.md": "", path: body})
}

// reqs is goRepo with requirement 0001's Requirements replaced.
func reqs(body string) map[string]string {
	return docOf("docs/requirements/0001-adds-numbers.md", docWith("Source: `adder.go:5`", body))
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
	sees(t, out, code, 1, "Measurement needs four lines")
}

func Test0006_5(t *testing.T) {
	t.Parallel()
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", docWith("Source: `adder.go:5` #12", acceptances("0001.1"))))
	sees(t, out, code, 1, "Source must cite either one existing line of code")
}

func Test0006_6(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1")+"        - When I add them again\n"))
	sees(t, out, code, 1, "needs exactly one Given, one When, and one Then")
}

func Test0006_7(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs("- 0001.1\n    - When I add them\n        - Given two numbers\n            - Then I get their sum\n"))
	sees(t, out, code, 1, "each acceptance is one Given, then one When, then one Then, each nested under the line above")
}

func Test0006_8(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1", "0001.1")))
	sees(t, out, code, 1, "0001.1 appears twice")
}
