package requirements

import (
	"strings"
	"testing"
)

const (
	cvCheck   = "architecture-code"
	cvView    = acRoot + acCode
	cvStore   = "src/store/store.go"
	cvModel   = "App/Model/Store.swift"
	cvHome    = "App/Screens/HomeScreen.swift"
	cvLink    = "in [store.go](../src/store/store.go)."
	cvSave    = "class save[\"2 store.Save(id string) Record\"]"
	cvEdge    = "save --> Record : 3"
	cvRecord  = "package store\n\n// Record is one saved record.\ntype Record struct {\n\tID string\n}\n\n"
	cvSaves   = "// Save writes one record.\nfunc Save(id string) Record {\n\treturn Record{ID: id}\n}\n"
	cvNoEdge  = "draws no "
	cvAllowIt = " edge that allows it"
)

// cvCode is the Go code view: a record type with its field and the save
// function returning it. The diagram starts on line 4, so the record sits
// on line 5, save on line 8, and the edge on line 9; extra lines and rows
// follow them.
func cvCode(extra string, rows ...string) string {
	diagram := "classDiagram\nclass Record[\"1 store.Record\"] {\n+ID\n}\n" + cvSave + "\n" + cvEdge + extra
	return acView(diagram, append([]string{"A saved record, " + cvLink, "Writes one record, " + cvLink, "Save returns the record."}, rows...)...)
}

// cvFixture is a Go repository whose code view matches its source.
func cvFixture() map[string]string {
	return with(acFixture(acRoot), map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.26\n",
		cvStore:  cvRecord + cvSaves,
		cvView:   cvCode(""),
	})
}

// cvNative is a Swift app whose code view shows a protocol, the store that
// conforms to it, and the screen that makes a store. The diagram starts on
// line 4: Saving on line 5, Store on 8, HomeScreen on 9, and the edges on
// lines 10 and 11.
func cvNative() map[string]string {
	files := acFixture(acRoot)
	files[cvModel] = "import Foundation\n\nprotocol Saving {\n    func save(_ id: String) -> Bool\n}\n\nfinal class Store: Saving {\n    func save(_ id: String) -> Bool { true }\n}\n"
	files[cvHome] = "import SwiftUI\n\nstruct HomeScreen {\n    let store = Store()\n}\n"
	files[cvView] = acView("classDiagram\nclass Saving[\"1 Saving\"] {\n<<interface>>\n}\nclass Store[\"2 Store\"]\nclass HomeScreen[\"3 HomeScreen\"]\nStore ..|> Saving : 4\nHomeScreen --> Store : 5",
		"Saves records, in [Store.swift](../App/Model/Store.swift).", "Keeps records, in [Store.swift](../App/Model/Store.swift).",
		"Lists records, in [HomeScreen.swift](../App/Screens/HomeScreen.swift).", "The store saves.", "The screen makes a store.")
	return files
}

func cvRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, cvCheck)
}

// cvEdit replaces old with new in one file of files.
func cvEdit(files map[string]string, file, old, new string) map[string]string {
	files[file] = strings.Replace(files[file], old, new, 1)
	return files
}

func Test0075_1(t *testing.T) {
	t.Parallel()
	out, code := cvRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, cvCheck)
}

func Test0075_2(t *testing.T) {
	t.Parallel()
	out, code := cvRun(t, cvFixture())
	sees(t, out, code, 0)
	passedWith(t, out, cvCheck, "1")
	moved := with(cvFixture(), map[string]string{cvStore: cvRecord, "src/store/save.go": "package store\n\n" + cvSaves})
	out, code = cvRun(t, moved)
	sees(t, out, code, 1, cvView+":8: save: store.Save(id string) Record is declared at src/store/save.go:4, not in the linked src/store/store.go")
	out, code = cvRun(t, cvEdit(moved, cvView, "Writes one record, in [store.go](../src/store/store.go)", "Writes one record, in [save.go](../src/store/save.go)"))
	sees(t, out, code, 0)
	renamed := cvEdit(cvFixture(), cvStore, "type Record struct", "type Entry struct")
	renamed = cvEdit(renamed, cvStore, "Record {\n\treturn Record{", "Entry {\n\treturn Entry{")
	out, code = cvRun(t, renamed)
	sees(t, out, code, 1, cvView+":5: Record: src/store/store.go declares no store.Record")
}

