// cspell:ignore Elts

package release

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// usageDirective marks a comment that documents one form of the fitness
// command line: `//fitness:usage <synopsis>`, after `fitness`. The rest of
// the comment says what that form does.
const usageDirective = "//fitness:usage"

// usageEntry is one documented form of the command line.
type usageEntry struct {
	// Synopses are the forms, after `fitness`, as the directives give them.
	Synopses []string
	// Summary is the first sentence of the comment.
	Summary string
}

// usageDoc is the documented command line: the forms that run checks, the
// subcommands in the order the subcommands table lists them, and the
// runner's package doc comment.
type usageDoc struct {
	Runner, Commands []usageEntry
	// Intro is the runner's package doc comment as Markdown.
	Intro string
}

// usageSources are the packages whose comments document the command line,
// under go/.
var usageSources = []string{"cmd/fitness", "internal/selection"}

// usage reads the documented command line. A subcommand whose table entry
// has no usage directive naming it is an error, so `fitness --help` and the
// llms page name every subcommand.
func (c Config) usage() (usageDoc, error) {
	var u usageDoc
	for _, rel := range usageSources {
		if err := u.read(filepath.Join(c.Root, "go", filepath.FromSlash(rel))); err != nil {
			return u, err
		}
	}
	slices.SortStableFunc(u.Runner, func(a, b usageEntry) int { return strings.Compare(a.Synopses[0], b.Synopses[0]) })
	return u, nil
}

// read adds the usage comments of the package in dir.
func (u *usageDoc) read(dir string) error {
	fset := token.NewFileSet()
	files, err := parseSources(fset, dir, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, file := range files {
		pkg := &doc.Package{Name: file.Name.Name}
		if err := u.readFile(fset, pkg, file); err != nil {
			return err
		}
	}
	return nil
}

// readFile sorts a file's usage comments into subcommands, by their place
// in the subcommands table, and forms that run checks.
func (u *usageDoc) readFile(fset *token.FileSet, pkg *doc.Package, file *ast.File) error {
	if pkg.Name == "main" && file.Doc != nil {
		u.Intro = markdown(pkg, strings.Replace(file.Doc.Text(), "Command fitness ", "fitness ", 1))
	}
	inTable, err := u.readTable(fset, pkg, file)
	if err == nil {
		u.readRunner(pkg, file, inTable)
	}
	return err
}

// readRunner adds each usage comment in file outside the subcommands
// table: the forms that run checks.
func (u *usageDoc) readRunner(pkg *doc.Package, file *ast.File, inTable map[*ast.CommentGroup]bool) {
	names := declNames(file)
	for _, group := range file.Comments {
		if e, ok := usageComment(pkg, group, names); ok && !inTable[group] {
			u.Runner = append(u.Runner, e)
		}
	}
}

// readTable reads each subcommand table entry's comment in file, in table
// order, and returns the comments it read.
func (u *usageDoc) readTable(fset *token.FileSet, pkg *doc.Package, file *ast.File) (map[*ast.CommentGroup]bool, error) {
	inTable := map[*ast.CommentGroup]bool{}
	table := subcommandTable(file)
	if table == nil {
		return inTable, nil
	}
	comments := ast.NewCommentMap(fset, table, file.Comments)
	for _, elt := range table.Elts {
		e, err := tableEntry(pkg, elt.(*ast.KeyValueExpr), comments, inTable)
		if err != nil {
			return nil, err
		}
		u.Commands = append(u.Commands, e)
	}
	return inTable, nil
}

// tableEntry reads one subcommand's comment, which must carry a usage
// directive for its name.
func tableEntry(pkg *doc.Package, kv *ast.KeyValueExpr, comments ast.CommentMap, inTable map[*ast.CommentGroup]bool) (usageEntry, error) {
	name := strings.Trim(kv.Key.(*ast.BasicLit).Value, `"`)
	for _, group := range comments[kv] {
		inTable[group] = true
		if e, ok := usageComment(pkg, group, nil); ok && strings.HasPrefix(e.Synopses[0], name) {
			return e, nil
		}
	}
	return usageEntry{}, fmt.Errorf("go/cmd/fitness: subcommand %q has no comment with a %q line naming it", name, usageDirective+" "+name)
}

// subcommandTable is the map literal assigned to subcommands in file, or
// nil.
func subcommandTable(file *ast.File) *ast.CompositeLit {
	var table *ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || !isIdent(assign.Lhs[0], "subcommands") {
			return table == nil
		}
		table, _ = assign.Rhs[0].(*ast.CompositeLit)
		return false
	})
	return table
}

