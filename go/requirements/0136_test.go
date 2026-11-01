package requirements

import (
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// commentLine matches a line `fitness comments` prints for one block.
var commentLine = regexp.MustCompile(`^\S+:\d+: \S+ (doc|inline): `)

// listComments writes files over happyRepo and runs `fitness comments`
// on the source files among them.
func listComments(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	var paths []string
	for _, p := range slices.Sorted(maps.Keys(files)) {
		if p != ".gitattributes" {
			paths = append(paths, p)
		}
	}
	return fitness(t, repo, nil, append([]string{"comments"}, paths...)...)
}

// listsExactly asserts the run exits 0 and its comment lines are want, in
// order.
func listsExactly(t *testing.T, out string, code int, want ...string) {
	t.Helper()
	var got []string
	for _, line := range strings.Split(out, "\n") {
		if commentLine.MatchString(line) {
			got = append(got, line)
		}
	}
	if code != 0 || !slices.Equal(got, want) {
		t.Errorf("exit %d, listed:\n%s\nwant:\n%s\noutput:\n%s", code, strings.Join(got, "\n"), strings.Join(want, "\n"), out)
	}
}

// backtick spells the quote that Go raw strings cannot hold.
const backtick = "`"

func Test0136_1(t *testing.T) {
	t.Parallel()
	out, code := listComments(t, map[string]string{
		"go/app/main.go": "package main\n\n// Main prints a link.\nfunc main() {\n\tprintln(\"see https://example.com // inside a string /* too */\")\n\tprintln(" +
			backtick + "// inside a string" + backtick + ")\n}\n",
		"swift/App.swift": `let url = "https://example.com // inside a string"
let raw = #"// inside a string"#
let mixed = "\("// inside a string")" // after a string
`,
		"ts/app.ts": "const url = \"https://example.com // inside a string\";\nconst text = " +
			backtick + "/* inside a string */ \\" + backtick + " // inside a string ${url}" + backtick + "; // after a template\n",
		"js/app.js": `const quote = 'it\'s // inside a string';
// Prints the quote.
console.log(quote);
`,
		"java/App.java": `class App {
  String url = "http://example.com // inside a string";
  char slash = '/'; // one slash
}
`,
		"kotlin/App.kt": `val text = """
  // inside a string
"""
/* outer /* nested */ still outer */
`,
		"rust/main.rs": `fn first<'a>(s: &'a str) -> &'a str { s } // takes a lifetime
fn main() { let u = "http://x // inside a string"; }
let r = r#"a "q" // inside a string"#; // after a raw string
`,
	})
	listsExactly(t, out, code,
		"go/app/main.go:3: go doc: Main prints a link.",
		"java/App.java:3: java inline: one slash",
		"js/app.js:2: javascript doc: Prints the quote.",
		"kotlin/App.kt:4: kotlin doc: outer /* nested */ still outer",
		"rust/main.rs:1: rust inline: takes a lifetime",
		"rust/main.rs:3: rust inline: after a raw string",
		"swift/App.swift:3: swift inline: after a string",
		"ts/app.ts:2: typescript inline: after a template",
	)
}

func Test0136_2(t *testing.T) {
	t.Parallel()
	out, code := listComments(t, map[string]string{
		"go/doc.go": `// Package doc holds docs.
// It spans two lines.
package doc

/*
Block explains
the next value.
*/
var x = 1 // trailing note
`,
		"swift/Doc.swift": `import Foundation

/// Greets a person.
/// - Parameter name: who.
func greet(name: String) {}

/**
 * A block doc.
 */
let y = 2 /* inline block */
`,
		"ts/doc.ts": `/**
 * Adds two numbers.
 */
export function add(a: number, b: number): number {
  return a + b; // the sum
}
`,
	})
	listsExactly(t, out, code,
		"go/doc.go:1: go doc: Package doc holds docs. It spans two lines.",
		"go/doc.go:5: go doc: Block explains the next value.",
		"go/doc.go:9: go inline: trailing note",
		"swift/Doc.swift:3: swift doc: Greets a person. - Parameter name: who.",
		"swift/Doc.swift:7: swift doc: A block doc.",
		"swift/Doc.swift:10: swift inline: inline block",
		"ts/doc.ts:1: typescript doc: Adds two numbers.",
		"ts/doc.ts:5: typescript inline: the sum",
	)
}

func Test0136_3(t *testing.T) {
	t.Parallel()
	out, code := listComments(t, map[string]string{
		"scripts/run.sh": `#!/bin/sh
# Runs the build.
echo "a # inside quotes" '# inside quotes' # after echo
cat <<EOF
# inside a here-document
EOF
size=${#HOME} # length
n=$((1 << SHIFT)) # shifted
# after the shift
`,
		"bin/deploy": "#!/usr/bin/env bash\n# Deploys the site.\n",
		"config/app.yml": `# App settings.
name: "a # inside quotes" # the name
url: http://example.com/#anchor
note: |
  # inside a block
list:
  - 'b # inside quotes'
`,
	})
	listsExactly(t, out, code,
		"bin/deploy:2: shell doc: Deploys the site.",
		"config/app.yml:1: yaml doc: App settings.",
		"config/app.yml:2: yaml inline: the name",
		"scripts/run.sh:2: shell doc: Runs the build.",
		"scripts/run.sh:3: shell inline: after echo",
		"scripts/run.sh:7: shell inline: length",
		"scripts/run.sh:8: shell inline: shifted",
		"scripts/run.sh:9: shell doc: after the shift",
	)
}

func Test0136_4(t *testing.T) {
	t.Parallel()
	out, code := listComments(t, map[string]string{
		".gitattributes": "go/gen.go linguist-generated\n",
		"go/gen.go":      "// Package gen is written by a tool.\npackage gen\n",
		"go/tool.go": `// Copyright 2026 Example. Licensed under MIT.

//go:build linux

// Package tool keeps a note.
package tool

// cspell:ignore zorp
var zorp = 1
`,
		"swift/Lint.swift": "// swiftlint:disable force_cast\nlet a = 1 // keeps this one\n",
		"scripts/lint.sh":  "#!/bin/bash\n# shellcheck disable=SC2086\necho $1 # keeps this one\n",
		"ts/lint.ts":       "/* eslint-disable no-console */\nconsole.log(1); // keeps this one\n",
	})
	listsExactly(t, out, code,
		"go/tool.go:5: go doc: Package tool keeps a note.",
		"scripts/lint.sh:3: shell inline: keeps this one",
		"swift/Lint.swift:2: swift inline: keeps this one",
		"ts/lint.ts:2: typescript inline: keeps this one",
	)
}

func Test0136_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"go/a/a.go":   "// Package a is in the folder.\npackage a\n",
		"go/b/b.go":   "// Package b is in the folder too.\npackage b\n",
		"ts/named.ts": "// Named by its absolute path.\nexport {};\n",
		"ts/other.ts": "// Named by no path.\nexport {};\n",
	})
	out, code := fitness(t, repo, nil, "comments", "go", filepath.Join(repo, "ts", "named.ts"))
	listsExactly(t, out, code,
		"go/a/a.go:1: go doc: Package a is in the folder.",
		"go/b/b.go:1: go doc: Package b is in the folder too.",
		"ts/named.ts:1: typescript doc: Named by its absolute path.",
	)
}

func Test0136_6(t *testing.T) {
	t.Parallel()
	out, code := listComments(t, map[string]string{
		"scripts/prose.sh":  "#!/bin/sh\n# shellCheck is a repo.\n# It keeps its second line.\n# shellcheck disable=SC2086\necho $1\n",
		"swift/Prose.swift": "// SwiftLint is a linter.\nlet a = 1\n// swiftlint:disable force_cast\nlet b = 2\n",
	})
	listsExactly(t, out, code,
		"scripts/prose.sh:2: shell doc: shellCheck is a repo. It keeps its second line.",
		"swift/Prose.swift:1: swift doc: SwiftLint is a linter.",
	)
}