func Test0075_3(t *testing.T) {
	t.Parallel()
	cases := []struct{ file, old, new, want string }{
		{cvStore, "Save(id string) Record {", "Save(id string, force bool) Record {", ":8: save: store.Save at src/store/store.go:9 has signature (string, bool) Record, not store.Save(id string) Record"},
		{cvView, cvSave, "class save[\"2 store.Save\"]", ":8: save: store.Save at src/store/store.go:9 is a function, not a type"},
		{cvStore, "\tID string\n", "\tKey string\n", ":6: Record: store.Record at src/store/store.go:4 declares no field ID"},
		{cvView, "+ID\n", "+ID: int\n", ":6: Record: field store.Record.ID at src/store/store.go:5 is string, not int"},
		{cvView, "+ID\n", "+ID\n+Load() Record\n", ":7: Record: store.Record at src/store/store.go:4 declares no method Load"},
	}
	for _, c := range cases {
		out, code := cvRun(t, cvEdit(cvFixture(), c.file, c.old, c.new))
		sees(t, out, code, 1, cvView+c.want)
	}
	files := cvEdit(cvFixture(), cvView, "+ID\n", "+ID: string\n+Load() Record\n")
	files = cvEdit(files, cvStore, cvSaves, cvSaves+"\n// Load reads the record back.\nfunc (r Record) Load() Record { return r }\n")
	out, code := cvRun(t, files)
	sees(t, out, code, 0)
}

func Test0075_4(t *testing.T) {
	t.Parallel()
	files := cvEdit(cvFixture(), cvStore, cvSaves, cvSaves+"\n// Batch is many records.\ntype Batch struct{}\n\ntype batch struct{}\n")
	out, code := cvRun(t, files)
	sees(t, out, code, 1, cvStore+":14: store.Batch is an exported type in src/store, which "+cvView+" shows, but no node names it")
	files[cvView] = cvCode("\nclass Batch[\"4 store.Batch\"]", "Many records, "+cvLink)
	out, code = cvRun(t, files)
	sees(t, out, code, 0)
	files[cvView] = cvCode("\nclass again[\"4 store.Record\"]", "The record again, "+cvLink)
	out, code = cvRun(t, files)
	sees(t, out, code, 1, cvView+":10: again and Record at "+cvView+":5 both name store.Record")
}

func Test0075_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", cvEdit(cvFixture(), cvView, cvEdge, ""))
	for _, changed := range []string{"", "README.md", cvStore, cvView} {
		out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, cvCheck)
		sees(t, out, code, 1, cvStore+":9: save references Record, but "+cvView+" "+cvNoEdge+"save -> Record"+cvAllowIt)
	}
	files := cvEdit(cvFixture(), cvView, cvEdge, "class load[\"3 store.Load() string\"]")
	files = cvEdit(files, cvView, "Save returns the record.", "Loads a name, "+cvLink)
	files[cvStore] += "\n// Load names a Record in prose only.\nfunc Load() string { return \"Record\" } // Record\n"
	out, code := cvRun(t, files)
	sees(t, out, code, 1, cvStore+":9: save references Record")
	files[cvView] = strings.Replace(files[cvView], "class load", cvEdge+"\nclass load", 1)
	files[cvView] = strings.Replace(files[cvView], "| 3 | Loads a name, "+cvLink, "| 3 | Save returns the record. | Shared by later checks. |\n| 4 | Loads a name, "+cvLink, 1)
	files[cvView] = strings.Replace(files[cvView], "\"3 store.Load", "\"4 store.Load", 1)
	out, code = cvRun(t, files)
	sees(t, out, code, 0)
	native := cvEdit(cvNative(), cvView, "HomeScreen --> Store : 5", "")
	native[cvHome] = "import SwiftUI\n\nstruct HomeScreen {\n    let title = \"Store\" // Store\n    let store = Store()\n}\n"
	out, code = cvRun(t, native)
	sees(t, out, code, 1, cvHome+":5: HomeScreen calls Store, but "+cvView+" "+cvNoEdge+"HomeScreen -> Store"+cvAllowIt)
	out, code = cvRun(t, cvNative())
	sees(t, out, code, 0)
}

// cvSink adds a Go interface and a type that satisfies it, drawn as a
// realization on line 12.
func cvSink(put string) map[string]string {
	files := cvFixture()
	files[cvStore] += "\n// Sink takes records.\ntype Sink interface {\n\tPut()\n}\n\n// Disk writes records.\ntype Disk struct{}\n\n// " + put + " writes one.\nfunc (Disk) " + put + "() {}\n"
	files[cvView] = cvCode("\nclass Sink[\"4 store.Sink\"]\nclass Disk[\"5 store.Disk\"]\nDisk ..|> Sink : 6", "Takes records, "+cvLink, "Writes records, "+cvLink, "Disk is a sink.")
	return files
}

