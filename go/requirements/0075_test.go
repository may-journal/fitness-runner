package requirements

import (
	"strings"
	"testing"
)

const (
	aoCheck    = "architecture-components"
	aoView     = acRoot + acComponent
	aoMain     = "src/cmd/service/main.go"
	aoStore    = "src/store/store.go"
	aoHome     = "App/Screens/HomeScreen.swift"
	aoEdge     = "Rel(entry, store, \"4\")"
	aoSaves    = "Saves through the store, see [source](../src/cmd/service/main.go#L3)."
	aoEntry    = "Component(entry, \"2 Entry\", \"Go\")\n"
	aoModel    = "Component(model, \"3 Model\", \"Swift\", \"App/Model\")\n"
	aoNoOwner  = ":1: no component in the components view names this source"
	aoNotDrawn = ", but " + aoView + " draws no "
	aoImports  = "package main\n\nimport \"example.com/app/src/store\"\n\nfunc main() { store.Save() }\n"
	aoScreen   = "import SwiftUI\n\nstruct HomeScreen {\n    let store = Store()\n}\n"
)

// aoComponents is the components view with the entry owning its folder,
// inside added to the service boundary, after added past the edge, and
// rows added past row 4. The diagram starts on line 4.
func aoComponents(inside, after string, rows ...string) string {
	diagram := "C4Component\nContainer_Boundary(service, \"1 Service (src/cmd/service)\") {\n" + aoEntry + "Component(store, \"3 Store\", \"Go\", \"src/store\")\n" + inside + "}\n" + aoEdge + after
	return acView(diagram, append([]string{"The service.", "`src/cmd/service` starts it.", "Keeps records.", aoSaves}, rows...)...)
}

// aoFixture is the Go architecture whose entry imports the store, so the
// drawn edge and its evidence on line 3 both hold.
func aoFixture() map[string]string {
	return with(acFixture(acRoot), map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.26\n",
		aoMain:   aoImports,
		aoStore:  "package store\n\n// Save writes one record.\nfunc Save() {}\n",
		aoView:   aoComponents("", ""),
	})
}

// aoNative is the native app with screens reading a model, a widget, and
// a notification extension, each under its container's boundary. The
// diagram starts on line 4; the model sits on line 7.
func aoNative() map[string]string {
	files := accNative()
	for _, gone := range []string{"App/App.swift", aoMain, aoStore, "src/store/store_test.go"} {
		files[gone] = ""
	}
	files[aoHome] = aoScreen
	files["App/Model/Store.swift"] = "import Foundation\n\nfinal class Store {\n    var records: [String] = []\n}\n"
	files[aoView] = acView("C4Component\nContainer_Boundary(main, \"1 App (App/)\") {\nComponent(ui, \"2 Screens\", \"Swift\", \"App/Screens\")\n"+aoModel+"}\nContainer_Boundary(widget, \"4 Widget\") {\nComponent(glance, \"5 Glance\", \"Swift\", \"Widget/\")\n}\nContainer_Boundary(notify, \"6 Notify\") {\nComponent(alert, \"7 Alert\", \"Swift\", \"Notify/\")\n}\nRel(ui, model, \"8\")",
		"The app.", "Screens.", "The model.", "The widget.", "Draws the glance.", "The notification extension.", "Posts alerts.", "Screens read the model.")
	return files
}

func aoRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, aoCheck)
}

// aoEdit replaces old with new in the components view of files.
func aoEdit(files map[string]string, old, new string) map[string]string {
	files[aoView] = strings.Replace(files[aoView], old, new, 1)
	return files
}

func Test0075_1(t *testing.T) {
	t.Parallel()
	out, code := aoRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, aoCheck)
}

