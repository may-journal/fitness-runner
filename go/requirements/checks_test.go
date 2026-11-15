// Helpers the engine's requirement tests share with the check tests that
// moved to fitness-checks.
package requirements

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// installedCheck is the program of check that tests/requirements.sh installed
// from its fitness-checks release, for a test that starts it itself.
func installedCheck(t *testing.T, check string) string {
	t.Helper()
	path := filepath.Join(os.Getenv("FITNESS_CHECKS_DIR"), "fitness-check-"+check)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("check %s is not installed from fitness-checks: %v", check, err)
	}
	return path
}

// seededChecks links the suite's fresh check release index and checks into
// the private cache env names. A hook then installs every check without
// reading GitHub again.
func seededChecks(t *testing.T, env []string) []string {
	t.Helper()
	shared, private := os.Getenv("FITNESS_CACHE_DIR"), ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "FITNESS_CACHE_DIR="); ok {
			private = v
		}
	}
	if shared == "" || private == "" {
		t.Fatal("seededChecks needs the suite's cache and a private one")
	}
	index, err := os.ReadFile(filepath.Join(shared, "check-index.json"))
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(private, "check-index.json"), index, 0o600))
	from := filepath.Join(shared, "checks")
	mustDo(t, filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		to := filepath.Join(private, "checks", strings.TrimPrefix(path, from))
		if entry.IsDir() {
			return os.MkdirAll(to, 0o700)
		}
		return os.Link(path, to)
	}))
	return env
}

// acFixture is a small complete architecture under root. up leads from the
// root back to the repository, for links.
func acFixture(root string) map[string]string {
	up := strings.Repeat("../", strings.Count(root, "/"))
	return map[string]string{
		root + acContext: acView("C4Context\nPerson(user, \"1 User\")\nSystem(app, \"2 App\")\nRel(user, app, \"3\")",
			"A person who uses the app.", "The system.", "Uses it.") +
			"\n```mermaid\nC4Context\ntitle Legend\nPerson(p, \"Person\")\n```\n",
		root + acContainer: acView("C4Container\nPerson(user, \"1 User\")\nSystem_Boundary(app, \"2 App\") {\nContainer(service, \"3 Service\", \"Go\", \"src/cmd/service\", $tags=\"v1\")\n}\nRel(user, service, \"4\")\nUpdateElementStyle(user, $fontColor=\"red\")",
			"Same person.", "The system.", "Runs requests.", "Calls it."),
		root + acComponent: acView("C4Component\nContainer_Boundary(service, \"1 Service (src/cmd/service)\") {\nComponent(entry, \"2 Entry\", \"Go\")\nComponent(store, \"3 Store\", \"Go\", \"src/store\")\n}\nRel(entry, store, \"4\")",
			"The service.", "`src/cmd/service/main.go` starts it.", "Keeps records.", "Saves through the store, see [source]("+up+"src/cmd/service/main.go"+acEvidence+"."),
		root + acCode: acView("classDiagram\nclass Record[\"1 Record\"] {\n+ID\n}\nclass save[\"2 save()\"]\nsave --> Record : 3",
			"A saved record.", "Writes one record.", "Save returns the record.") +
			"\n```mermaid\nclassDiagram\nclass Example\nParent <|-- Child : inherits\n```\n",
		root + acJourney: "## Legend\n\n```mermaid\nflowchart TB\nperson((1 Person)):::persona\nclassDef persona fill:#fff\n```\n\n| # | Description | Why |\n| --- | --- | --- |\n| 1 | Persona | A legend row. |\n\n" +
			acView("flowchart TB\nuser((1 User)):::persona\nopen(2 Open the app):::action\ndone[3 Record saved]:::screen\nuser --> open\nopen -->|saves| done\nclassDef persona fill:#fff",
				"Same person.", "Opens it, `0001.1`.", "Shows the record, see [0001]("+up+"docs/requirements/0001-open.md)."),
		"src/cmd/service/main.go": "package main\n\nfunc main() {}\n",
		"src/store/store.go":      "package store\n",
		"src/store/store_test.go": "package store\n",
		"tools/check.sh":          "#!/bin/sh\nexit 0\n",
		acDoc:                     acDocWith("User opens the app"),
	}
}

// architectureChecks are the six checks that read the architecture model.
var architectureChecks = []string{"architecture-actors", cvCheck, "architecture-components", "architecture-containers", "architecture-contract", "architecture-journey"}

