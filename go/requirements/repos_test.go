package requirements

import (
	"encoding/json"
	"strings"
	"testing"
)

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
	doubles := "- 0002.1\n  - Given one number\n    - When I double it\n      - Then I get twice that number\n"
	return with(pairedRepo(), map[string]string{
		"reqs/0001_test.go":                    "",
		"reqs/0001_0002_test.go":               testFile("reqs", "Test0001_1", "Test0002_1"),
		"docs/requirements/0002-adds-again.md": strings.Replace(docWith("Source: `adder.go:5`", doubles), "0001 Adds Numbers", "0002 Adds Again", 1),
	})
}

// swiftRepo is a Swift package whose Swift Testing test test0001_1 owns
// 0001.1, beside a shared helper that imports apple-ui-testing-library.
func swiftRepo() map[string]string {
	return map[string]string{
		swiftScreens:                             "import UITestingLibrary\n\n/// Screens every test renders.\nenum Screens {}\n",
		"Package.swift":                          "// swift-tools-version:5.9\nimport PackageDescription\n\nlet package = Package(name: \"Adder\")\n",
		"Sources/Adder/Adder.swift":              "/// Adds numbers.\npublic enum Adder {\n  /// Returns the sum of a and b.\n  public static func add(_ a: Int, _ b: Int) -> Int { a + b }\n}\n",
		swiftTest:                                swiftTestingFile("test0001_1"),
		"docs/requirements/0001-adds-numbers.md": docWith("Source: `Sources/Adder/Adder.swift:4`", acceptances("0001.1")),
	}
}

