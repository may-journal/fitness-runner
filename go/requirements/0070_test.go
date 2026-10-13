package requirements

import (
	"strconv"
	"strings"
	"testing"
)

const (
	acCheck     = "architecture-contract"
	acRoot      = "architecture/"
	acContext   = "01-system-context.md"
	acContainer = "02-containers.md"
	acComponent = "03-components.md"
	acCode      = "04-code.md"
	acJourney   = "05-user-journey.md"
	acEvidence  = "#L3)"
)

// acView writes a view as people already do: a heading, a diagram, and its
// callout table. Nothing is added for the check. The diagram starts on line
// 4, so its Nth line is line N+3; the first row sits on line count+8.
func acView(diagram string, rows ...string) string {
	body := "# View\n\n```mermaid\n" + diagram + "\n```\n\n| # | Description | Why |\n| --- | --- | --- |\n"
	for i, row := range rows {
		body += "| " + strconv.Itoa(i+1) + " | " + row + " | Shared by later checks. |\n"
	}
	return body
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

func acRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, acCheck)
}

// acEdit replaces old with new in one view of the default fixture.
func acEdit(view, old, new string) map[string]string {
	files := acFixture(acRoot)
	files[acRoot+view] = strings.Replace(files[acRoot+view], old, new, 1)
	return files
}

func Test0070_1(t *testing.T) {
	t.Parallel()
	out, code := acRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, acCheck)
}

func Test0070_2(t *testing.T) {
	t.Parallel()
	for _, root := range []string{acRoot, "docs/architecture/"} {
		files := acFixture(root)
		out, code := acRun(t, files)
		sees(t, out, code, 0)
		passedWith(t, out, acCheck, "5")
		files[root+acContext] = strings.NewReplacer("\"1 User\"", "\"21 User\"", "\"2 App\"", "\"22 App\"", "\"3\"", "\"23\"", "| 1 |", "| 21 |", "| 2 |", "| 22 |", "| 3 |", "| 23 |").Replace(files[root+acContext])
		out, code = acRun(t, files)
		sees(t, out, code, 0)
		files[root+acContext] = acView("---\ntitle: Context\n---\nflowchart LR\naccTitle: Context\nuser((1 User)):::persona\nUpdateApp[2 App]:::system\nuser -->|3| UpdateApp\nclassDef persona fill:#fff", "A person.", "The system.", "Uses it.")
		out, code = acRun(t, files)
		sees(t, out, code, 0)
	}
}

func Test0070_3(t *testing.T) {
	t.Parallel()
	files := acFixture(acRoot)
	delete(files, acRoot+acJourney)
	out, code := acRun(t, files)
	sees(t, out, code, 1, "architecture/01-system-context.md:1: missing architecture view 05-user-journey.md")
	files = acFixture(acRoot)
	files[acRoot+acCode] = "# View\n\nNo diagram yet.\n"
	out, code = acRun(t, files)
	sees(t, out, code, 1, "architecture/04-code.md:1: view has no diagram outside its legends")
}

func Test0070_4(t *testing.T) {
	t.Parallel()
	cases := []struct{ view, old, new, want string }{
		{acContext, "System(app, \"2 App\")", "System(user, \"2 App\")", "architecture/01-system-context.md:6: duplicate ID user; first at architecture/01-system-context.md:5"},
		{acContext, "System(app,", "System(2app,", "architecture/01-system-context.md:6: invalid ID \"2app\""},
		{acContext, "Rel(user, app,", "Rel(user, ghost,", "architecture/01-system-context.md:7: unresolved ID ghost"},
		{acCode, "save --> Record", "save --> Missing", "architecture/04-code.md:9: unresolved ID Missing"},
		{acComponent, "Container_Boundary(service, \"1 Service (src/cmd/service)\") {\n", "", "architecture/03-components.md:5: component entry needs an enclosing boundary"},
		{acCode, "classDiagram", "sequenceDiagram", "architecture/04-code.md:3: unsupported Mermaid form \"sequenceDiagram\""},
		{acContext, "Rel(user, app, \"3\")", "Rel(user, app, \"3\")\nShape(user, \"9\")", "architecture/01-system-context.md:8: unsupported C4 macro \"Shape\""},
		{acContainer, "Person(user, \"1 User\")", "Person(user, \"1 Customer\")", "architecture/02-containers.md:5: actor user is named \"Customer\" here and \"User\" at architecture/01-system-context.md:5"},
		{acContainer, "System_Boundary", "Person(member, \"5 User\")\nSystem_Boundary", "architecture/02-containers.md:6: actor name \"User\" is ambiguous: member here and user at architecture/01-system-context.md:5"},
	}
	for _, c := range cases {
		out, code := acRun(t, acEdit(c.view, c.old, c.new))
		sees(t, out, code, 1, c.want)
	}
}

