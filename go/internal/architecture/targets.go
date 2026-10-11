package architecture

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/swift"
)

var swiftTarget = regexp.MustCompile(`\.(executableTarget|testTarget|target)\s*\(\s*name:`)
var yamlTarget = regexp.MustCompile(`^  ([A-Za-z][A-Za-z0-9_-]*):\s*$`)

// manifestTargets reads the supported declarative target forms. Later runtime
// checks decide which targets produce deployable products; no equality is
// asserted here between target names and the diagram's runtime nodes.
func (m *Model) manifestTargets(root string, files []string) {
	for _, file := range files {
		switch path.Base(file) {
		case "Package.swift":
			m.swiftTargets(root, file)
		case "project.yml", "project.yaml":
			m.yamlTargets(root, file)
		}
	}
	m.targetReferences()
}

func (m *Model) swiftTargets(root, file string) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		m.fail(Location{file, 1}, "cannot read target manifest: %v", err)
		return
	}
	code, literals := swift.Mask(string(body))
	for _, match := range swiftTarget.FindAllStringSubmatchIndex(code, -1) {
		at := Location{file, strings.Count(code[:match[0]], "\n") + 1}
		m.swiftTarget(code, match, literals, at)
	}
}

func (m *Model) swiftTarget(code string, match []int, literals []swift.Literal, at Location) {
	name, ok := swiftTargetName(code, match[1], literals)
	if !ok {
		m.fail(at, "SwiftPM target needs a literal name")
		return
	}
	category := "product"
	if code[match[2]:match[3]] == "testTarget" {
		category = "test"
	}
	m.manifestTarget("swift-package:"+at.Path+":"+name, category, at)
}

func swiftTargetName(code string, after int, literals []swift.Literal) (string, bool) {
	for _, literal := range literals {
		if literal.Start < after {
			continue
		}
		if strings.TrimSpace(code[after:literal.Start]) != "" {
			return "", false
		}
		name, err := strconv.Unquote(literal.Text)
		return name, err == nil && name != ""
	}
	return "", false
}

func (m *Model) yamlTargets(root, file string) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		m.fail(Location{file, 1}, "cannot read target manifest: %v", err)
		return
	}
	active := false
	for i, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(line) == "targets:" {
			active = true
			continue
		}
		active = m.yamlTargetLine(file, line, i+1, active)
	}
}

func (m *Model) yamlTargetLine(file, line string, number int, active bool) bool {
	if line != "" && !strings.HasPrefix(line, " ") {
		return false
	}
	if match := yamlTarget.FindStringSubmatch(line); active && match != nil {
		m.manifestTarget("xcode-project:"+file+":"+match[1], "product", Location{file, number})
	}
	return active
}

func (m *Model) manifestTarget(id, category string, at Location) {
	source := Source{Path: at.Path, Category: category, At: at}
	m.ownSource(&source, map[string]int{})
	m.addTarget(id, source)
}

func (m *Model) targetReferences() {
	for _, d := range m.Declarations {
		if d.Target != "" && !m.hasTarget(d.Target) {
			m.fail(d.At, "unresolved target %s; use a supported Go, SwiftPM, or XcodeGen target", d.Target)
		}
	}
}

func (m *Model) hasTarget(target string) bool {
	for _, targetEntry := range m.Targets {
		if targetEntry.ID == target {
			return true
		}
	}
	return false
}
