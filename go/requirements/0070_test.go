package requirements

import (
	"strings"
	"testing"
)

const acCheck = "architecture-contract"
const acContext = "01-system-context.md"
const acCode = "04-code.md"
const acJourney = "05-user-journey.md"

// acView keeps the fixture's contract beside the actual diagram. Fixtures
// are committed by example before the installed command reads them.
func acView(diagram string, contracts ...string) string {
	body := "```mermaid\n" + diagram + "\n```\n\n| # | Contract |\n| --- | --- |\n"
	for _, contract := range contracts {
		body += "| " + contract + " |\n"
	}
	return body
}

func acFixture(root string) map[string]string {
	return map[string]string{
		root + acContext: acView("C4Context\nPerson(host, \"1 Host\")\nSystem(app, \"2 App\")\nRel(host, app, \"3\")",
			"1 | id=host; type=actor; name=Host; aliases=Owner", "2 | id=app; type=system",
			"3 | id=uses-app; type=relationship; from=host; to=app; kind=uses; mode=allowed; evidence=declared"),
		root + "02-containers.md": acView("C4Container\nContainer(run, \"1 App\", \"Go\")",
			"1 | id=run; type=runtime; parent=app; target=go:src"),
		root + "03-components.md": acView("C4Component\nComponent(core, \"1 Core\", \"Go\")",
			"1 | id=core; type=component; parent=run; owns=src/,Tests/,tools/,generated/"),
		root + acCode: acView("classDiagram\nclass Result[\"1 Result\"]",
			"1 | id=result; type=symbol; parent=core; path=src/main.go; symbol=Result; evidence=static:src/main.go:2"),
		root + acJourney: acView("flowchart TB\nhost((1 Host))\nopen[2 Open]\nhost -->|3| open",
			"1 | ref=host", "2 | id=open; type=behavior; requirements=0001,0001.1",
			"3 | id=opens; type=relationship; from=host; to=open; kind=navigates; mode=required; evidence=declared"),
		"src/main.go":                    "package main\ntype Result struct {}\nfunc main() {}\n",
		"src/App.swift":                  "struct App {}\n",
		"Tests/AppTests.swift":           "import Foundation\n",
		"tools/check.sh":                 "#!/bin/sh\nexit 0\n",
		"generated/data.go":              "package generated\n",
		".gitattributes":                 "generated/* linguist-generated=true\n",
		"docs/requirements/0001-open.md": "# 0001 Open\n\n## Why\n\nI open the app.\n\n## Measurement\n\nopens\n-\ntries\n\nSource: #1\n\n## Requirements\n\n- 0001.1\n    - Given Host opens the app\n        - When it starts\n            - Then it opens\n",
	}
}

func acRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, acCheck)
}

func Test0070_1(t *testing.T) {
	t.Parallel()
	out, code := acRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, acCheck)
}

func Test0070_2(t *testing.T) {
	t.Parallel()
	acTargetCases(t)
	acC4Variants(t)
	acRelationships(t, false)
	for _, root := range []string{"architecture/", "docs/architecture/"} {
		files := acFixture(root)
		files["src/project.yml"] = "name: Example\ntargets:\n  App:\n    type: application\n    sources: [.]\n"
		files[root+"02-containers.md"] = strings.ReplaceAll(files[root+"02-containers.md"], "target=go:src", "target=xcode-project:src/project.yml:App")
		out, code := acRun(t, files)
		sees(t, out, code, 0)
		files[root+acContext] = acView("flowchart TB\nhost((21 Host))\napp[22 App]\nhost -->|23| app",
			"21 | id=host; type=actor; name=Host; aliases=Owner", "22 | id=app; type=system",
			"23 | id=uses-app; type=relationship; from=host; to=app; kind=uses; mode=allowed; evidence=declared")
		files[root+acCode] += "\n```mermaid\n%% architecture:legend\nclassDiagram\nclass Example[\"1 Example\"]\n```\n"
		out, code = acRun(t, files)
		sees(t, out, code, 0)
		passedWith(t, out, acCheck, "5")
	}
}

func Test0070_3(t *testing.T) {
	t.Parallel()
	files := acFixture("architecture/")
	delete(files, "architecture/"+acJourney)
	out, code := acRun(t, files)
	sees(t, out, code, 1, "architecture/01-system-context.md:1: missing architecture view 05-user-journey.md")
}

func Test0070_4(t *testing.T) {
	t.Parallel()
	acBadTargets(t)
	acRelationships(t, true)
	acBadRefType(t)
	cases := map[string]string{
		"id=host; type=symbol; parent=core":                      "duplicate ID host; first declared at architecture/01-system-context.md:10",
		"id=result; type=symbol; parent=lost":                    "unresolved ID lost",
		"id=result; type=unknown; parent=core":                   "unsupported node type",
		"id=result; type=symbol; parent=core; mystery=yes":       "unsupported contract field",
		"id=result; type=symbol; parent=core; name":              "malformed contract field",
		"id=result; type=symbol; parent=result":                  "parent cycle",
		"id=result; type=symbol; parent=core; target=go:missing": "unresolved target go:missing",
		"id=result; type=symbol; parent=core; type=actor":        "duplicate contract field",
	}
	for replacement, want := range cases {
		files := acFixture("architecture/")
		files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "id=result; type=symbol; parent=core", replacement)
		out, code := acRun(t, files)
		sees(t, out, code, 1, "architecture/04-code.md:8:", want)
	}
	files := acFixture("architecture/")
	files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "classDiagram", "sequenceDiagram")
	out, code := acRun(t, files)
	sees(t, out, code, 1, "architecture/04-code.md:1: unsupported Mermaid form")
}

func Test0070_5(t *testing.T) {
	t.Parallel()
	for _, claim := range []string{"", "; owns=src/"} {
		files := acFixture("architecture/")
		files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "id=result", "id=result"+claim)
		want := "overlapping ownership of src/main.go:1; first claimed at architecture/03-components.md:8"
		if claim == "" {
			files["extra.go"] = "package extra\n"
			want = "unowned product source extra.go:1"
		}
		out, code := acRun(t, files)
		sees(t, out, code, 1, "architecture/", want)
	}
}

func Test0070_6(t *testing.T) {
	t.Parallel()
	for _, proof := range []string{"static:src/main.go:900", "runtime:trace.json:1", "magic:src/main.go:1"} {
		files := acFixture("architecture/")
		files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "static:src/main.go:2", proof)
		out, code := acRun(t, files)
		sees(t, out, code, 1, "architecture/04-code.md:8:", "evidence")
	}
}

func Test0070_7(t *testing.T) {
	t.Parallel()
	files := acFixture("architecture/")
	files["architecture/"+acJourney] = strings.ReplaceAll(files["architecture/"+acJourney], "0001.1", "0001.9")
	out, code := acRun(t, files)
	sees(t, out, code, 1, "architecture/05-user-journey.md:11: unresolved requirement 0001.9")
	files = acFixture("architecture/")
	file := "docs/requirements/0001-open.md"
	files[file] = strings.ReplaceAll(files[file], "Given Host", "Because Host")
	out, code = acRun(t, files)
	sees(t, out, code, 1, file+":18: each acceptance is one Given")
}

func Test0070_8(t *testing.T) {
	t.Parallel()
	files := acFixture("architecture/")
	files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "static:src/main.go:2", "static:src/main.go:900")
	repo := example(t, "happyRepo", files)
	for _, changed := range []string{"README.md", "src/App.swift", "architecture/" + acContext} {
		out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, acCheck)
		sees(t, out, code, 1, "architecture/04-code.md:8: unresolved static evidence src/main.go:900")
	}
}
