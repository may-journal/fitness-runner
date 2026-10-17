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

// checked runs only the requirements check on happyRepo plus files, so a
// TypeScript repo meets no JavaScript tool.
func checked(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "requirements")
}

// scriptChecked runs the requirements check on scriptRepo with its test file
// replaced by statements.
func scriptChecked(t *testing.T, statements ...string) (string, int) {
	t.Helper()
	return checked(t, with(scriptRepo(), map[string]string{"tests/adder.test.ts": vitestFile(statements...)}))
}

// notTests mention 0001.1 in ways vitest runs no test for.
var notTests = []string{
	`// test("0001.1 in a comment", () => {});`,
	`const note = "test('0001.1 in a string', () => {})";`,
	`describe("0001.1 a describe title", () => {});`,
	`class Matcher { test(value: string) { return /0001\.1/.test(value); } }`,
	`class Typed { test(value: string): boolean { return value === "0001.1"; } }`,
	`interface Check { test(value: string): boolean; }`,
}

func Test0065_20(t *testing.T) {
	t.Parallel()
	out, _ := swiftJudged(t, nil)
	passedWith(t, out, "requirements", "2")
	out, _ = swiftJudged(t, map[string]string{
		"Tests/AdderTests/AdderTests.swift": "import Testing\n\nstruct AdderSpec {\n  @Test(\"adds\") func test0001_1() {}\n}\n",
	})
	passedWith(t, out, "requirements", "2")
	swiftOwnsNothing(t, xcTestFile()+"// func test0001_1() {}\n/* nested /* func test0001_1() {} */ */\nlet note = \"\\(\"func test0001_1() {}\")\"\nlet raw = #\"\"\"\n\" func test0001_1() {}\n\"\"\"#\n")
	swiftOwnsNothing(t, xcTestFile()+"\nstruct Helper {\n  func test0001_1() {}\n}\n")
	for _, owner := range []string{
		`test("0001.1 adds two numbers", () => {});`,
		`describe("adder", () => { it.each([1, 2])("0001.1 adds %i", (n) => { expect(n).toBeTruthy(); }); });`,
		"test.skip(`0001.1 adds two numbers`, () => {});",
		`test("0001.1 adds two numbers", () => { test.skip(process.env.CI === undefined, "0001.1 runs in CI"); });`,
	} {
		out, code := scriptChecked(t, append([]string{owner}, notTests...)...)
		sees(t, out, code, 0)
		passedWith(t, out, "requirements", "2")
	}
	jestFolder := map[string]string{"tests/adder.test.ts": "", "src/__tests__/adder.ts": vitestFile(`test("0001.1 adds two numbers", () => {});`)}
	out, code := checked(t, with(scriptRepo(), jestFolder))
	sees(t, out, code, 0)
	passedWith(t, out, "requirements", "2")
	out, code = scriptChecked(t, notTests...)
	sees(t, out, code, 1, `0001.1 has no test; name exactly one test test("0001.1 …")`)
	junitOwnership(t)
}

// junitOwnership checks a JUnit test owns 0001.1 by its name and test
// annotation, and a comment, string or helper method does not.
func junitOwnership(t *testing.T) {
	t.Helper()
	parameterized := "import org.junit.jupiter.params.ParameterizedTest;\n\nclass AdderTest {\n  @ParameterizedTest\n  @ValueSource(ints = {1, 2})\n  void test0001_1(int n) {}\n}\n"
	out, code := checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": parameterized}))
	sees(t, out, code, 0)
	passedWith(t, out, "requirements", "2")
	notJUnit := "class AdderTest {\n  // @Test void test0001_1() {}\n  String note = \"@Test void test0001_1() {}\";\n\n  void test0001_1() {}\n}\n"
	out, code = checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": notJUnit}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test test0001_1")
}

func Test0065_21(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
	swiftOwnsNothing(t, xcTestFile())
	out, code = checked(t, with(scriptRepo(), map[string]string{"tests/adder.test.ts": ""}))
	sees(t, out, code, 1, `0001.1 has no test; name exactly one test test("0001.1 …")`)
	out, code = checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": junitFile("AdderTest")}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test test0001_1")
}

func Test0065_17(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
}

func Test0065_18(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines")
}

func Test0065_22(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	sees(t, out, code, 1, "TestAddsAgain proves no requirement")
	out, code = swiftJudged(t, map[string]string{"Tests/AdderTests/AdderTests.swift": xcTestFile("test0001_1", "testAddsAgain")})
	sees(t, out, code, 1, "testAddsAgain proves no requirement; rename it testNNNN_N")
	out, code = scriptChecked(t, `test("0001.1 adds two numbers", () => {});`, `it("adds again", () => {});`)
	sees(t, out, code, 1, `tests/adder.test.ts:4: it("adds again") proves no requirement; rename it test("NNNN.N …")`)
	out, code = checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": junitFile("AdderTest", "test0001_1", "addsAgain")}))
	sees(t, out, code, 1, "src/test/java/AdderTest.java:6: addsAgain proves no requirement; rename it testNNNN_N")
}

func Test0065_23(t *testing.T) {
	t.Parallel()
	unchangedTestFails(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}), "TestAddsAgain proves no requirement")
	for _, body := range []string{xcTestFile("testAddsAgain"), "import Testing\n@Test func addsAgain() {}\n"} {
		files := with(swiftRepo(), map[string]string{"Tests/ExtraTests.swift": body})
		unchangedTestFails(t, files, "proves no requirement; rename it testNNNN_N")
	}
	files := with(scriptRepo(), map[string]string{"tests/extra.spec.js": vitestFile(`it("adds again", () => {});`)})
	unchangedTestFails(t, files, `it("adds again") proves no requirement`)
	files = with(junitRepo(), map[string]string{"src/test/java/ExtraTest.java": junitFile("ExtraTest", "addsAgain")})
	unchangedTestFails(t, files, "addsAgain proves no requirement")
}

func Test0065_24(t *testing.T) {
	t.Parallel()
	for _, repo := range []map[string]string{goRepo(), swiftRepo(), scriptRepo(), junitRepo()} {
		out, code := checked(t, with(repo, map[string]string{"docs/requirements/0001-adds-numbers.md": ""}))
		sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
	}
	out, code := checked(t, map[string]string{"src/app.js": "export const app = 1;\n"})
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
}

func Test0065_25(t *testing.T) {
	t.Parallel()
	for _, files := range []map[string]string{nil, {"vite.config.ts": "export default {};\n", ".eslintrc.cjs": "module.exports = {};\n"}} {
		out, code := checked(t, files)
		sees(t, out, code, 0)
		passedWithNoFiles(t, out, "requirements")
	}
}