func Test0075_12(t *testing.T) {
	t.Parallel()
	files := with(aoFixture(), map[string]string{
		"src/api/api.go":        "package api\n",
		"src/api/api.config.ts": "export const port = 8080;\n",
		"index.mjs":             "export const ready = true;\n",
		"rollup.config.mjs":     "export default { input: \"index.mjs\" };\n",
		".eslintrc.cjs":         "module.exports = {};\n",
		"package.json":          "{}\n",
	})
	out, code := aoRun(t, files)
	sees(t, out, code, 1, "src/api/api.go"+aoNoOwner, "src/api/api.config.ts"+aoNoOwner, "index.mjs"+aoNoOwner)
	if strings.Contains(out, "rollup.config.mjs") || strings.Contains(out, ".eslintrc") {
		t.Errorf("a root tool config needs no owner:\n%s", out)
	}
	files = aoEdit(files, "\"src/store\")", "\"src/store\", \"and src/api\")")
	files = aoEdit(files, aoEntry, "Component(entry, \"2 Entry\", \"Go\", \"Loads index.mjs, per package.json\")\n")
	files = aoEdit(files, "| `src/cmd/service` starts it. |", "| `src/cmd/service` starts it, per `package.json`. |")
	files = aoEdit(files, "| Keeps records. |", "| Keeps records, `index.mjs` and `missing.mjs` export them, per `package.json`. |")
	out, code = aoRun(t, files)
	sees(t, out, code, 0)
	passedWith(t, out, aoCheck, "1")
	out, code = fitness(t, example(t, "happyRepo", files), nil, acCheck)
	sees(t, out, code, 0)
	files["src/api/api.go"], files["src/api/api.config.ts"] = "", ""
	out, code = fitness(t, example(t, "happyRepo", files), nil, acCheck)
	sees(t, out, code, 1, aoView+":7: unresolved source path src/api")
}

func Test0075_3(t *testing.T) {
	t.Parallel()
	second := "\n```mermaid\nC4Component\nContainer_Boundary(jobs, \"5 Jobs\") {\nComponent(cache, \"6 Cache\", \"Go\", \"src/store\")\n}\n```\n\n| # | Description | Why |\n| --- | --- | --- |\n| 5 | Jobs. | Later. |\n| 6 | Cache. | Later. |\n"
	files := aoFixture()
	files[aoView] += second
	out, code := aoRun(t, files)
	sees(t, out, code, 1, aoStore+":1: components store at "+aoView+":7 and cache at "+aoView+":22 both name this source")
	files = aoEdit(files, "\"Go\", \"src/store\")\n}\n```", "\"Go\", \"src/cache\")\n}\n```")
	files["src/cache/cache.go"] = "package cache\n"
	out, code = aoRun(t, files)
	sees(t, out, code, 0)
	passedWith(t, out, aoCheck, "1")
}

func Test0075_4(t *testing.T) {
	t.Parallel()
	out, code := aoRun(t, aoFixture())
	sees(t, out, code, 0)
	passedWith(t, out, aoCheck, "1")
	want := aoMain + ":3: entry imports src/store, owned by store" + aoNotDrawn + "entry -> store edge"
	out, code = aoRun(t, aoEdit(aoFixture(), aoEdge, ""))
	sees(t, out, code, 1, want)
	out, code = aoRun(t, aoEdit(aoFixture(), aoEdge, "Rel(store, entry, \"4\")"))
	sees(t, out, code, 1, want+"; "+aoView+":9 draws store -> entry")
	out, code = aoRun(t, aoNative())
	sees(t, out, code, 0)
	out, code = aoRun(t, aoEdit(aoNative(), "Rel(ui, model, \"8\")", ""))
	sees(t, out, code, 1, aoHome+":4: ui names Store, owned by model"+aoNotDrawn+"ui -> model edge")
}

