package requirements

import "strings"

// The Given files a user's repo adds to happyRepo. They live here, not as
// tracked example repos, so this repo's own checks never judge them.

// goRepo is a Go module whose requirement 0001 has its one owning test.
func goRepo() map[string]string {
	return map[string]string{
		"go.mod":                                 "module example.com/adder\n\ngo 1.22\n",
		"adder.go":                               "// Package adder adds numbers.\npackage adder\n\n// Add returns the sum of a and b.\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
		"adder_test.go":                          testFile("adder", "Test0001_1"),
		"docs/requirements/0001-adds-numbers.md": requirementDoc("0001.1"),
	}
}

// pairedRepo is goRepo with requirement 0001's test in reqs/0001_test.go,
// named for it, beside a shared helper that proves no requirement. Only the
// requirements check runs, so commits meet no other check.
func pairedRepo() map[string]string {
	return with(goRepo(), map[string]string{
		".fitnessrc.json":    `{"policy": "external", "checks": ["requirements"]}` + "\n",
		"adder_test.go":      "",
		"reqs/0001_test.go":  testFile("reqs", "Test0001_1"),
		"reqs/tools_test.go": "package reqs\n\n// two is a helper the tests share.\nfunc two() int { return 2 }\n",
	})
}

// twoRepo is pairedRepo whose one test file proves requirements 0001 and 0002.
// Requirement 0002 covers its own area, so the two docs never overlap.
func twoRepo() map[string]string {
	doubles := "- 0002.1\n    - Given one number\n        - When I double it\n            - Then I get twice that number\n"
	return with(pairedRepo(), map[string]string{
		"reqs/0001_test.go":                    "",
		"reqs/0001_0002_test.go":               testFile("reqs", "Test0001_1", "Test0002_1"),
		"docs/requirements/0002-adds-again.md": strings.Replace(docWith("Source: `adder.go:5`", doubles), "0001 Adds Numbers", "0002 Adds Again", 1),
	})
}

// swiftRepo is a Swift package whose XCTest method test0001_1 owns 0001.1.
func swiftRepo() map[string]string {
	return map[string]string{
		"Package.swift":                          "// swift-tools-version:5.9\nimport PackageDescription\n\nlet package = Package(name: \"Adder\")\n",
		"Sources/Adder/Adder.swift":              "/// Adds numbers.\npublic enum Adder {\n  /// Returns the sum of a and b.\n  public static func add(_ a: Int, _ b: Int) -> Int { a + b }\n}\n",
		"Tests/AdderTests/AdderTests.swift":      xcTestFile("test0001_1"),
		"docs/requirements/0001-adds-numbers.md": docWith("Source: `Sources/Adder/Adder.swift:4`", acceptances("0001.1")),
	}
}

// xcTestFile is an XCTest case class declaring one empty test per name.
func xcTestFile(names ...string) string {
	body := "import XCTest\n\nfinal class AdderTests: XCTestCase {\n"
	for _, n := range names {
		body += "  func " + n + "() {}\n"
	}
	return body + "}\n"
}

// testFile is a Go test file in pkg declaring one passing test per name.
func testFile(pkg string, names ...string) string {
	body := "package " + pkg + "\n\nimport \"testing\"\n"
	for _, n := range names {
		body += "\nfunc " + n + "(t *testing.T) {}\n"
	}
	return body
}

// requirementDoc is requirement 0001 with one Given/When/Then per ID.
func requirementDoc(ids ...string) string {
	return docWith("Source: `adder.go:5`", acceptances(ids...))
}

// acceptances renders one well-formed chain per ID.
func acceptances(ids ...string) string {
	body := ""
	for _, id := range ids {
		body += "- " + id + "\n    - Given two numbers\n        - When I add them\n            - Then I get their sum\n"
	}
	return body
}

// docWith is requirement 0001 with the given Source line and Requirements.
func docWith(source, reqs string) string {
	return "---\nrelatedConfigurations: ['../../.fitnessrc.json']\n---\n\n# 0001 Adds Numbers\n\n## Why\n\n" +
		"I can add two numbers without doing it by hand.\n\n## Measurement\n\nsums that are right\n-\nsums asked for\n\n" +
		source + "\n\n## Requirements\n\n" + reqs
}

// with returns base with overrides applied.
func with(base map[string]string, overrides map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// hungTool is a JavaScript repo whose own prettier runs prelude, then hangs
// after starting a child process that records its PID in prettier.pid.
func hungTool(timeoutMs, prelude string) map[string]string {
	return map[string]string{
		".fitnessrc.json":            `{"timeoutMs": ` + timeoutMs + "}\n",
		"package.json":               "{\"name\": \"app\", \"private\": true}\n",
		"app.js":                     "const a = 1;\n",
		"node_modules/.bin/prettier": "#!/bin/sh\n" + prelude + "/bin/sleep 30 >/dev/null 2>&1 &\necho $! > \"$PWD/prettier.pid\"\nwait\n",
	}
}

// planBody is a Plan that follows the template, with one task.
func planBody(task string) string {
	return "> One clear pitch for this Plan.\n\n## Background\n\nThis Plan is a test fixture.\n\n## What needs to happen\n\n" + task + "\n"
}
