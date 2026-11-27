// Package requirements proves the Java module's docs/requirements as a user
// meets them. Each test runs the installer at FITNESS_INSTALL in a copied
// example repo.
package requirements

import (
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/acceptance"
)

func TestMain(m *testing.M) {
	acceptance.Main(m)
}

func Test0001_1(t *testing.T) {
	acceptance.Cases{
		{"dynamic tests, a dynamic container, Named arguments, a Java-only escape, a text block and other calls: the check sees each case name, then its location, then the step a name opens with", func(t *testing.T) {
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
			out := acceptance.SharedScan(t, map[string]string{"src/test/java/app/CasesTest.java": suite})
			at := func(line string) string { return `"Loc":"src/test/java/app/CasesTest.java:` + line + `"` }
			acceptance.Wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1",`+at("11")+`},`+
				`{"Name":"group",`+at("12")+`},`+
				`{"Name":"0001.1.2 keeps  spaces","Step":"0001.1.2",`+at("13")+`},`+
				`{"Name":"0001.1.3 named","Step":"0001.1.3",`+at("14")+`},`+
				`{"Name":"plain",`+at("14")+`},`+
				`{"Name":"tab\there 'q'",`+at("19")+`}],`+at("8"),
				`"Name":"helper","ID":"",`+at("24"))
		}},
	}.Run(t)
}
