package views

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
)

// identifier matches a name in a line of Swift code.
var identifier = regexp.MustCompile(`[A-Za-z_]\w*`)

func afterDot(code string, start int) bool {
	return start > 0 && code[start-1] == '.'
}

// relate reads each Swift node file with comments and literals stripped:
// inheritance clauses, then names of other Swift nodes.
func relate(g *codemodel.Graph, repo codemodel.Repo) {
	for _, file := range g.Files(func(file string) bool { return path.Ext(file) == swift.Ext }) {
		relateFile(g, file, repo.Read(file))
	}
}

func relateFile(g *codemodel.Graph, file string, lines []string) {
	sc := &codemodel.Scanner{Literals: true}
	for i, line := range lines {
		at := codemodel.Location{Path: file, Line: i + 1}
		code := sc.Line(line)
		if from := g.Owner(at); from != nil {
			relateLine(g, from, code, at)
		}
	}
}

func relateLine(g *codemodel.Graph, from *codemodel.Node, code string, at codemodel.Location) {
	if d := g.Header(at); d != nil {
		g.Supers(from, d, at)
		return
	}
	for _, span := range identifier.FindAllStringIndex(code, -1) {
		if to := target(g, code, span); to != nil {
			g.Observe(from, to, kind(code, span[1]), at)
		}
	}
}

// target resolves an identifier to the one node of that name, or
// Type.member to that member's node.
func target(g *codemodel.Graph, code string, span []int) *codemodel.Node {
	name := code[span[0]:span[1]]
	if !afterDot(code, span[0]) {
		return g.Unique(swift.Ext, name)
	}
	owner := identifier.FindAllString(code[:span[0]-1], -1)
	if len(owner) == 0 || !strings.HasSuffix(code[:span[0]-1], owner[len(owner)-1]) {
		return nil
	}
	return g.Unique(swift.Ext, owner[len(owner)-1]+"."+name)
}

func kind(code string, end int) string {
	if strings.HasPrefix(strings.TrimSpace(code[end:]), "(") {
		return codemodel.RelCalls
	}
	return codemodel.RelReferences
}

// conditions tracks the #if directives a line sits under.
type conditions struct {
	stack []string
}

// directive reads a conditional compilation line, comments already
// stripped, reporting whether the line was one.
func (c *conditions) directive(line string) bool {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "#if "):
		c.stack = append(c.stack, line)
	case strings.HasPrefix(line, "#elseif ") || line == "#else":
		c.replace(line)
	case line == "#endif":
		c.stack = c.stack[:max(len(c.stack)-1, 0)]
	default:
		return false
	}
	return true
}

func (c *conditions) replace(line string) {
	if len(c.stack) > 0 {
		c.stack[len(c.stack)-1] = line + " of " + c.stack[len(c.stack)-1]
	}
}

func (c *conditions) String() string {
	return strings.Join(c.stack, ", ")
}
