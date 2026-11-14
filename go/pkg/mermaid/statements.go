package mermaid

import (
	"regexp"
	"slices"
	"strings"
)

// The statement forms the architecture model and the template palette both
// read, so the two never disagree on what a diagram declares.
var (
	// C4Macro is one C4 macro call: its name, its arguments, and a "{" when
	// it opens a boundary.
	C4Macro = regexp.MustCompile(`^([A-Za-z_]+)\((.*)\)\s*(\{)?$`)
	// ClassMember is a class diagram member line, such as `Record : +ID`,
	// naming its class.
	ClassMember = regexp.MustCompile(`^([A-Za-z_][\w-]*)\s*:`)
	// Edge is one flowchart or class diagram arrow: its source, arrow,
	// pipe label, target, and colon label.
	Edge = regexp.MustCompile(`^([A-Za-z_][\w-]*)\s*(?:"[^"]*"\s*)?(<-->|<\|--|<\|\.\.|<--|<\.\.|\*--|o--|--\*|--o|--x|--\|>|\.\.\|>|-->|-\.->|==>|\.\.>|---|--|\.\.)\s*(?:"[^"]*"\s*)?(?:\|([^|]*)\|\s*)?([A-Za-z_][\w-]*)\s*(?::\s*(.*))?$`)
)

// c4Elements are the C4 macros that declare an element or a boundary,
// each also in its _Ext form.
var c4Elements = strings.Fields("Person System SystemDb SystemQueue Container ContainerDb ContainerQueue Component ComponentDb ComponentQueue Boundary Enterprise_Boundary System_Boundary Container_Boundary")

// IsC4Element reports a C4 macro that declares an element or a boundary,
// as against a relationship, a style, or a layout macro.
func IsC4Element(name string) bool {
	return slices.Contains(c4Elements, strings.TrimSuffix(name, "_Ext"))
}
