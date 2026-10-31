// Package codemodel is the language-neutral model the architecture views
// read code through: declarations, their members, the references between
// files, and the relations between declarations a code view shows. Each
// language module brings a Reader that fills it; this package and the views
// that use it know no language.
//
//fitness:container cli
//fitness:component kit Checks share it as one engine, so no check keeps its own copy.
package codemodel

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// Location is a one-based line in tracked repository content.
type Location struct {
	Path string
	Line int
}

func (l Location) String() string { return fmt.Sprintf("%s:%d", l.Path, l.Line) }

// Declaration kinds the code view covers. Local bindings inside a function
// body are never declarations here. An extension adds members and the
// types it inherits to a type declared elsewhere, but declares none
// itself.
const (
	KindType      = "type"
	KindInterface = "interface"
	KindFunc      = "function"
	KindMethod    = "method"
	KindField     = "field"
	KindValue     = "value"
	KindExtension = "extension"
)

// Kinds of relation a source shows between two declarations.
const (
	RelCalls      = "calls"
	RelReferences = "references"
	RelEmbeds     = "embeds"
	RelInherits   = "inherits from"
	RelConforms   = "conforms to"
)

// Param is one parameter as the source declares it.
type Param struct {
	Name, Type string
}

// Decl is one declaration: a type, function, method, field, or top-level
// value. Owner is the enclosing or receiver type of a member. Package is
// the namespace a label may qualify the name with, empty when the language
// has none. Scope groups the declarations whose members join: a member
// belongs to the type of its owner's name in the same scope. Condition is
// the build condition it sits under. Line and End bound it in its file.
type Decl struct {
	File, Name, Owner, Kind, Package, Scope, Condition string
	Line, End                                          int
	Params                                             []Param
	Results                                            []string
	Type                                               string
	// Exported marks a type the code view must show.
	Exported bool
	// Supers lists the types a type's header extends or inherits, in order.
	Supers []string
	// Implements lists the interfaces a type's header says it implements.
	Implements []string
	// Open is the line a type's body opens on, which ends a header split
	// over lines.
	Open int
	// Body is the byte offset of a type's opening brace, when its reader
	// reads headers by token. Only tokens before it read as the header.
	Body int
}

// Full is the declaration's qualified name, such as checkkit.Result or
// Store.save.
func (d *Decl) Full() string {
	var kept []string
	for _, part := range []string{d.Package, d.Owner, d.Name} {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ".")
}

// Key is the name its members give as their owner: Owner.Name for a
// nested type, so two nested types of one name keep their own members,
// else the name.
func (d *Decl) Key() string {
	if d.Owner == "" {
		return d.Name
	}
	return d.Owner + "." + d.Name
}

// At is the declaration's first line.
func (d *Decl) At() Location { return Location{d.File, d.Line} }

// Where names the declaration's location and any build condition.
func (d *Decl) Where() string {
	if d.Condition == "" {
		return d.At().String()
	}
	return d.At().String() + " under " + d.Condition
}

// Folder is the folder of the declaring file.
func (d *Decl) Folder() string { return path.Dir(d.File) }

// Callable reports a function or method.
func (d *Decl) Callable() bool { return d.Kind == KindFunc || d.Kind == KindMethod }

// TypeLike reports a type or interface.
func (d *Decl) TypeLike() bool { return d.Kind == KindType || d.Kind == KindInterface }

// Container reports a type or extension, whose body holds members.
func (d *Decl) Container() bool { return d.TypeLike() || d.Kind == KindExtension }

// Signature renders the parameter types and results as a label states
// them, such as (string, int) error.
func (d *Decl) Signature() string {
	types := make([]string, len(d.Params))
	for i, p := range d.Params {
		types[i] = p.Type
	}
	return strings.TrimSpace(fmt.Sprintf("(%s) %s", strings.Join(types, ", "), strings.Join(d.Results, ", ")))
}

// Index holds every declaration the readers read, in reading order, and
// the errors met while reading.
type Index struct {
	Decls  []*Decl
	Errors []string
}

// Add records a declaration and returns it.
func (x *Index) Add(d *Decl) *Decl {
	x.Decls = append(x.Decls, d)
	return d
}

// Named lists declarations with name, optionally only those owned by
// owner, in the given files; no files means every file.
func (x *Index) Named(name, owner string, files []string) []*Decl {
	var found []*Decl
	for _, d := range x.Decls {
		if d.Name == name && d.Owner == owner && inFiles(files, d.File) {
			found = append(found, d)
		}
	}
	return found
}

func inFiles(files []string, file string) bool {
	return len(files) == 0 || slices.Contains(files, file)
}

// Members lists the members of a type: the declarations its key owns in
// its scope.
func (x *Index) Members(t *Decl) []*Decl {
	var found []*Decl
	for _, d := range x.Decls {
		if d.Owner == t.Key() && d.Scope == t.Scope {
			found = append(found, d)
		}
	}
	return found
}

// Reference is one source line that reaches another path: an import of a
// folder, or a use of a symbol declared in a file. Condition is the build
// condition the line sits under, or empty.
type Reference struct {
	At                    Location
	To, Symbol, Condition string
}

// What says how the line reaches the other path.
func (r Reference) What() string {
	if r.Symbol != "" {
		return "names " + r.Symbol
	}
	return "imports " + r.To
}

// Under names the build condition the line sits under, if any.
func (r Reference) Under() string {
	if r.Condition == "" {
		return ""
	}
	return " under " + r.Condition
}