// atEachFile names line 3 of each inComments file, followed by message.
func atEachFile(message string) []string {
	return []string{"app/App.swift:3: " + message, "app/index.ts:3: " + message, "app/main.go:3: " + message}
}

// bareEntryText is entryFileText without front matter, which is optional.
func bareEntryText(stamp, item string) string {
	return "### " + stamp + "\n\n- " + item + "\n- Test: cover it.\n- Docs: describe it.\n"
}

// buildOutputCheck is the check these tests run.
const buildOutputCheck = "build-output-untracked"

// buildOutputIgnore is a .gitignore that ignores dist.
const buildOutputIgnore = "dist\n"

// changelogBase is happyRepo's changelog.
const changelogBase = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n" +
	"### 2026.10.08.1400\n\n- Docs: add the readme.\n- Docs: add the changelog.\n- Docs: add the license note.\n"

// changelogEntry is happyRepo's changelog with a new section, holding one
// bullet item, on top.
func changelogEntry(stamp, item string) string {
	return strings.Replace(changelogBase, "## Changes\n\n", "## Changes\n\n### "+stamp+"\n\n- "+item+"\n\n", 1)
}

// dupBlock builds lines of distinct words, so a shared prefix yields an
// exact copy and different prefixes never match. Six lines of ten words
// plus ";" is 66 tokens, past the 50-token minimum.
func dupBlock(prefix string, lines, perLine int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < perLine; j++ {
			fmt.Fprintf(&b, "%s_%d_%d ", prefix, i, j)
		}
		b.WriteString(";\n")
	}
	return b.String()
}

// dupClone is a copied block long enough to count as a clone.
var dupClone = dupBlock("c", 6, 10)

// entryFileText is an entry file under stamp with three typed bullets, the
// first one item, below the front matter older entries carry.
func entryFileText(stamp, item string) string {
	return "---\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n\n" + bareEntryText(stamp, item)
}

// goTest runs `fitness-install -- go-test` on happyRepo with files written
// over it and env added.
func goTest(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, "go-test")
}

// heldCheck is ADR 0001's check folder, its README holding the decision.
func heldCheck() map[string]string {
	return map[string]string{"fitness/adr-0001/README.md": heldDecision, "fitness/adr-0001/check": verdictScript(passing)}
}

// iloHead returns repo's HEAD commit.
func iloHead(t *testing.T, repo string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	mustDo(t, err)
	return strings.TrimSpace(string(out))
}

// inComments writes a doc comment of lines on line 3 of a Go, a Swift and a TypeScript file.
func inComments(lines ...string) map[string]string {
	mark := func(marker, code string, lines []string) string {
		var b strings.Builder
		for _, l := range lines {
			b.WriteString(strings.TrimRight(marker+" "+l, " ") + "\n")
		}
		return b.String() + code
	}
	return map[string]string{
		"app/main.go":   "package main\n\n" + mark("//", "func main() {}\n", lines),
		"app/App.swift": "import Foundation\n\n" + mark("///", "func run() {}\n", lines),
		"app/index.ts":  "export {};\n\n" + mark("//", "export const x = 1;\n", lines),
	}
}

// javaVersionRelease is a Maven build whose compiler release is release.
func javaVersionRelease(release string) string {
	return "<project><properties><maven.compiler.release>" + release + "</maven.compiler.release></properties></project>\n"
}

// javaVersionRun runs `fitness-install -- java-version` on a Java repo with
// files written over it and a JDK 21 running.
func javaVersionRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	_, out, code := javaRun(t, javaRepo(t, files), "java-version", nil)
	return out, code
}

// kebabDocsCheck is the check under test.
const kebabDocsCheck = "markdown-filename-kebab-case"

// kebabDocsRepo is a repo whose docs follow kebab-case, with files added.
func kebabDocsRepo(files map[string]string) map[string]string {
	repo := map[string]string{
		"docs/api-design.md": "# API\n",
		"docs/user-guide.md": "# Guide\n",
	}
	maps.Copy(repo, files)
	return repo
}

// longSentence has 30 words, past the 23-word default.
var longSentence = strings.Repeat("word ", 29) + "end.\n"

// noBoldItalicRule is the advice every finding carries.
const noBoldItalicRule = " (use only when explicitly required): "

// numbered is a numbered diagram and its matching callout table.
const numbered = "```mermaid\nflowchart TD\n  A[1 Start]:::s --> B[2 End]:::s\n  classDef s fill:#eef\n```\n\n" +
	"| # | Element | Description | Why |\n| - | - | - | - |\n| 1 | Start | First. | Begins. |\n| 2 | End | Last. | Ends. |\n"

