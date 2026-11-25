package requirements

import (
	"strings"
	"testing"
)

func Test0009_1(t *testing.T) {
	cases{
		{"t.Run cases, nested ones, a named case table, an anonymous one and rows without a name", func(t *testing.T) {
			t.Parallel()
			suite := "package cases\n\nimport \"testing\"\n\n" +
				"type edgeCase struct {\n\tname string\n\trun  func(t *testing.T)\n}\n\n" +
				"type table []struct {\n\tName string\n\twant int\n}\n\n" +
				"type empty struct{}\n\ntype listed struct {\n\tnames []string\n}\n\n" +
				"func Test0001_1(t *testing.T) {\n" +
				"\tt.Run(\"0001.1.1 installs\", func(t *testing.T) {\n\t\tt.Run(\"nested\", func(t *testing.T) {})\n\t})\n" +
				"\trunCases(t, []edgeCase{\n\t\t{\"0001.1.2: updates\", nil},\n\t\t{name: \"keyed\"},\n\t\t{run: nil},\n\t\t{},\n\t\tshared,\n\t})\n" +
				"\t_ = table{{`0001.1.10 raw`, 1}}\n" +
				"\t_ = []struct{ in, want string }{{\"not a case\", \"x\"}}\n" +
				"\t_ = []empty{{}}\n\t_ = []listed{{nil}}\n\t_ = []int{1}\n\t_ = map[string]int{}\n" +
				"\tt.Run(name, nil)\n\tt.Run(\"three\", nil, nil)\n\thelper(\"not a case\")\n" +
				"\t_ = []edgeCase{{0, nil}}\n" +
				"\ttype local struct{ name string }\n\tvar count int\n\t_ = []local{{\"0001.1.6 local\"}}\n" +
				"\t_ = []*edgeCase{{\"0001.1.7 pointer\", nil}, &edgeCase{name: \"addressed\"}}\n" +
				"\t_ = map[string]struct{ want int }{\"0001.1.8 mapped\": {1}}\n\t_ = map[int]edgeCase{}\n" +
				"\tt.Run(\"0001.1.4, then\", nil)\n\tt.Run(\"0001.1.5.1 deeper\", nil)\n\tt.Run(\"0001.1.9.\", nil)\n" +
				"}\n\nfunc TestHelper(t *testing.T) {\n\tt.Run(\"0001.1.3 outside\", nil)\n}\n"
			out := sharedScan(t, map[string]string{"cases/0001_test.go": suite})
			wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1","Loc":"cases/0001_test.go:22"},`+
				`{"Name":"nested","Loc":"cases/0001_test.go:23"},`+
				`{"Name":"0001.1.2: updates","Step":"0001.1.2","Loc":"cases/0001_test.go:26"},`+
				`{"Name":"keyed","Loc":"cases/0001_test.go:27"},`+
				`{"Name":"0001.1.10 raw","Step":"0001.1.10","Loc":"cases/0001_test.go:32"},`+
				`{"Name":"0001.1.6 local","Step":"0001.1.6","Loc":"cases/0001_test.go:44"},`+
				`{"Name":"0001.1.7 pointer","Step":"0001.1.7","Loc":"cases/0001_test.go:45"},`+
				`{"Name":"addressed","Loc":"cases/0001_test.go:45"},`+
				`{"Name":"0001.1.8 mapped","Step":"0001.1.8","Loc":"cases/0001_test.go:46"},`+
				`{"Name":"0001.1.4, then","Step":"0001.1.4","Loc":"cases/0001_test.go:48"},`+
				`{"Name":"0001.1.5.1 deeper","Loc":"cases/0001_test.go:49"},`+
				`{"Name":"0001.1.9.","Step":"0001.1.9","Loc":"cases/0001_test.go:50"}],"Loc":"cases/0001_test.go:21"`,
				`"Name":"TestHelper","ID":"","Loc"`)
		}},
	}.run(t)
}

