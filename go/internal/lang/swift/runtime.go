// Package swift is the Swift language module's home: how a Swift repo
// proves a runtime and names its requirement tests.
//
//fitness:container cli
//fitness:component kit Its language's checks share it.
package swift

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Technology is the name a Swift product's container must give its
// language.
const Technology = "Swift"

// Ext is the Swift source file ending.
const Ext = ".swift"

// xcodeGenProject is the XcodeGen manifest that generates an Xcode project.
const xcodeGenProject = "project.yml"

// Runtime reads XcodeGen and Xcode projects for targets, and Swift product
// sources for platform services. An XcodeGen project.yml target of type
// application, *extension, or tool, its own or from targetTemplates, or a
// project.pbxproj native target of those types, proves an app, extension,
// or tool target whose container names Swift. A template source may name
// ${target_name}; other template attributes are not substituted. Xcode
// frameworks, libraries, and test bundles are source types, never
// products, and project.pbxproj is read even when marked
// linguist-generated. A product file importing CloudKit, StoreKit,
// HealthKit, or WeatherKit uses a platform service.
var Runtime = runtimes.Reader{
	Manifests: map[string]func(*runtimes.Finder, string){xcodeGenProject: xcodegen, "project.pbxproj": pbxproj},
	Sources:   map[string]func(*runtimes.Finder, string){Ext: platformServices},
}

// target is one XcodeGen or Xcode target while its manifest is read.
type target struct {
	name, kind         string
	sources, templates []string
	at                 runtimes.Location
	typed              bool
}

// targetKind maps a target type to the runtime it ships. Frameworks,
// libraries, and test bundles are source types and map to "".
func targetKind(t string) string {
	t = strings.TrimPrefix(t, "com.apple.product-type.")
	switch {
	case t == "application" || strings.HasPrefix(t, "application."):
		return "app target"
	case strings.Contains(t, "extension"):
		return "extension target"
	case t == "tool":
		return "tool target"
	}
	return ""
}

// emit records a finished target as a product when its type ships a
// runtime. A target without a type cannot be resolved.
func emit(f *runtimes.Finder, file string, t *target) {
	if !t.typed {
		f.Fail(t.at, "target %s has no type", t.name)
		return
	}
	if t.kind != "" {
		f.Products = append(f.Products, runtimes.Product{Kind: t.kind, Name: t.name, Technology: Technology, Paths: sourceFolders(f, file, t), At: t.at})
	}
}

func sourceFolders(f *runtimes.Finder, file string, t *target) []string {
	var folders []string
	for _, source := range t.sources {
		folder := runtimes.Beside(file, source)
		if !f.Exists(folder) {
			f.Fail(t.at, "target %s names a source %s that does not exist", t.name, folder)
			continue
		}
		folders = append(folders, folder)
	}
	return folders
}

var (
	yamlKey    = regexp.MustCompile(`^(\s*)([^\s:#-][^:]*):\s*(.*?)\s*$`)
	yamlItem   = regexp.MustCompile(`^(\s*)-\s*(?:path:\s*)?(.*?)\s*$`)
	nativeName = regexp.MustCompile(`^\s*name\s*=\s*"?([^";]+)"?\s*;`)
	nativeType = regexp.MustCompile(`^\s*productType\s*=\s*"?([^";]+)"?\s*;`)
)

// xcodegen reads the targets of an XcodeGen project.yml: each target's
// type, sources, and templates. A target takes the type and sources of
// the templates it names.
func xcodegen(f *runtimes.Finder, file string) {
	lines, ok := f.Lines(file)
	if !ok {
		return
	}
	r := &yamlTargets{file: file, templates: map[string]*target{}}
	for i, line := range lines {
		r.line(line, i+1)
	}
	r.finish()
	for _, t := range r.targets {
		r.resolve(t)
		emit(f, file, t)
	}
}

// yamlTargets reads the targets and targetTemplates sections by
// indentation: section keys at 0, target names at 2, fields at 4, and
// list items at the first item's indent.
type yamlTargets struct {
	file       string
	section    string
	current    *target
	list       *[]string
	listIndent int
	targets    []*target
	templates  map[string]*target
}

func (r *yamlTargets) line(line string, n int) {
	if comment(line) {
		return
	}
	if item := yamlItem.FindStringSubmatch(line); item != nil && r.list != nil {
		r.item(len(item[1]), item[2])
		return
	}
	if key := yamlKey.FindStringSubmatch(line); key != nil {
		r.key(len(key[1]), key[2], uncommented(key[3]), n)
	}
}