// vcfConfig is a local Vitest config holding the given coverage thresholds
// over src/**/*.ts, the way a consumer repo writes one.
func vcfConfig(branches string) string {
	return "export default {\n  test: {\n    coverage: {\n      include: ['src/**/*.ts'],\n      provider: 'v8',\n" +
		"      thresholds: { branches: " + branches + ", functions: 100, lines: 100, statements: 100 },\n" +
		"    },\n    include: ['src/**/*.test.ts'],\n  },\n};\n"
}

// vcfFullRun is what `vitest run --coverage` prints when every line ran.
var vcfFullRun = response{
	Match: []string{"run --coverage"},
	Stdout: "\n RUN  v3.2.4 /repo\n\n \u2713 src/add.test.ts (1 test) 2ms\n\n" +
		" Test Files  1 passed (1)\n      Tests  1 passed (1)\n   Start at  10:15:02\n   Duration  412ms\n\n" +
		vcfTable("All files |     100 |      100 |     100 |     100 |                   \n"+
			" add.ts   |     100 |      100 |     100 |     100 |                   \n"),
}

// vcfRepo is happyRepo with vcfProject and then files written over it.
func vcfRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{}
	for _, set := range []map[string]string{vcfProject, files} {
		for name, body := range set {
			all[name] = body
		}
	}
	return example(t, "happyRepo", all)
}

func vcfRun(t *testing.T, repo string, env []string) (string, int) {
	t.Helper()
	return fitness(t, repo, env, "vitest-coverage-full")
}

// withEntries is files over happyRepo holding each named entry file.
func withEntries(entries map[string]string) map[string]string {
	files := map[string]string{}
	for name, body := range entries {
		files["changelog/"+name+".md"] = body
	}
	return files
}

// The architecture view file names the moved architecture tests used.
const (
	acContext   = "01-system-context.md"
	acContainer = "02-containers.md"
	acComponent = "03-components.md"
	acCode      = "04-code.md"
	acJourney   = "05-user-journey.md"
	acEvidence  = "#L3)"
)

// cvCheck is the architecture-code check.
const cvCheck = "architecture-code"

// heldDecision is a check README that still holds its decision.
const heldDecision = "# 0001 — Small files\n\nSmall files read fast.\n\n## Context\n\nLong files hide their point.\n\n" +
	"## Decision\n\n1. Every Markdown file stays small.\n\n## Consequences\n\n1. Readers find the point fast.\n\n" +
	"## Check\n\nFails a long Markdown file.\n"

// acView writes a view as people already do, with a heading, a diagram and its
// callout table, and nothing added for the check. The diagram's Nth line is line
// N+3, and the first row sits on line count+8.
func acView(diagram string, rows ...string) string {
	body := "# View\n\n```mermaid\n" + diagram + "\n```\n\n| # | Description | Why |\n| --- | --- | --- |\n"
	for i, row := range rows {
		body += "| " + strconv.Itoa(i+1) + " | " + row + " | Shared by later checks. |\n"
	}
	return body
}

// vcfProject is a small JavaScript project whose one function is fully tested.
var vcfProject = map[string]string{
	"package.json":      `{"type": "module"}` + "\n",
	"vitest.config.mjs": vcfConfig("100"),
	"src/add.ts":        "export const add = (a: number, b: number): number => a + b;\n",
	"src/add.test.ts": "import { expect, test } from 'vitest';\nimport { add } from './add';\n\n" +
		"test('adds', () => {\n  expect(add(1, 2)).toBe(3);\n});\n",
}

// vcfTable is the coverage table Vitest's v8 provider prints after a run.
func vcfTable(rows string) string {
	rule := "----------|---------|----------|---------|---------|-------------------\n"
	return " % Coverage report from v8\n" + rule +
		"File      | % Stmts | % Branch | % Funcs | % Lines | Uncovered Line #s \n" + rule + rows + rule
}

// acDoc is the requirement doc the architecture fixture opens.
const acDoc = "docs/requirements/0001-open.md"

// acDocWith is requirement 0001 with one acceptance whose Given is given.
// The Given sits on line 16.
func acDocWith(given string) string {
	return "# 0001 Open\n\n## Why\n\nI open the app.\n\n## Measurement\n\nopens\n-\ntries\n\n## Requirements\n\n- 0001.1\n    - Given " + given + "\n        - When it starts\n            - Then it opens\n"
}
