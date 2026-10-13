package requirements

import "testing"

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

func Test0065_9(t *testing.T) {
	t.Parallel()
	out, _ := swiftJudged(t, nil)
	passedWith(t, out, "requirements", "2")
	out, _ = swiftJudged(t, map[string]string{
		"Tests/AdderTests/AdderTests.swift": "import Testing\n\nstruct AdderSpec {\n  @Test(\"adds\") func test0001_1() {}\n}\n",
	})
	passedWith(t, out, "requirements", "2")
}

func Test0065_10(t *testing.T) {
	t.Parallel()
	swiftOwnsNothing(t, xcTestFile()+"// func test0001_1() {}\n/* nested /* func test0001_1() {} */ */\nlet note = \"\\(\"func test0001_1() {}\")\"\nlet raw = #\"\"\"\n\" func test0001_1() {}\n\"\"\"#\n")
	swiftOwnsNothing(t, xcTestFile()+"\nstruct Helper {\n  func test0001_1() {}\n}\n")
}

func Test0065_11(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
	swiftOwnsNothing(t, xcTestFile())
}

func Test0065_12(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
	out, code = judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines")
}

func Test0065_13(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	sees(t, out, code, 1, "TestAddsAgain proves no requirement")
	out, code = swiftJudged(t, map[string]string{"Tests/AdderTests/AdderTests.swift": xcTestFile("test0001_1", "testAddsAgain")})
	sees(t, out, code, 1, "testAddsAgain proves no requirement; rename it testNNNN_N")
}

func Test0065_14(t *testing.T) {
	t.Parallel()
	unchangedTestFails(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}), "TestAddsAgain proves no requirement")
	for _, body := range []string{xcTestFile("testAddsAgain"), "import Testing\n@Test func addsAgain() {}\n"} {
		files := with(swiftRepo(), map[string]string{"Tests/ExtraTests.swift": body})
		unchangedTestFails(t, files, "proves no requirement; rename it testNNNN_N")
	}
}

func Test0065_15(t *testing.T) {
	t.Parallel()
	for _, repo := range []map[string]string{goRepo(), swiftRepo()} {
		out, code := judged(t, with(repo, map[string]string{"docs/requirements/0001-adds-numbers.md": ""}))
		sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
	}
}

func Test0065_16(t *testing.T) {
	t.Parallel()
	out, _ := judged(t, nil)
	passedWithNoFiles(t, out, "requirements")
}