func Test0075_5(t *testing.T) {
	t.Parallel()
	row := aoView + ":17: "
	cases := []struct{ old, new, want string }{
		{"cmd/service/main.go#L3)", "store/store.go#L1)", "evidence src/store/store.go#L1 is owned by store, not entry"},
		{"#L3)", "#L1)", "evidence src/cmd/service/main.go#L1 names nothing store owns"},
		{"../src/cmd/service/main.go#L3)", "../tools/check.sh)", "unsupported evidence tools/check.sh; Go and Swift sources only"},
	}
	for _, c := range cases {
		out, code := aoRun(t, aoEdit(aoFixture(), c.old, c.new))
		sees(t, out, code, 1, row+c.want)
	}
	files := aoFixture()
	files[aoView] = aoComponents("", "\nSystem_Ext(mailer, \"5 Mailer\", \"SMTP relay\")\nRel(entry, mailer, \"6\")", "Relays mail.", "Sends through the relay, see [source](../src/cmd/service/main.go#L7).")
	files[aoMain] = aoImports + "\nfunc send(c rune) bool { return c == '\"' } // smtp relay\n"
	out, code := aoRun(t, files)
	sees(t, out, code, 1, aoView+":21: evidence src/cmd/service/main.go#L7 does not name Mailer outside comments")
	files[aoMain] = aoImports + "var body = `raw\n` // smtp relay\n"
	out, code = aoRun(t, files)
	sees(t, out, code, 1, aoView+":21: evidence src/cmd/service/main.go#L7 does not name Mailer outside comments")
	files[aoMain] = aoImports + "\nfunc send() string { return \"https://smtp\" }\n"
	out, code = aoRun(t, files)
	sees(t, out, code, 0)
	renamed := aoEdit(aoFixture(), "#L3)", "#L5)")
	renamed[aoStore] = "package records\n\nfunc Save() {}\n"
	renamed["src/store/tools.go"] = "//go:build ignore\n\npackage main\n"
	renamed[aoMain] = strings.Replace(aoImports, "store.Save()", "records.Save()", 1)
	out, code = aoRun(t, renamed)
	sees(t, out, code, 0)
	renamed[aoMain] = strings.Replace(aoImports, "store.Save()", "_ = 1", 1)
	out, code = aoRun(t, renamed)
	sees(t, out, code, 1, row+"evidence src/cmd/service/main.go#L5 names nothing store owns")
}

// aoWorker adds a worker command whose job reads the store to the Go
// architecture. The service boundary opens on line 5; the worker boundary
// opens on line 10, or line 11 once the store moves to a shared boundary.
func aoWorker() map[string]string {
	files := aoFixture()
	files[acRoot+acContainer] = strings.Replace(files[acRoot+acContainer], "}\nRel", "Container(worker, \"5 Worker\", \"Go\", \"src/cmd/worker\")\n}\nRel", 1)
	files[aoView] = aoComponents("", "\nContainer_Boundary(worker, \"5 Worker (src/cmd/worker)\") {\nComponent(job, \"6 Job\", \"Go\", \"src/cmd/worker\")\n}\nRel(job, store, \"7\")", "The worker.", "Runs jobs.", "Reads records.")
	files["src/cmd/worker/main.go"] = aoImports
	return files
}

func Test0075_6(t *testing.T) {
	t.Parallel()
	out, code := aoRun(t, aoWorker())
	sees(t, out, code, 1, "src/cmd/worker/main.go:3: Go command worker reaches store here, but store is under container service at "+aoView+":5, not worker; a component two runtimes share belongs outside every container boundary")
	shared := aoEdit(aoWorker(), "Component(store, \"3 Store\", \"Go\", \"src/store\")\n}", "}\nSystem_Boundary(shared, \"Shared\") {\nComponent(store, \"3 Store\", \"Go\", \"src/store\")\n}")
	out, code = aoRun(t, shared)
	sees(t, out, code, 0)
	passedWith(t, out, aoCheck, "1")
	moved := aoEdit(aoEdit(shared, aoEntry, ""), "Component(job,", aoEntry+"Component(job,")
	out, code = aoRun(t, moved)
	sees(t, out, code, 1, aoMain+":1: Go command service is built from entry, which is under container worker at "+aoView+":11, not service")
	files := aoEdit(aoNative(), "Rel(ui, model, \"8\")", "Rel(ui, model, \"8\")\nRel(glance, model, \"9\")")
	files["Widget/Widget.swift"] = "import WidgetKit\n\nlet shared = Store()\n"
	out, code = aoRun(t, files)
	sees(t, out, code, 1, "Widget/Widget.swift:3: extension target Widget reaches model here, but model is under container main at "+aoView+":5, not widget")
	out, code = aoRun(t, aoEdit(files, aoModel+"}", "}\nSystem_Boundary(shared, \"Shared\") {\n"+aoModel+"}"))
	sees(t, out, code, 0)
}

