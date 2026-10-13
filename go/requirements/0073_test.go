package requirements

import (
	"strings"
	"testing"
)

const (
	accCheck    = "architecture-containers"
	accNoOwner  = " has no container in the containers view"
	accProject  = "project.yml"
	accPackage  = "package.json"
	accPbxproj  = "App.xcodeproj/project.pbxproj"
	accDevTools = "{\n  \"name\": \"portal\",\n  \"private\": true,\n  \"bin\": null,\n  \"devDependencies\": {\n    \"prettier\": \"3.0.0\"\n  }\n}\n"
	accService  = "{\n  \"name\": \"portal\",\n  \"scripts\": {\n    \"start\": \"node server.js\"\n  }\n}\n"
)

// accRootCommand is a Go command at the repository root, which go build
// names after the module path's last element.
var accRootCommand = map[string]string{"go.mod": "module example.com/tool\n\ngo 1.26\n", "main.go": "package main\n"}

// accNative is a native app with Node development tools: XcodeGen declares
// an app, two extensions through a template, and a test bundle, and the
// view declares the three runtimes by path or by name. The Notify target
// starts on line 17 of project.yml.
func accNative() map[string]string {
	files := acFixture(acRoot)
	files["src/cmd/service/main.go"] = "package service\n\n// Service is a library.\n"
	files[acRoot+acContainer] = acView("C4Container\nPerson(user, \"1 User\")\nSystem_Boundary(app, \"2 App\") {\nContainer(main, \"3 App\", \"Swift 6, SwiftUI\", \"App/\")\nContainer(widget, \"4 Widget\", \"Swift\")\nContainer(notify, \"5 Notify\", \"Swift\")\n}\nRel(user, main, \"6\")",
		"Same person.", "The app.", "The app target.", "The widget target.", "The notification target.", "Opens it.")
	files[accProject] = "name: App\ntargetTemplates:\n  Extension:\n    type: app-extension\n    sources: [\"${target_name}\"]\ntargets:\n  App:\n    type: application # the app\n    platform: iOS\n    sources: [App]\n  Widget:\n    templates: [Extension]\n    sources: # folders\n      - path: Widget\n        excludes:\n          - \"**/*.md\"\n  Notify:\n    templates: [Extension]\n    sources: Notify\n  AppTests:\n    type: bundle.unit-test\n    platform: iOS\n    sources: [Tests]\n"
	files["App/App.swift"] = "import SwiftUI\n"
	files["Widget/Widget.swift"] = "import WidgetKit\n"
	files["Notify/Notify.swift"] = "import UserNotifications\n"
	files["Tests/AppTests.swift"] = "import XCTest\n"
	files[accPackage] = accDevTools
	return files
}

// accXcode is the native app with an Xcode project instead of XcodeGen,
// declaring the app and the widget.
func accXcode(targets ...[2]string) map[string]string {
	files := accNative()
	delete(files, accProject)
	files[acRoot+acContainer] = strings.Replace(files[acRoot+acContainer], "Container(notify, \"5 Notify\", \"Swift\")\n", "", 1)
	files[accPbxproj] = pbxproj(targets...)
	return files
}

// pbxproj renders native targets as Xcode writes them; an empty type leaves
// the productType line out.
func pbxproj(targets ...[2]string) string {
	body := "// !$*UTF8*$!\n{\n\tobjects = {\n"
	for _, t := range targets {
		body += "\t\tA /* " + t[0] + " */ = {\n\t\t\tisa = PBXNativeTarget;\n\t\t\tname = " + t[0] + ";\n"
		if t[1] != "" {
			body += "\t\t\tproductType = \"com.apple.product-type." + t[1] + "\";\n"
		}
		body += "\t\t};\n"
	}
	return body + "\t};\n}\n"
}

func accRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, accCheck)
}

// accEdit replaces old with new in the containers view of files.
func accEdit(files map[string]string, old, new string) map[string]string {
	files[acRoot+acContainer] = strings.Replace(files[acRoot+acContainer], old, new, 1)
	return files
}

func Test0073_1(t *testing.T) {
	t.Parallel()
	out, code := accRun(t, accNative())
	sees(t, out, code, 0)
	passedWith(t, out, accCheck, "1")
	out, code = accRun(t, acFixture(acRoot))
	sees(t, out, code, 0)
	generated := accEdit(accNative(), "\"3 App\"", "\"3 Main\"")
	generated[accPbxproj] = pbxproj([2]string{"App", "application"})
	out, code = accRun(t, generated)
	sees(t, out, code, 0)
	out, code = accRun(t, accEdit(accNative(), "Container(notify, \"5 Notify\", \"Swift\")\n", ""))
	sees(t, out, code, 1, "project.yml:17: extension target Notify"+accNoOwner)
}