func Test0009_2(t *testing.T) {
	cases{
		{"dynamic tests, a dynamic container, Named arguments, a Java-only escape, a text block and other calls", func(t *testing.T) {
			t.Parallel()
			suite := "package app;\n\nimport static org.junit.jupiter.api.DynamicTest.dynamicTest;\n\nimport org.junit.jupiter.api.*;\n\n" +
				"class CasesTest {\n" +
				"  @TestFactory\n  Stream<DynamicNode> test0001_1() {\n    return Stream.of(\n" +
				"      dynamicTest(\"0001.1.1 installs\", () -> {}),\n" +
				"      DynamicContainer.dynamicContainer(\"group\", Stream.of(\n" +
				"        DynamicTest.dynamicTest(\"0001.1.2 keeps\\s spaces\", () -> {}))),\n" +
				"      Arguments.of(Named.of(\"0001.1.3 named\", 1), Named.named(\"plain\", 2)),\n" +
				"      dynamicTest(\"\"\"\n        block\"\"\", () -> {}),\n" +
				"      dynamicTest('c', () -> {}),\n" +
				"      dynamicTest(\"bad \\q\", () -> {}),\n" +
				"      dynamicTest(\"tab\\there \\'q\\'\", () -> {}),\n" +
				"      named(\"matcher\"), Other.named(\"other\"), XNamed.of(\"lookalike\"),\n" +
				"      fail(\"not a case\"));\n  }\n\n" +
				"  @Test\n  void helper() {\n    dynamicTest(\"0001.1.4 outside\", () -> {});\n  }\n}\n"
			out := sharedScan(t, map[string]string{"src/test/java/app/CasesTest.java": suite})
			at := func(line string) string { return `"Loc":"src/test/java/app/CasesTest.java:` + line + `"` }
			wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1",`+at("11")+`},`+
				`{"Name":"group",`+at("12")+`},`+
				`{"Name":"0001.1.2 keeps  spaces","Step":"0001.1.2",`+at("13")+`},`+
				`{"Name":"0001.1.3 named","Step":"0001.1.3",`+at("14")+`},`+
				`{"Name":"plain",`+at("14")+`},`+
				`{"Name":"tab\there 'q'",`+at("19")+`}],`+at("8"),
				`"Name":"helper","ID":"",`+at("24"))
		}},
	}.run(t)
}

func Test0009_3(t *testing.T) {
	cases{
		{"named activities, a nested one, a computed name, a name inside a string and tests without a body", func(t *testing.T) {
			t.Parallel()
			suite := "import XCTest\n\nfinal class CasesTests: XCTestCase {\n" +
				"    func test0001_1() {\n" +
				"        XCTContext.runActivity(named: \"0001.1.1 installs\") { _ in\n" +
				"            XCTContext.runActivity(named: \"nested\") { _ in }\n        }\n" +
				"        XCTContext.runActivity(named: name) { _ in }\n" +
				"        XCTContext.runActivity(named: \"0001.1.3 \\(item)\") { _ in }\n" +
				"        let text = \"XCTContext.runActivity(named: \\\"fake\\\")\"\n    }\n\n" +
				"    func testHelper() {\n        XCTContext.runActivity(named: \"0001.1.2 outside\") { _ in }\n    }\n}\n"
			out := sharedScan(t, map[string]string{
				"Tests/AppTests/CasesTests.swift": suite,
				"Tests/AppTests/OpenTests.swift":  "import XCTest\n\nfinal class OpenTests: XCTestCase {\n    func test0002_2() {\n        XCTContext.runActivity(named: \"0002.2.1 unclosed\") { _ in\n",
				"Tests/AppTests/BareTests.swift": "import XCTest\n\nfinal class BareTests: XCTestCase {\n    func test0002_1()\n" +
					"    func helper() { XCTContext.runActivity(named: \"stolen\") { _ in } }\n    func test0002_3()\n",
			})
			wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1","Loc":"Tests/AppTests/CasesTests.swift:5"},`+
				`{"Name":"nested","Loc":"Tests/AppTests/CasesTests.swift:6"}],"Loc":"Tests/AppTests/CasesTests.swift:4"`,
				`"Name":"testHelper","ID":"","Loc"`,
				`"ID":"0002.1","Loc"`,
				`"ID":"0002.2","Cases":[{"Name":"0002.2.1 unclosed","Step":"0002.2.1","Loc":"Tests/AppTests/OpenTests.swift:5"}]`)
		}},
	}.run(t)
}

// sharedScan runs every check on a repo holding files, and returns what
// fitness shows. The repo's own scan-reader check prints the test scan
// the runner shares.
func sharedScan(t *testing.T, files map[string]string) string {
	t.Helper()
	files["fitness/scan-reader/README.md"] = houseReadme
	files["fitness/scan-reader/check"] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"scan-reader\", \"tests\": true}'; exit 0; }\n" +
		"cat \"$FITNESS_TEST_SCAN\" >&2\necho '" + passing + "'\n"
	out, _ := fitness(t, example(t, "happyRepo", files), nil, "--all")
	return out
}

// wants fails for each fragment the output lacks.
func wants(t *testing.T, out string, fragments ...string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(out, want) {
			t.Errorf("the shared scan must hold %s, got:\n%s", want, out)
		}
	}
}
