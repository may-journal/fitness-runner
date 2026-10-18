package requirements

import "testing"

// swiftTest is the path of swiftRepo's requirement test file, and
// swiftScreens of the helper that imports apple-ui-testing-library.
const (
	swiftTest    = "Tests/AdderTests/AdderTests.swift"
	swiftScreens = "Tests/AdderTests/Screens.swift"
)

// libraryHint is the part of every test library failure that says what to
// use instead.
const libraryHint = "write it with Swift Testing and UITestingLibrary, hosted in the app, as https://github.com/may-journal/apple-ui-testing-library shows"

// swiftChecked runs only the requirements check on swiftRepo with files
// written over it.
func swiftChecked(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return checked(t, with(swiftRepo(), files))
}

// librarySuite is a Swift Testing file whose test0001_1 runs body on
// apple-ui-testing-library, after the given extra imports.
func librarySuite(imports, body string) string {
	return "import Testing\nimport UITestingLibrary\n" + imports + "\n@MainActor @Test func test0001_1() async throws {\n" + body + "}\n"
}

// systemSuite is an end-to-end XCTest class whose test0001_1 drives the
// app through UITestingLibrarySystem.
const systemSuite = "import UITestingLibrary\nimport UITestingLibrarySystem\nimport XCTest\n\n@MainActor\nfinal class SystemRequirements: XCTestCase {\n" +
	"  func test0001_1() throws {\n    let session = SystemSession(bundleIdentifier: \"org.example.app\", timeout: 10)\n" +
	"    session.launch()\n    let reset = [\"xcrun\", \"simctl\", \"erase\", \"all\"]\n    XCTAssertFalse(reset.isEmpty)\n    XCTAssertNotNil(session.application as XCUIApplication?)\n  }\n}\n"

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
	for _, owner := range []string{
		"class AdderTest {\n  @ParameterizedTest\n  @ValueSource(ints = {1, 2})\n  void test0001_1(int n) {}\n}\n",
		"class AdderTest {\n  @TestFactory\n  Stream<DynamicTest> test0001_1() {\n    return Stream.empty();\n  }\n}\n",
		// TestNG runs a marked class's public methods, but not its setup.
		"@Test\npublic class AdderTest {\n  public AdderTest() {}\n\n  @BeforeTest\n  public void setUp() {}\n\n  public void test0001_1() {}\n}\n",
	} {
		out, code := checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": owner}))
		sees(t, out, code, 0)
		passedWith(t, out, "requirements", "2")
	}
	notJUnit := "class AdderTest {\n  // @Test void test0001_1() {}\n  String note = \"@Test void test0001_1() {}\";\n\n  void test0001_1() {}\n\n  @BeforeTest\n  void test0001_1() {}\n}\n"
	out, code := checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": notJUnit}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test test0001_1")
}

func Test0065_26(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
	swiftOwnsNothing(t, xcTestFile())
	out, code = checked(t, with(scriptRepo(), map[string]string{"tests/adder.test.ts": ""}))
	sees(t, out, code, 1, `0001.1 has no test; name exactly one test test("0001.1 …")`)
	out, code = checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": junitFile("AdderTest")}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test test0001_1")
	out, code = judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
	out, code = judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines")
}

func Test0065_27(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	sees(t, out, code, 1, "TestAddsAgain proves no requirement")
	out, code = swiftJudged(t, map[string]string{swiftTest: swiftTestingFile("test0001_1", "testAddsAgain")})
	sees(t, out, code, 1, "testAddsAgain proves no requirement; rename it testNNNN_N")
	out, code = scriptChecked(t, `test("0001.1 adds two numbers", () => {});`, `it("adds again", () => {});`)
	sees(t, out, code, 1, `tests/adder.test.ts:4: it("adds again") proves no requirement; rename it test("NNNN.N …")`)
	out, code = checked(t, with(junitRepo(), map[string]string{"src/test/java/AdderTest.java": junitFile("AdderTest", "test0001_1", "addsAgain")}))
	sees(t, out, code, 1, "src/test/java/AdderTest.java:6: addsAgain proves no requirement; rename it testNNNN_N")
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

func Test0065_28(t *testing.T) {
	t.Parallel()
	for _, body := range []string{xcTestFile("test0001_1"), "import Testing\n@testable import XCTest\n\n@Test func test0001_1() {}\n"} {
		out, code := swiftChecked(t, map[string]string{swiftTest: body})
		sees(t, out, code, 1, swiftTest+":", "imports XCTest; only an end-to-end test that imports UITestingLibrarySystem may use it, so "+libraryHint)
	}
	out, code := swiftChecked(t, map[string]string{swiftTest: xcTestFile("test0001_1")})
	sees(t, out, code, 1, swiftTest+":3: subclasses XCTestCase; "+libraryHint)
	out, code = swiftChecked(t, map[string]string{swiftTest: librarySuite("", "  let app = XCUIApplication()\n  app.launch()\n")})
	sees(t, out, code, 1, swiftTest+":5: drives Xcode UI tests through XCUIApplication, which launches the app in a separate process; "+libraryHint)
	boot := "  let shell = Process()\n  shell.arguments = [\"xcrun\", \"simctl\", \"boot\", \"booted\"]\n"
	out, code = swiftChecked(t, map[string]string{swiftTest: librarySuite("import Foundation\n", boot)})
	sees(t, out, code, 1, swiftTest+":7: boots a simulator through simctl outside the end-to-end mode; "+libraryHint)
	for _, framework := range []string{"ViewInspector", "SnapshotTesting", "KIF", "EarlGrey", "Hammer", "Nimble"} {
		out, code = swiftChecked(t, map[string]string{swiftTest: librarySuite("import "+framework+"\n", "")})
		sees(t, out, code, 1, swiftTest+":3: imports "+framework+", a UI test framework requirement tests may not use; "+libraryHint)
	}
	out, code = swiftChecked(t, map[string]string{swiftScreens: ""})
	sees(t, out, code, 1, swiftTest+": nothing in Tests/ imports UITestingLibrary; "+libraryHint)
	// A library import in another top folder does not reach the test.
	out, code = swiftChecked(t, map[string]string{swiftScreens: "", "Sources/Adder/Screens.swift": "import UITestingLibrary\n"})
	sees(t, out, code, 1, "nothing in Tests/ imports UITestingLibrary")
}

func Test0065_29(t *testing.T) {
	t.Parallel()
	// Words in comments and strings are no import or call.
	quiet := "import Testing\n// import XCTest\n/* XCUIApplication() */\n@MainActor @Test func test0001_1() async throws {\n  let note = \"XCUIApplication\"\n  _ = note\n}\n"
	for _, body := range []string{swiftTestingFile("test0001_1"), librarySuite("", ""), quiet} {
		out, code := swiftChecked(t, map[string]string{swiftTest: body})
		sees(t, out, code, 0)
		passedWith(t, out, "requirements", "2")
	}
	out, code := swiftChecked(t, map[string]string{swiftTest: systemSuite})
	sees(t, out, code, 0)
	passedWith(t, out, "requirements", "2")
}