// aoResolved is the native app with a sync component beside the model:
// each declares a load overload and a save extension member on the
// model's store. The screen loads through the model's overload, saves
// through both extensions. The model's save, past alternative #if headers
// and a < operator, calls the sync one from an if body after a closure.
func aoResolved(line7 string) map[string]string {
	files := aoEdit(aoNative(), aoModel+"}", aoModel+"Component(sync, \"9 Sync\", \"Swift\", \"App/Sync\")\n}")
	files = aoEdit(files, "Rel(ui, model, \"8\")", "Rel(ui, model, \"8\")\nRel(sync, model, \"10\")")
	files[aoView] += "| 9 | Syncs records. | Shared by later checks. |\n| 10 | Sync fills the store. | Shared by later checks. |\n"
	files["App/Model/Load.swift"] = "func load(from path: String) -> Store {\n    Store()\n}\n"
	files["App/Model/Saving.swift"] = "#if os(iOS)\nextension Store {\n#else\nextension Store: Sendable {\n#endif\n    static func < (lhs: Store, rhs: Store) -> Bool { true }\n" +
		"    func save(_ record: String) {\n        if let kept = [record].first(where: { !$0.isEmpty }) { save(all: [kept]) }\n    }\n}\n"
	files["App/Sync/Load.swift"] = "func load(into store: Store) {\n    store.records = []\n}\n"
	files["App/Sync/Saving.swift"] = "extension Store {\n    func save(all records: [String]) {}\n}\n"
	files[aoHome] = "import SwiftUI\n\nstruct HomeScreen {\n    func refresh(save: Bool) {\n        let store = load(from: \"records\")\n        store.save(\"first\")\n" + line7 + "    }\n}\n"
	return files
}

func Test0075_11(t *testing.T) {
	t.Parallel()
	out, code := aoRun(t, aoResolved("        if save { store.save(all: []) }\n"))
	sees(t, out, code, 1, aoHome+":7: ui names Store.save(all:), owned by sync"+aoNotDrawn+"ui -> sync edge",
		"App/Model/Saving.swift:8: model names Store.save(all:), owned by sync"+aoNotDrawn+"model -> sync edge")
	out, code = aoRun(t, aoResolved("        if save { load(into: store) }\n"))
	sees(t, out, code, 1, aoHome+":7: ui names load(into:), owned by sync"+aoNotDrawn+"ui -> sync edge")
	files := aoResolved("        func load(into store: Store) {}\n        if save { load(into: store) }\n")
	files["App/Model/Saving.swift"] = "extension Store {\n    func save(_ record: String) {}\n}\n"
	out, code = aoRun(t, files)
	sees(t, out, code, 0)
	passedWith(t, out, aoCheck, "1")
}

func Test0075_7(t *testing.T) {
	t.Parallel()
	files := with(aoFixture(), map[string]string{
		"src/api/api.go":           "package api\n\nfunc Ping() {}\n",
		"src/cmd/service/linux.go": "//go:build linux\n\npackage main\n\nimport \"example.com/app/src/api\"\n\nfunc ping() { api.Ping() }\n",
	})
	api := "Component(api, \"5 API\", \"Go\", \"src/api\")\n"
	files[aoView] = aoComponents(api, "", "Answers pings.")
	out, code := aoRun(t, files)
	sees(t, out, code, 1, "src/cmd/service/linux.go:5: entry imports src/api under go:build linux, owned by api"+aoNotDrawn+"entry -> api edge")
	files[aoView] = aoComponents(api, "\nRel(entry, api, \"6\")", "Answers pings.", "Pings the API.")
	out, code = aoRun(t, files)
	sees(t, out, code, 0)
	native := aoEdit(aoNative(), "Rel(ui, model, \"8\")", "")
	native[aoHome] = "import SwiftUI\n\nstruct HomeScreen {\n#if DEBUG // tools\n    let store = Store()\n#endif\n}\n"
	out, code = aoRun(t, native)
	sees(t, out, code, 1, aoHome+":5: ui names Store under #if DEBUG, owned by model"+aoNotDrawn+"ui -> model edge")
	native[aoHome] = "import SwiftUI\n\nstruct HomeScreen {\n#if DEBUG\n    let debug = true\n#endif // DEBUG\n    let store = Store()\n}\n"
	out, code = aoRun(t, native)
	sees(t, out, code, 1, aoHome+":7: ui names Store, owned by model"+aoNotDrawn+"ui -> model edge")
	native[aoView] = aoNative()[aoView]
	out, code = aoRun(t, native)
	sees(t, out, code, 0)
}
