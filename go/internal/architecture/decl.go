package architecture

import (
	"fmt"
	"path"
	"strings"
)

// Declaration kinds the code view covers. Local bindings inside a function
// body are never declarations here.
const (
	kindType      = "type"
	kindInterface = "interface"
	kindFunc      = "function"
	kindMethod    = "method"
	kindField     = "field"
	kindValue     = "value"
)

// param is one parameter as the source declares it.
type param struct {
	name, typ string
}

// decl is one Go, Swift, or TypeScript declaration: a type, function, method, field, or
// top-level value. Owner is the enclosing or receiver type of a member.
// Package is the Go package name, empty otherwise. Condition is the build
// condition it sits under. Line and End bound it in its file.
type decl struct {
	File, Name, Owner, Kind, Package, Condition string
	Line, End                                   int
	Params                                      []param
	Results                                     []string
	Type                                        string
	// Exported marks a type the code view must show: a Go exported type,
	// a Swift public or open one, or an exported TypeScript class.
	Exported bool
	// Supers lists a Swift type's inheritance clause, or what a TypeScript
	// type extends, in order.
	Supers []string
	// Implements lists the interfaces a TypeScript class implements.
	Implements []string
	// Open is the line a TypeScript type's body opens on, which ends a
	// header split over lines.
	Open int
}

// full is the declaration's qualified name, such as checkkit.Result or
// Store.save.
func (d *decl) full() string {
	parts := []string{d.Package, d.Owner, d.Name}
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ".")
}

func (d *decl) at() Location { return Location{d.File, d.Line} }

// where names the declaration's location and any build condition.
func (d *decl) where() string {
	if d.Condition == "" {
		return d.at().String()
	}
	return d.at().String() + " under " + d.Condition
}

func (d *decl) folder() string { return path.Dir(d.File) }

func (d *decl) callable() bool { return d.Kind == kindFunc || d.Kind == kindMethod }

func (d *decl) typeLike() bool { return d.Kind == kindType || d.Kind == kindInterface }

// container reports a type or extension, whose body holds members.
func (d *decl) container() bool { return d.typeLike() || d.Kind == kindExtension }

// signature renders the parameter types and results as a label states
// them, such as (string, int) error.
func (d *decl) signature() string {
	types := make([]string, len(d.Params))
	for i, p := range d.Params {
		types[i] = p.typ
	}
	return strings.TrimSpace(fmt.Sprintf("(%s) %s", strings.Join(types, ", "), strings.Join(d.Results, ", ")))
}

// declIndex holds every declaration read from Go, Swift, and TypeScript
// sources.
type declIndex struct {
	decls  []*decl
	errors []string
}

func (x *declIndex) add(d *decl) *decl {
	x.decls = append(x.decls, d)
	return d
}

// named lists declarations with name, optionally only those owned by
// owner, in the given files; no files means every file.
func (x *declIndex) named(name, owner string, files []string) []*decl {
	var found []*decl
	for _, d := range x.decls {
		if d.Name == name && d.Owner == owner && inFiles(files, d.File) {
			found = append(found, d)
		}
	}
	return found
}

// members lists the members of a type declared in one Go package folder or
// TypeScript file, or anywhere for a Swift type, which extensions may add
// to elsewhere.
func (x *declIndex) members(t *decl) []*decl {
	var found []*decl
	for _, d := range x.decls {
		if d.Owner == t.Name && samePackage(d, t) {
			found = append(found, d)
		}
	}
	return found
}

func inFiles(files []string, file string) bool {
	return len(files) == 0 || contains(files, file)
}

// samePackage reports two Go declarations in one package folder, two
// Swift ones, or two TypeScript ones in one file, since a TypeScript class
// has no extensions.
func samePackage(d, t *decl) bool {
	if typeScript(d.File) || typeScript(t.File) {
		return d.File == t.File
	}
	return d.Package == t.Package && (t.Package == "" || d.folder() == t.folder())
}

func wheres(decls []*decl) string {
	places := make([]string, len(decls))
	for i, d := range decls {
		places[i] = d.where()
	}
	return strings.Join(places, ", ")
}
