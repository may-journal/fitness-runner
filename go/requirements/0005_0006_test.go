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

func Test0005_1(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
}

func Test0005_2(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
}

func Test0005_3(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines")
}

func Test0005_4(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	sees(t, out, code, 1, "TestAddsAgain proves no requirement")
}

func Test0005_5(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"docs/requirements/0001-adds-numbers.md": ""}))
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
}

func Test0005_6(t *testing.T) {
	out, _ := judged(t, nil)
	passedWithNoFiles(t, out, "requirements")
}

func Test0005_7(t *testing.T) {
	repo := example(t, "happyRepo", with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	write(t, repo, map[string]string{"README.md": readme + "\nOne more short line.\n"})
	git(t, repo, "add", "README.md")
	out, _ := fitness(t, repo, nil, "--policy=external")
	passedWithNoFiles(t, out, "requirements")
}

func Test0005_8(t *testing.T) {
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

// docOf is goRepo with requirement 0001 replaced by body at path.
func docOf(path, body string) map[string]string {
	return with(goRepo(), map[string]string{"docs/requirements/0001-adds-numbers.md": "", path: body})
}

// reqs is goRepo with requirement 0001's Requirements replaced.
func reqs(body string) map[string]string {
	return docOf("docs/requirements/0001-adds-numbers.md", docWith("Source: `adder.go:5`", body))
}

func Test0006_1(t *testing.T) {
	out, code := judged(t, docOf("docs/requirements/0002-adds-numbers.md", requirementDoc("0001.1")))
	sees(t, out, code, 1, "name the file NNNN-kebab-title.md and title it `# NNNN Title` with the same ID")
}

func Test0006_2(t *testing.T) {
	out, code := judged(t, with(goRepo(), map[string]string{"docs/requirements/0001-other.md": requirementDoc("0001.1")}))
	sees(t, out, code, 1, "requirement 0001 is used by")
}

func Test0006_3(t *testing.T) {
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", requirementDoc("0001.1")+"\n## Out of scope\n\n- Subtraction.\n"))
	sees(t, out, code, 1, "sections must be exactly ## Why, ## Measurement, ## Requirements")
}

func Test0006_4(t *testing.T) {
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", strings.Replace(requirementDoc("0001.1"), "\n-\n", "\nover\n", 1)))
	sees(t, out, code, 1, "Measurement needs four lines")
}

func Test0006_5(t *testing.T) {
	out, code := judged(t, docOf("docs/requirements/0001-adds-numbers.md", docWith("Source: `adder.go:5` #12", acceptances("0001.1"))))
	sees(t, out, code, 1, "Source must cite either one existing line of code")
}

func Test0006_6(t *testing.T) {
	out, code := judged(t, reqs(acceptances("0001.1")+"        - When I add them again\n"))
	sees(t, out, code, 1, "needs exactly one Given, one When, and one Then")
}

func Test0006_7(t *testing.T) {
	out, code := judged(t, reqs("- 0001.1\n    - When I add them\n        - Given two numbers\n            - Then I get their sum\n"))
	sees(t, out, code, 1, "each acceptance is one Given, then one When, then one Then, each nested under the line above")
}

func Test0006_8(t *testing.T) {
	out, code := judged(t, reqs(acceptances("0001.1", "0001.1")))
	sees(t, out, code, 1, "0001.1 appears twice")
}
