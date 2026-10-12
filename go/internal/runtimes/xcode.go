package runtimes

import (
	"regexp"
	"strings"
)

// target is one XcodeGen or Xcode target while its manifest is read.
type target struct {
	name, kind string
	sources    []string
	at         Location
	typed      bool
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
func (f *finder) emit(file string, t *target) {
	if t == nil {
		return
	}
	if !t.typed {
		f.fail(t.at, "target %s has no type", t.name)
		return
	}
	if t.kind != "" {
		f.Products = append(f.Products, Product{Kind: t.kind, Name: t.name, Technology: "Swift", Paths: f.sourceFolders(file, t), At: t.at})
	}
}

func (f *finder) sourceFolders(file string, t *target) []string {
	var folders []string
	for _, source := range t.sources {
		folder := beside(file, source)
		if !f.exists[folder] {
			f.fail(t.at, "target %s names a source %s that does not exist", t.name, folder)
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
// type and sources, which are a list, a flow sequence, or one path.
func (f *finder) xcodegen(file string) {
	lines, ok := f.lines(file)
	if !ok {
		return
	}
	r := &yamlTargets{f: f, file: file}
	for i, line := range lines {
		r.line(line, i+1)
	}
	f.emit(file, r.current)
}

type yamlTargets struct {
	f         *finder
	file      string
	current   *target
	inTargets bool
	inSources bool
}

func (r *yamlTargets) line(line string, n int) {
	if comment(line) {
		return
	}
	if item := yamlItem.FindStringSubmatch(line); item != nil && r.inSources {
		r.current.sources = append(r.current.sources, item[2])
		return
	}
	if key := yamlKey.FindStringSubmatch(line); key != nil {
		r.key(len(key[1]), key[2], key[3], n)
	}
}

func comment(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "#")
}

// key reads a top-level key, a target name under targets, or a target
// field, by its indentation.
func (r *yamlTargets) key(indent int, name, value string, n int) {
	if indent <= 4 {
		r.inSources = false
	}
	switch indent {
	case 0:
		r.top(name)
	case 2:
		r.target(name, n)
	case 4:
		r.field(name, value)
	}
}

func (r *yamlTargets) top(name string) {
	r.f.emit(r.file, r.current)
	r.current, r.inTargets = nil, name == "targets"
}

func (r *yamlTargets) target(name string, n int) {
	if !r.inTargets {
		return
	}
	r.f.emit(r.file, r.current)
	r.current = &target{name: name, at: Location{r.file, n}}
}

func (r *yamlTargets) field(name, value string) {
	if r.current == nil {
		return
	}
	switch name {
	case "type":
		r.current.kind, r.current.typed = targetKind(value), true
	case "sources":
		r.inSources = value == ""
		r.current.sources = append(r.current.sources, flowSequence(value)...)
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

// pbxproj reads the native targets of an Xcode project: a block holding
// isa = PBXNativeTarget with its name and productType.
func (f *finder) pbxproj(file string) {
	lines, ok := f.lines(file)
	if !ok {
		return
	}
	var current *target
	for i, line := range lines {
		current = f.pbxLine(file, current, line, i+1)
	}
}

func (f *finder) pbxLine(file string, current *target, line string, n int) *target {
	switch {
	case strings.Contains(line, "isa = PBXNativeTarget;"):
		return &target{at: Location{file, n}}
	case current == nil:
		return nil
	case strings.TrimSpace(line) == "};":
		f.emit(file, current)
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