// swiftTestingFile is a Swift Testing suite declaring one empty test per
// name.
func swiftTestingFile(names ...string) string {
	body := "import Testing\n\n@MainActor @Suite struct AdderSpec {\n"
	for _, n := range names {
		body += "  @Test func " + n + "() async throws {}\n"
	}
	return body + "}\n"
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

// acceptances renders one well-formed chain per ID, nested two spaces deeper
// at each level as Biome formats it.
func acceptances(ids ...string) string {
	body := ""
	for _, id := range ids {
		body += "- " + id + "\n  - Given two numbers\n    - When I add them\n      - Then I get their sum\n"
	}
	return body
}

// docWith is requirement 0001 with the given Source line and Requirements,
// its Measurement ratio set apart by blank lines as Biome formats it.
func docWith(source, reqs string) string {
	return "---\nrelatedConfigurations: ['../../.fitnessrc.json']\n---\n\n# 0001 Adds Numbers\n\n## Why\n\n" +
		"I can add two numbers without doing it by hand.\n\n## Measurement\n\nsums that are right\n\n-\n\nsums asked for\n\n" +
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

// planView is the saved gh answer to the pre-push gate's one read of issue n
// in repo0014: its URL, body, label names and comment bodies.
func planView(t *testing.T, n int, body string, labels []string, comments ...string) response {
	t.Helper()
	out, err := json.Marshal(map[string]any{
		"url":      "https://github.com/" + repo0014 + "/issues/" + itoa(n),
		"body":     body,
		"labels":   append([]string{}, labels...),
		"comments": append([]string{}, comments...),
	})
	mustDo(t, err)
	return response{Match: []string{"issue view " + itoa(n) + " ", "--json url,body,labels,comments"}, Stdout: string(out) + "\n"}
}

// junitRepo is a Maven project whose JUnit test test0001_1 owns 0001.1.
func junitRepo() map[string]string {
	return map[string]string{
		"pom.xml":                                "<project/>\n",
		"src/main/java/Adder.java":               "/** Adds numbers. */\nclass Adder {\n  static int add(int a, int b) {\n    return a + b;\n  }\n}\n",
		"src/test/java/AdderTest.java":           junitFile("AdderTest", "test0001_1"),
		"docs/requirements/0001-adds-numbers.md": docWith("Source: `src/main/java/Adder.java:3`", acceptances("0001.1")),
	}
}

// junitFile is a JUnit test class declaring one empty test per name.
func junitFile(class string, names ...string) string {
	body := "import org.junit.jupiter.api.Test;\n\nclass " + class + " {\n"
	for _, n := range names {
		body += "  @Test\n  void " + n + "() {}\n"
	}
	return body + "}\n"
}

// scriptRepo is a TypeScript project whose vitest test titled "0001.1 …"
// owns 0001.1.
func scriptRepo() map[string]string {
	return map[string]string{
		"src/adder.ts":                           "/** Adds numbers. */\nexport const add = (a: number, b: number): number => a + b;\n",
		"tests/adder.test.ts":                    vitestFile(`test("0001.1 adds two numbers", () => {});`),
		"docs/requirements/0001-adds-numbers.md": docWith("Source: `src/adder.ts:2`", acceptances("0001.1")),
	}
}

// vitestFile is a vitest test file holding the given statements.
func vitestFile(statements ...string) string {
	return "import { describe, expect, it, test } from \"vitest\";\n\n" + strings.Join(statements, "\n") + "\n"
}

// architectedRepo is a Go service whose requirement, owning test, and five
// architecture views keep every check green.
func architectedRepo() map[string]string {
	views := acFixture("docs/architecture/")
	files := map[string]string{
		"go.mod":                  "module example.com/service\n\ngo 1.22\n",
		"src/cmd/service/main.go": "// Command service saves one record.\npackage main\n\nimport \"example.com/service/src/store\"\n\nfunc main() { store.Save() }\n",
		"src/store/store.go":      "// Package store keeps records.\npackage store\n\n// Record is one saved record.\ntype Record struct{ ID int }\n\n// Save keeps one record and returns it.\nfunc Save() Record { return Record{} }\n",
		"src/store/store_test.go": testFile("store", "Test0001_1"),
		acDoc:                     front("../../") + acDocWith("User opens the app"),
	}
	// Connectors carry letter callouts, as mermaid-callouts requires.
	edits := map[string][]string{
		acContext: {"Rel(user, app, \"3\")", "Rel(user, app, \"A\")", "| 3 | Uses it.", "| A | Uses it."},
		acContainer: {"| 2 | The system. |", "| 2 | The app that holds the service. |",
			"Rel(user, service, \"4\")", "Rel(user, service, \"A\")", "| 4 | Calls it.", "| A | Calls it."},
		acComponent: {"main.go#L3", "main.go#L6", "Rel(entry, store, \"4\")", "Rel(entry, store, \"A\")", "| 4 | Saves", "| A | Saves"},
		acCode: {"class save[\"2 save()\"]\nsave --> Record : 3", "class Save[\"2 Save()\"]\nSave --> Record : A", "| 3 | Save returns", "| A | Save returns",
			"A saved record.", "A saved record, in [store.go](../../src/store/store.go#L5).",
			"Writes one record.", "Writes one record, in [store.go](../../src/store/store.go#L8)."},
		acJourney: {"open(2 Open the app):::action\ndone[3 Record saved]:::screen\nuser --> open\nopen -->|saves| done",
			"open(2 Open the app, Req 0001):::action\nuser --> open", "| 3 | Shows the record, see [0001](../../docs/requirements/0001-open.md). | Shared by later checks. |\n", ""},
	}
	for view, pairs := range edits {
		body := views["docs/architecture/"+view]
		for i := 0; i < len(pairs); i += 2 {
			body = strings.Replace(body, pairs[i], pairs[i+1], 1)
		}
		files["docs/architecture/"+view] = front("../../") + body
	}
	return with(views, files)
}

// front is the front matter a Markdown doc at depth up carries.
func front(up string) string {
	return "---\nrelatedConfigurations: ['" + up + ".fitnessrc.json']\n---\n\n"
}

// notesHead opens a changelog in the house layout.
const notesHead = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n"

// notesEntry is a timestamped entry that follows the house rules.
const notesEntry = "### 2026.10.09.1500\n\n- Fix: one.\n- Fix: two.\n- Fix: three.\n\n"

// notesRelease is a 1.1.1 release as Release Please writes it: a linked
// version heading, then each subheading with one bullet per scope given.
func notesRelease(subheadings map[string][]string) string {
	s := "## [1.1.1](https://github.com/o/r/compare/v1.1.0...v1.1.1) (2026-10-10)\n\n"
	for _, h := range []string{"Features", "Bug Fixes"} {
		if len(subheadings[h]) == 0 {
			continue
		}
		s += "\n### " + h + "\n\n"
		for _, scope := range subheadings[h] {
			s += "* **" + scope + ":** change it ([#239](https://github.com/o/r/issues/239)) ([c9f4729](https://github.com/o/r/commit/c9f4729))\n"
		}
		s += "\n"
	}
	return s
}

// notesChecks runs the named checks on happyRepo with changelog.
func notesChecks(t *testing.T, changelog, checks string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"CHANGELOG.md": changelog})
	return fitness(t, repo, nil, "--policy=external", "--checks="+checks, "--all")
}

// notesTwo is a release with one feature and one fix.
var notesTwo = map[string][]string{"Features": {"install"}, "Bug Fixes": {"release"}}