func comment(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "#")
}

// uncommented drops a trailing "# comment" and surrounding quotes; a value
// that is only a comment is empty.
func uncommented(value string) string {
	if strings.HasPrefix(value, "#") {
		return ""
	}
	value, _, _ = strings.Cut(value, " #")
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

// item adds a list entry at the list's own indent; deeper entries, such
// as a source's excludes, belong to it and are skipped.
func (r *yamlTargets) item(indent int, value string) {
	if r.listIndent < 0 {
		r.listIndent = indent
	}
	if indent == r.listIndent {
		*r.list = append(*r.list, uncommented(value))
	}
}

func (r *yamlTargets) key(indent int, name, value string, n int) {
	if indent <= 4 {
		r.list, r.listIndent = nil, -1
	}
	switch indent {
	case 0:
		r.finish()
		r.section = name
	case 2:
		r.target(name, n)
	case 4:
		r.field(name, value)
	}
}

// finish files the current target under its section.
func (r *yamlTargets) finish() {
	if r.current == nil {
		return
	}
	if r.section == "targetTemplates" {
		r.templates[r.current.name] = r.current
	} else {
		r.targets = append(r.targets, r.current)
	}
	r.current = nil
}

func (r *yamlTargets) target(name string, n int) {
	if r.section != "targets" && r.section != "targetTemplates" {
		return
	}
	r.finish()
	r.current = &target{name: name, at: runtimes.Location{Path: r.file, Line: n}}
}

func (r *yamlTargets) field(name, value string) {
	if r.current == nil {
		return
	}
	switch name {
	case "type":
		r.current.kind, r.current.typed = targetKind(value), true
	case "sources":
		r.listOf(&r.current.sources, value)
	case "templates":
		r.listOf(&r.current.templates, value)
	}
}

// listOf reads a flow sequence or one bare value now, or opens a block
// list for the items that follow.
func (r *yamlTargets) listOf(into *[]string, value string) {
	*into = append(*into, flowSequence(value)...)
	if value == "" {
		r.list, r.listIndent = into, -1
	}
}

// flowSequence reads "[a, b]" or one bare path; an empty value is a block
// list that follows.
func flowSequence(value string) []string {
	if value == "" {
		return nil
	}
	var paths []string
	for _, item := range strings.Split(strings.Trim(value, "[]"), ",") {
		if item = strings.Trim(strings.TrimSpace(item), `"'`); item != "" {
			paths = append(paths, item)
		}
	}
	return paths
}

// resolve gives a target the type and sources of the templates it names,
// with ${target_name} in a template source replaced by the target's name.
func (r *yamlTargets) resolve(t *target) {
	for _, name := range t.templates {
		tmpl := r.templates[name]
		if tmpl == nil {
			continue
		}
		if !t.typed {
			t.kind, t.typed = tmpl.kind, tmpl.typed
		}
		for _, source := range tmpl.sources {
			t.sources = append(t.sources, strings.ReplaceAll(source, "${target_name}", t.name))
		}
	}
}

// pbxproj reads the native targets of an Xcode project: a block holding
// isa = PBXNativeTarget with its name and productType.
func pbxproj(f *runtimes.Finder, file string) {
	if f.Exists(path.Join(path.Dir(path.Dir(file)), xcodeGenProject)) {
		return // XcodeGen generates this project; its project.yml is read instead
	}
	lines, ok := f.Lines(file)
	if !ok {
		return
	}
	var current *target
	for i, line := range lines {
		current = pbxLine(f, file, current, line, i+1)
	}
}

func pbxLine(f *runtimes.Finder, file string, current *target, line string, n int) *target {
	switch {
	case strings.Contains(line, "isa = PBXNativeTarget;"):
		return &target{at: runtimes.Location{Path: file, Line: n}}
	case current == nil:
		return nil
	case strings.TrimSpace(line) == "};":
		emit(f, file, current)
		return nil
	}
	pbxField(current, line, n)
	return current
}

func pbxField(t *target, line string, n int) {
	if match := nativeName.FindStringSubmatch(line); match != nil {
		t.name, t.at.Line = match[1], n
	}
	if match := nativeType.FindStringSubmatch(line); match != nil {
		t.kind, t.typed = targetKind(match[1]), true
	}
}