func Test0073_2(t *testing.T) {
	t.Parallel()
	cases := []struct {
		files map[string]string
		want  string
	}{
		{map[string]string{"src/cmd/api/main.go": "package main\n"}, "src/cmd/api/main.go:1: Go command api" + accNoOwner},
		{map[string]string{accPackage: accService}, "package.json:4: Node service portal" + accNoOwner},
		{map[string]string{accPackage: "{\n  \"name\": \"portal\",\n  \"bin\": \"cli.js\"\n}\n"}, "package.json:3: Node service portal" + accNoOwner},
		{map[string]string{accPackage: accDevTools, "Dockerfile": "FROM node\n"}, "Dockerfile:1: Node service portal" + accNoOwner},
		{accRootCommand, "main.go:1: Go command tool" + accNoOwner},
	}
	for _, c := range cases {
		out, code := accRun(t, with(acFixture(acRoot), c.files))
		sees(t, out, code, 1, c.want)
	}
	out, code := accRun(t, accEdit(with(acFixture(acRoot), accRootCommand), "}\nRel", "Container(tool, \"5 Tool\", \"Go\")\n}\nRel"))
	sees(t, out, code, 0)
	out, code = accRun(t, with(acFixture(acRoot), map[string]string{accPackage: accDevTools, "src/gen/gen.go": "//go:build ignore\n\npackage main\n"}))
	sees(t, out, code, 0)
}

func Test0073_3(t *testing.T) {
	t.Parallel()
	want := "architecture/02-containers.md:8: container widget matches no build target, entry point, or deploy file by name"
	out, code := accRun(t, accXcode([2]string{"App", "application"}, [2]string{"Widget", "app-extension"}))
	sees(t, out, code, 0)
	out, code = accRun(t, accXcode([2]string{"App", "application"}))
	sees(t, out, code, 1, want)
	files := accNative()
	files[accProject] = strings.Replace(files[accProject], "  Widget:\n    templates: [Extension]\n    sources: # folders\n      - path: Widget\n        excludes:\n          - \"**/*.md\"\n", "", 1)
	out, code = accRun(t, files)
	sees(t, out, code, 1, want)
}

func Test0073_4(t *testing.T) {
	t.Parallel()
	out, code := accRun(t, acEdit(acContainer, "\"Go\", \"src/cmd/service\"", "\"Node\", \"src/cmd/service\""))
	sees(t, out, code, 1, "architecture/02-containers.md:7: container service says \"Node\" but Go command service at src/cmd/service/main.go:1 is Go")
	out, code = accRun(t, acEdit(acContainer, "\"Go\", \"src/cmd/service\"", "\"Go 1.26\", \"src/cmd/service\""))
	sees(t, out, code, 0)
}

func Test0073_5(t *testing.T) {
	t.Parallel()
	files := accEdit(acFixture(acRoot), "}\nRel", "Container(store, \"5 Store\", \"Go\", \"src/store\")\n}\nRel")
	out, code := accRun(t, files)
	sees(t, out, code, 1, "architecture/02-containers.md:8: container store claims only source layers (src/store): no build target, entry point, or deploy file is under them; show them in the component view")
	files["src/store/store.go"] = "package main\n"
	out, code = accRun(t, files)
	sees(t, out, code, 0)
}

func Test0073_6(t *testing.T) {
	t.Parallel()
	files := accEdit(acFixture(acRoot), "}\nRel", "ContainerDb(db, \"5 Records\", \"SQLite\")\n}\nRel")
	out, code := accRun(t, files)
	sees(t, out, code, 1, "architecture/02-containers.md:8: store db names no schema, configuration, or source path")
	out, code = accRun(t, accEdit(files, "\"SQLite\")", "\"SQLite\", \"src/store\")"))
	sees(t, out, code, 0)
	files = with(accNative(), map[string]string{"App/Sync.swift": "import CloudKit\n"})
	out, code = accRun(t, files)
	sees(t, out, code, 1, "App/Sync.swift:1: CloudKit is used here but no external system in the containers view names it")
	out, code = accRun(t, accEdit(files, "}\nRel", "}\nSystem_Ext(icloud, \"7 Sync\", \"Apple\", \"CloudKit records\")\nRel"))
	sees(t, out, code, 0)
}

func Test0073_7(t *testing.T) {
	t.Parallel()
	out, code := accRun(t, with(acFixture(acRoot), map[string]string{accPackage: "{\n  \"name\": \"portal\",\n"}))
	sees(t, out, code, 1, "package.json:1: cannot read package.json")
	files := accNative()
	files[accProject] = strings.Replace(files[accProject], "    templates: [Extension]\n    sources: Notify\n", "    sources: Notify\n", 1)
	out, code = accRun(t, files)
	sees(t, out, code, 1, "project.yml:17: target Notify has no type")
	files = accNative()
	files[accProject] = strings.Replace(files[accProject], "sources: Notify", "sources: Missing", 1)
	out, code = accRun(t, files)
	sees(t, out, code, 1, "project.yml:17: target Notify names a source Missing that does not exist")
	out, code = accRun(t, accXcode([2]string{"App", "application"}, [2]string{"Widget", ""}))
	sees(t, out, code, 1, "App.xcodeproj/project.pbxproj:11: target Widget has no type")
}
