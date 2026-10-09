package requirements

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
