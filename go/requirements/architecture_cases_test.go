package requirements

import (
	"strings"
	"testing"
)

const acContainer = "architecture/02-containers.md"
const acComponent = "architecture/03-components.md"

// acTargetCases exercises stable target IDs and source categories through the
// same installed check as the complete-contract acceptance.
func acTargetCases(t *testing.T) {
	t.Helper()
	for _, dir := range []string{"src/testimonials", "src/testimony.go", "src/contest"} {
		files := acFixture("architecture/")
		files[dir+"/main.go"] = "package main\nfunc main() {}\n"
		files[acContainer] = strings.ReplaceAll(files[acContainer], "target=go:src", "target=go:"+dir)
		out, code := acRun(t, files)
		sees(t, out, code, 0)
	}
	files := acSwiftFixture()
	files["src/Package.swift"] = "import PackageDescription\nlet url = \"https://example.test/a/*b*/\"\nlet package = Package(name: \"Example\", targets: [\n.executableTarget(/* note */ name: /* nested /* note */ */ \"Real\"), .testTarget(name: \"Checks\")])\n"
	out, code := acRun(t, files)
	sees(t, out, code, 0)
}

func acSwiftFixture() map[string]string {
	files := acFixture("architecture/")
	files[acContainer] = strings.ReplaceAll(files[acContainer], "target=go:src", "target=swift-package:src/Package.swift:Real")
	return files
}

func acBadTargets(t *testing.T) {
	t.Helper()
	for _, target := range []string{"src/main.go", "src/test/main.go", "go:src/test", "go:src/Tests"} {
		files := acFixture("architecture/")
		files["src/test/main.go"] = "package main\n"
		files["src/Tests/main.go"] = "package main\n"
		files[acContainer] = strings.ReplaceAll(files[acContainer], "target=go:src", "target="+target)
		out, code := acRun(t, files)
		sees(t, out, code, 1, "unresolved target "+target)
	}
	for _, text := range []string{
		"// .executableTarget(name: \"Real\")\n",
		"/* outer /* nested */ .executableTarget(name: \"Real\") */\n",
		"let note = #\".executableTarget(name: \"Real\")\"#\n",
		"let note = \"\"\"\n.executableTarget(name: \"Real\")\n\"\"\"\n",
		"let note = \"\\(\".executableTarget(name: \\\"Real\\\")\")\"\n",
	} {
		files := acSwiftFixture()
		files["src/Package.swift"] = "import PackageDescription\nlet package = Package(name: \"Example\", targets: [])\n" + text
		out, code := acRun(t, files)
		sees(t, out, code, 1, "unresolved target swift-package:src/Package.swift:Real")
	}
}

func acC4Variants(t *testing.T) {
	t.Helper()
	for _, prefix := range []string{"System", "Container", "Component"} {
		for _, suffix := range []string{"", "Db", "Queue", "_Ext", "Db_Ext", "Queue_Ext"} {
			files := acFixture("architecture/")
			files["architecture/"+acContext] = strings.ReplaceAll(files["architecture/"+acContext], "System(app,", prefix+suffix+"(app,")
			out, code := acRun(t, files)
			sees(t, out, code, 0)
		}
	}
}

// acRelationships reuses an edge declared in a later view, then checks that
// class arrow direction and table refs retain its semantic meaning.
func acRelationships(t *testing.T, bad bool) {
	t.Helper()
	for _, arrow := range []string{"-->", "<|--", "<|..", "<--", "<.."} {
		files := acFixture("architecture/")
		from, to := "host", "app"
		if strings.HasPrefix(arrow, "<") {
			from, to = to, from
		}
		files["architecture/"+acCode] = acView("classDiagram\nclass host[\"1 Host\"]\nclass app[\"2 App\"]\n"+from+" "+arrow+" "+to+" : 3",
			"1 | ref=host", "2 | ref=app", "3 | ref=uses-app")
		acForwardRef(files, arrow)
		if bad {
			files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], from+" "+arrow+" "+to, to+" "+arrow+" "+from)
		}
		out, code := acRun(t, files)
		acRelationshipResult(t, out, code, bad)
	}
}

func acForwardRef(files map[string]string, arrow string) {
	if arrow == "-->" {
		return
	}
	declaration := "id=uses-app; type=relationship; from=host; to=app; kind=uses; mode=allowed; evidence=declared"
	files["architecture/"+acContext] = strings.ReplaceAll(files["architecture/"+acContext], declaration, "ref=uses-app")
	files["architecture/"+acCode] = strings.ReplaceAll(files["architecture/"+acCode], "ref=uses-app", declaration)
}

func acRelationshipResult(t *testing.T, out string, code int, bad bool) {
	t.Helper()
	if bad {
		sees(t, out, code, 1, "relationship endpoints disagree")
		return
	}
	sees(t, out, code, 0)
}

func acBadRefType(t *testing.T) {
	t.Helper()
	files := acFixture("architecture/")
	files["architecture/"+acCode] = acView("classDiagram\nclass Thing[\"1 Thing\"]", "1 | ref=uses-app")
	out, code := acRun(t, files)
	sees(t, out, code, 1, "callout type disagrees")
}
