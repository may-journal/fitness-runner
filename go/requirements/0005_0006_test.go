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

func Test0005_2(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
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

func Test0005_16(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
	swiftOwnsNothing(t, xcTestFile())
}

func Test0005_17(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2", "TestAddsAgain")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines", "TestAddsAgain proves no requirement")
	out, code = swiftJudged(t, map[string]string{"Tests/AdderTests/AdderTests.swift": xcTestFile("test0001_1", "test0001_2", "testAddsAgain")})
	sees(t, out, code, 1, "test0001_2 names 0001.2, which no requirement defines", "testAddsAgain proves no requirement; rename it testNNNN_N")
}

func Test0005_18(t *testing.T) {
	t.Parallel()
	noDocs := map[string]string{"docs/requirements/0001-adds-numbers.md": ""}
	out, code := judged(t, with(goRepo(), noDocs))
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
	out, code = swiftJudged(t, noDocs)
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
}

func Test0005_19(t *testing.T) {
	t.Parallel()
	out, _ := swiftJudged(t, nil)
	passedWith(t, out, "requirements", "2")
	out, _ = swiftJudged(t, map[string]string{
		"Tests/AdderTests/AdderTests.swift": "import Testing\n\nstruct AdderSpec {\n  @Test(\"adds\") func test0001_1() {}\n}\n",
	})
	passedWith(t, out, "requirements", "2")
}

func Test0005_20(t *testing.T) {
	t.Parallel()
	swiftOwnsNothing(t, xcTestFile()+"// func test0001_1() {}\n/* nested /* func test0001_1() {} */ */\nlet note = \"\\(\"func test0001_1() {}\")\"\nlet raw = #\"\"\"\n\" func test0001_1() {}\n\"\"\"#\n")
	swiftOwnsNothing(t, xcTestFile()+"\nstruct Helper {\n  func test0001_1() {}\n}\n")
}

func Test0005_21(t *testing.T) {
	t.Parallel()
	out, _ := judged(t, nil)
	passedWithNoFiles(t, out, "requirements")
	repo := example(t, "happyRepo", with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	write(t, repo, map[string]string{"README.md": readme + "\nOne more short line.\n"})
	git(t, repo, "add", "README.md")
	out, _ = fitness(t, repo, nil, "--policy=external")
	passedWithNoFiles(t, out, "requirements")
}

// swiftJudged runs fitness on swiftRepo with files written over it.
func swiftJudged(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return judged(t, with(swiftRepo(), files))
}

// swiftOwnsNothing is a test file whose test0001_1 is not a test.
func swiftOwnsNothing(t *testing.T, body string) {
	t.Helper()
	out, code := swiftJudged(t, map[string]string{"Tests/AdderTests/AdderTests.swift": body})
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test test0001_1")
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

func Test0006_8(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1", "0001.1")))
	sees(t, out, code, 1, "0001.1 appears twice")
}

func Test0006_9(t *testing.T) {
	t.Parallel()
	again := strings.Replace(docWith("Source: `adder.go:5`", acceptances("0002.1")), "# 0001 Adds Numbers", "# 0002 Adds Again", 1)
	out, code := judged(t, with(goRepo(), map[string]string{
		"adder_test.go":                        testFile("adder", "Test0001_1", "Test0002_1"),
		"docs/requirements/0002-adds-again.md": again,
	}))
	sees(t, out, code, 1, "docs/requirements/0001-adds-numbers.md and docs/requirements/0002-adds-again.md both cover")
}

func Test0006_10(t *testing.T) {
	t.Parallel()
	out, code := judged(t, reqs(acceptances("0001.1")+"        - When I add them again\n"))
	sees(t, out, code, 1, "needs exactly one Given, one When, and one Then")
	out, code = judged(t, reqs("- 0001.1\n    - When I add them\n        - Given two numbers\n            - Then I get their sum\n"))
	sees(t, out, code, 1, "each acceptance is one Given, then one When, then one Then, each nested under the line above")
}