func Test0070_5(t *testing.T) {
	t.Parallel()
	out, code := acRun(t, acEdit(acComponent, "Component(entry, \"2 Entry\", \"Go\")", "Component(entry, \"2 Entry\", \"Go\", \"src/store/store.go\")"))
	sees(t, out, code, 1, "architecture/03-components.md:7: overlapping source claims: store claims src/store and entry claims src/store/store.go at architecture/03-components.md:6")
	out, code = acRun(t, acEdit(acComponent, "\"src/store\"", "\"src/stores\""))
	sees(t, out, code, 1, "architecture/03-components.md:7: unresolved source path src/stores")
}

func Test0070_6(t *testing.T) {
	t.Parallel()
	for anchor, want := range map[string]string{"#L900)": "src/cmd/service/main.go#L900", "#line3)": "src/cmd/service/main.go#line3", "#L3-L1)": "src/cmd/service/main.go#L3-L1"} {
		out, code := acRun(t, acEdit(acComponent, acEvidence, anchor))
		sees(t, out, code, 1, "architecture/03-components.md:17: unresolved evidence "+want)
	}
	out, code := acRun(t, acEdit(acComponent, "service/main.go"+acEvidence, "service/gone.go"+acEvidence))
	sees(t, out, code, 1, "architecture/03-components.md:17: unresolved evidence src/cmd/service/gone.go#L3")
}

func Test0070_7(t *testing.T) {
	t.Parallel()
	out, code := acRun(t, acEdit(acJourney, "`0001.1`", "`0001.9`"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:28: unresolved requirement 0001.9 in docs/requirements")
	out, code = acRun(t, acEdit(acJourney, "0001-open.md", "0002-open.md"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:29: unresolved requirement 0002 in docs/requirements")
	files := acFixture(acRoot)
	files[acDoc] = strings.ReplaceAll(files[acDoc], "Source: #1\n", "")
	out, code = acRun(t, files)
	sees(t, out, code, 0)
	if strings.Contains(out, acDoc) {
		t.Errorf("doc shape belongs to the requirements check, got:\n%s", out)
	}
}

func Test0070_9(t *testing.T) {
	t.Parallel()
	drifts := []struct {
		check, want string
		files       map[string]string
		changed     []string
	}{
		{acCheck, "architecture/03-components.md:17: unresolved evidence src/cmd/service/main.go#L900", acEdit(acComponent, acEvidence, "#L900)"), []string{"README.md", "src/store/store.go", acRoot + acContext}},
		{accCheck, "src/cmd/api/main.go:1: Go command api" + accNoOwner, with(acFixture(acRoot), map[string]string{"src/cmd/api/main.go": "package main\n"}), []string{"README.md", "src/store/store.go", acRoot + acContainer}},
		{aoCheck, "src/api/api.go" + aoNoOwner, with(aoFixture(), map[string]string{"src/api/api.go": "package api\n"}), []string{"README.md", aoStore, aoView}},
	}
	for _, d := range drifts {
		repo := example(t, "happyRepo", d.files)
		for _, changed := range d.changed {
			out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, d.check)
			sees(t, out, code, 1, d.want)
		}
	}
}
