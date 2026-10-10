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

func Test0065_1(t *testing.T) {
	t.Parallel()
	out, _ := swiftJudged(t, nil)
	passedWith(t, out, "requirements", "2")
}

func Test0065_2(t *testing.T) {
	t.Parallel()
	out, _ := swiftJudged(t, map[string]string{
		"Tests/AdderTests/AdderTests.swift": "import Testing\n\nstruct AdderSpec {\n  @Test(\"adds\") func test0001_1() {}\n}\n",
	})
	passedWith(t, out, "requirements", "2")
}

func Test0065_3(t *testing.T) {
	t.Parallel()
	swiftOwnsNothing(t, xcTestFile())
}

func Test0065_4(t *testing.T) {
	t.Parallel()
	out, code := swiftJudged(t, map[string]string{"Tests/AdderTests/AdderTests.swift": xcTestFile("test0001_1", "testAddsAgain")})
	sees(t, out, code, 1, "testAddsAgain proves no requirement; rename it testNNNN_N")
}

func Test0065_5(t *testing.T) {
	t.Parallel()
	swiftOwnsNothing(t, xcTestFile()+"// func test0001_1() {}\n/* nested /* func test0001_1() {} */ */\nlet note = \"\\(\"func test0001_1() {}\")\"\nlet raw = #\"\"\"\n\" func test0001_1() {}\n\"\"\"#\n")
}

func Test0065_6(t *testing.T) {
	t.Parallel()
	swiftOwnsNothing(t, xcTestFile()+"\nstruct Helper {\n  func test0001_1() {}\n}\n")
}

func Test0065_7(t *testing.T) {
	t.Parallel()
	out, code := swiftJudged(t, map[string]string{"docs/requirements/0001-adds-numbers.md": ""})
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
}
