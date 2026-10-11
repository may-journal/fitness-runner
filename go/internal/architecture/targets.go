package architecture

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var swiftTarget = regexp.MustCompile(`\.(executableTarget|testTarget|target)\s*\(\s*name:\s*"([^"]+)"`)
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
	for _, match := range swiftTarget.FindAllStringSubmatchIndex(string(body), -1) {
		kind, name := string(body[match[2]:match[3]]), string(body[match[4]:match[5]])
		category := "product"
		if kind == "testTarget" {
			category = "test"
		}
		m.manifestTarget("swift-package:"+file+":"+name, category, Location{file, strings.Count(string(body[:match[0]]), "\n") + 1})
	}
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
	for _, source := range m.Sources {
		if source.Path == target {
			return true
		}
	}
	return false
}