func Test0075_6(t *testing.T) {
	t.Parallel()
	out, code := cvRun(t, cvEdit(cvNative(), cvHome, "let store = Store()", "let count = 0"))
	sees(t, out, code, 1, cvView+":11: "+cvView+" draws association HomeScreen -> Store, but no source shows HomeScreen calls or references or embeds Store")
	out, code = cvRun(t, cvSink("Put"))
	sees(t, out, code, 0)
	out, code = cvRun(t, cvSink("Write"))
	sees(t, out, code, 1, cvView+":12: "+cvView+" draws realization Disk -> Sink, but no source shows Disk conforms to Sink")
	out, code = cvRun(t, cvEdit(cvNative(), cvModel, "final class Store: Saving {", "final class Store {"))
	sees(t, out, code, 1, cvView+":10: "+cvView+" draws realization Store -> Saving, but no source shows Store conforms to Saving")
	base := cvEdit(cvNative(), cvModel, "final class Store: Saving {", "class Base {}\n\nfinal class Store: Base, Saving {")
	base = cvEdit(base, cvView, "HomeScreen --> Store : 5", "HomeScreen --> Store : 5\nclass Base[\"6 Base\"]\nBase <|-- Store : 7")
	base = cvEdit(base, cvView, "| 5 | The screen makes a store. | Shared by later checks. |\n", "| 5 | The screen makes a store. | Shared by later checks. |\n| 6 | The base, in [Store.swift](../App/Model/Store.swift). | Shared. |\n| 7 | Store extends it. | Shared. |\n")
	out, code = cvRun(t, base)
	sees(t, out, code, 0)
	out, code = cvRun(t, cvEdit(base, cvModel, "Store: Base, Saving", "Store: Saving"))
	sees(t, out, code, 1, cvView+":13: "+cvView+" draws inheritance Store -> Base, but no source shows Store embeds or inherits from Base")
	linked := cvEdit(cvFixture(), cvView, "Save returns the record.", "Save returns the record, see [source](../src/store/store.go#L10).")
	out, code = cvRun(t, linked)
	sees(t, out, code, 0)
	out, code = cvRun(t, cvEdit(linked, cvView, "store.go#L10)", "store.go#L1)"))
	sees(t, out, code, 1, cvView+":16: evidence src/store/store.go#L1 shows no save -> Record relation")
}

func Test0075_7(t *testing.T) {
	t.Parallel()
	files := cvFixture()
	files[cvView] = cvCode("\nclass Contract[\"4 Save contract\"]\nsave --> Contract : 5", "What saving promises.", "Save keeps the promise.")
	out, code := cvRun(t, files)
	sees(t, out, code, 1, cvView+":10: Contract \"Save contract\" names no declaration; name a Go or Swift symbol, or mark it <<abstract>>")
	files[cvView] = cvCode("\nclass Contract[\"4 Save contract\"] {\n<<abstract>>\n+promise: one record\n}\nsave --> Contract : 5", "What saving promises.", "Save keeps the promise.")
	out, code = cvRun(t, files)
	sees(t, out, code, 0)
	files[cvView] = strings.Replace(files[cvView], "Save keeps the promise.", "Save keeps the promise, see [source](../src/store/store.go#L10).", 1)
	out, code = cvRun(t, files)
	sees(t, out, code, 1, cvView+":23: unsupported evidence src/store/store.go#L10; save -> Contract touches an abstract node, which no source declares")
}

func Test0075_8(t *testing.T) {
	t.Parallel()
	opener := "class opener[\"4 store.open()\"]"
	both := "Opens storage, in [linux](../src/store/open_linux.go) or [darwin](../src/store/open_darwin.go)."
	files := with(cvFixture(), map[string]string{
		"src/store/open_linux.go":  "//go:build linux\n\npackage store\n\nfunc open() {}\n",
		"src/store/open_darwin.go": "//go:build darwin\n\npackage store\n\nfunc open() {}\n",
	})
	files[cvView] = cvCode("\n"+opener, both)
	out, code := cvRun(t, files)
	sees(t, out, code, 1, cvView+":10: opener: store.open is declared 2 times, at src/store/open_darwin.go:5 under go:build darwin, src/store/open_linux.go:5 under go:build linux; link one file or line range, or state its signature")
	files[cvView] = cvCode("\n"+opener, "Opens storage, in [linux](../src/store/open_linux.go).")
	out, code = cvRun(t, files)
	sees(t, out, code, 0)
	native := cvNative()
	native[cvModel] = strings.Replace(native[cvModel], "    func save(_ id: String) -> Bool { true }\n", "#if DEBUG\n    func save(_ id: String) -> Bool { true }\n#else\n    func save(_ id: String) -> Bool { false }\n#endif\n    func save(_ ids: [String]) -> Bool { true }\n", 1)
	native[cvView] = strings.Replace(native[cvView], "HomeScreen --> Store : 5", "HomeScreen --> Store : 5\nclass saveOne[\"6 Store.save(String) Bool\"]\nclass saveAll[\"7 Store.save([String]) Bool\"]", 1)
	native[cvView] += "| 6 | Saves one, in [Store.swift](../App/Model/Store.swift#L7-L12). | Shared. |\n| 7 | Saves many, in [Store.swift](../App/Model/Store.swift). | Shared. |\n"
	out, code = cvRun(t, native)
	sees(t, out, code, 1, cvView+":12: saveOne: Store.save is declared 2 times, at "+cvModel+":9 under #if DEBUG, "+cvModel+":11 under #else of #if DEBUG")
	native[cvView] = strings.Replace(native[cvView], "Store.swift#L7-L12", "Store.swift#L8-L9", 1)
	out, code = cvRun(t, native)
	sees(t, out, code, 0)
}