// isIdent reports whether expr is the identifier name.
func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// declNames are the names a file declares at top level.
func declNames(file *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, obj := range file.Scope.Objects {
		names[obj.Name] = true
	}
	return names
}

// usageComment reads a comment's usage directives and the first sentence
// of its text, without the declared name it opens with. ok is false for a
// comment without a directive.
func usageComment(pkg *doc.Package, group *ast.CommentGroup, names map[string]bool) (e usageEntry, ok bool) {
	for _, line := range group.List {
		if rest, found := strings.CutPrefix(line.Text, usageDirective); found {
			e.Synopses = append(e.Synopses, strings.TrimSpace(rest))
		}
	}
	text := strings.TrimSpace(group.Text())
	if first, rest, cut := strings.Cut(text, " "); cut && names[first] {
		text = capitalize(rest)
	}
	e.Summary = pkg.Synopsis(text)
	return e, len(e.Synopses) > 0
}

// usageFile holds the documented command line and the llms page the
// fitness binary is built with, relative to the repository root.
const usageFile = "go/cmd/fitness/usage.go"

// llmsFile is the llms page at the repository root, which each release
// also ships.
const llmsFile = "llms.txt"

// usageSource lays out the usage file: the forms that run checks, the
// subcommands, and the llms page.
func usageSource(u usageDoc, page string) ([]byte, error) {
	return format.Source([]byte("// Code generated by `fitness-release checks`; DO NOT EDIT.\n\n" +
		"package main\n\n" +
		"// runnerUsage are the documented forms that run checks.\n" +
		"var runnerUsage = " + entriesSource(u.Runner) + "\n\n" +
		"// commandUsage are the documented subcommands, in table order.\n" +
		"var commandUsage = " + entriesSource(u.Commands) + "\n\n" +
		"// llmsPage is the page `fitness llms` prints and llms.txt holds.\n" +
		"const llmsPage = " + strconv.Quote(page) + "\n"))
}

// entriesSource writes entries as a Go slice literal.
func entriesSource(entries []usageEntry) string {
	var b strings.Builder
	b.WriteString("[]usageEntry{\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "\t{synopses: %#v, summary: %q},\n", e.Synopses, e.Summary)
	}
	return b.String() + "}"
}

// llmsPage is one page that teaches an AI the whole CLI: what fitness is,
// its exit codes, every command, every built-in check, and the docs topics.
func llmsPage(u usageDoc, checks []checkDoc, topics []topicDoc) string {
	var b strings.Builder
	b.WriteString("# fitness\n\n" + u.Intro)
	b.WriteString("\n## Run checks\n\n")
	writeUsage(&b, u.Runner)
	b.WriteString("\n## Commands\n\n")
	writeUsage(&b, u.Commands)
	b.WriteString("\n## Checks\n\n`fitness explain <check>` prints a check's whole documentation.\n\n")
	for _, d := range checks {
		fmt.Fprintf(&b, "- `%s`: %s\n", d.Name, d.Summary)
	}
	b.WriteString("\n## Topics\n\n")
	for _, t := range topics {
		fmt.Fprintf(&b, "- `fitness docs %s`: %s\n", t.Name, t.Summary)
	}
	return b.String()
}

// writeUsage lists each entry's forms and summary.
func writeUsage(b *strings.Builder, entries []usageEntry) {
	for _, e := range entries {
		forms := make([]string, len(e.Synopses))
		for i, s := range e.Synopses {
			forms[i] = "`" + strings.TrimSpace("fitness "+s) + "`"
		}
		fmt.Fprintf(b, "- %s: %s\n", strings.Join(forms, " or "), e.Summary)
	}
}
