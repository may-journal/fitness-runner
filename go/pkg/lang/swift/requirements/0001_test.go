// Package requirements proves the Swift module's docs/requirements as a user
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
		{"named activities, a nested one, a computed name, a name inside a string and tests without a body: the check sees each activity name, then its location, then the step a name opens with", func(t *testing.T) {
			t.Parallel()
			suite := "import XCTest\n\nfinal class CasesTests: XCTestCase {\n" +
				"    func test0001_1() {\n" +
				"        XCTContext.runActivity(named: \"0001.1.1 installs\") { _ in\n" +
				"            XCTContext.runActivity(named: \"nested\") { _ in }\n        }\n" +
				"        XCTContext.runActivity(named: name) { _ in }\n" +
				"        XCTContext.runActivity(named: \"0001.1.3 \\(item)\") { _ in }\n" +
				"        let text = \"XCTContext.runActivity(named: \\\"fake\\\")\"\n    }\n\n" +
				"    func testHelper() {\n        XCTContext.runActivity(named: \"0001.1.2 outside\") { _ in }\n    }\n}\n"
			out := acceptance.SharedScan(t, map[string]string{
				"Tests/AppTests/CasesTests.swift": suite,
				"Tests/AppTests/OpenTests.swift":  "import XCTest\n\nfinal class OpenTests: XCTestCase {\n    func test0002_2() {\n        XCTContext.runActivity(named: \"0002.2.1 unclosed\") { _ in\n",
				"Tests/AppTests/BareTests.swift": "import XCTest\n\nfinal class BareTests: XCTestCase {\n    func test0002_1()\n" +
					"    func helper() { XCTContext.runActivity(named: \"stolen\") { _ in } }\n    func test0002_3()\n",
			})
			acceptance.Wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1","Loc":"Tests/AppTests/CasesTests.swift:5"},`+
				`{"Name":"nested","Loc":"Tests/AppTests/CasesTests.swift:6"}],"Loc":"Tests/AppTests/CasesTests.swift:4"`,
				`"Name":"testHelper","ID":"","Loc"`,
				`"ID":"0002.1","Loc"`,
				`"ID":"0002.2","Cases":[{"Name":"0002.2.1 unclosed","Step":"0002.2.1","Loc":"Tests/AppTests/OpenTests.swift:5"}]`)
		}},
	}.Run(t)
}
