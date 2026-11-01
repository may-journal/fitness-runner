package release

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// checkDoc is what the generator reads from one check package's code.
type checkDoc struct {
	// Name is the check name, the folder name after fitness-check-.
	Name string
	// Folder is the package folder under go/cmd.
	Folder string
	// Summary is the first sentence of the package doc comment, without its
	// "Command fitness-check-<name>" opening.
	Summary string
	// Body is the whole package doc comment as Markdown, without that
	// opening.
	Body string
	// Comments is how the check treats code comments, as its
	// checkkit.Describe declares.
	Comments string
}

// readCheckDoc parses the Go package in dir, leaving out tests, and reads
// its doc comment. A package without one, or whose comment does not open
// with "Command fitness-check-<name>", is an error, so no check ships
// without a summary.
func readCheckDoc(dir string) (checkDoc, error) {
	folder := filepath.Base(dir)
	name := strings.TrimPrefix(folder, "fitness-check-")
	pkg, err := parseDoc(dir)
	if err != nil {
		return checkDoc{}, err
	}
	opening := "Command " + folder + " "
	rest, ok := strings.CutPrefix(strings.TrimSpace(pkg.Doc), opening)
	if !ok {
		return checkDoc{}, fmt.Errorf("go/cmd/%s: the package doc comment must open with %q", folder, opening)
	}
	rest = capitalize(rest)
	comments, err := describedComments(dir, folder)
	if err != nil {
		return checkDoc{}, err
	}
	return checkDoc{Name: name, Folder: folder, Summary: synopsis(pkg, rest), Body: markdown(pkg, rest), Comments: comments}, nil
}

// commentTreatments maps each checkkit constant a check may declare to its
// value.
var commentTreatments = map[string]string{
	"CommentsJudged":        checkkit.CommentsJudged,
	"CommentsStripped":      checkkit.CommentsStripped,
	"CommentsNotApplicable": checkkit.CommentsNotApplicable,
}

// describedComments reads the Comments field of every checkkit.Describe
// literal in the package in dir. Each must set the same checkkit constant,
// so no check ships without one or declares two.
func describedComments(dir, folder string) (string, error) {
	files, err := parseSources(token.NewFileSet(), dir, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	var values []string
	for _, file := range files {
		values = append(values, describeLiterals(file)...)
	}
	if unique := slices.Compact(slices.Sorted(slices.Values(values))); len(unique) != 1 || unique[0] == "" {
		return "", fmt.Errorf("go/cmd/%s: every checkkit.Describe must set Comments to one of %s", folder, treatmentNames())
	}
	return values[0], nil
}

// treatmentNames lists the checkkit Comments constants, sorted and joined
// for an error message.
func treatmentNames() string {
	names := slices.Sorted(maps.Keys(commentTreatments))
	for i, name := range names {
		names[i] = "checkkit." + name
	}
	return strings.Join(names, ", ")
}

// describeLiterals lists the treatment each checkkit.Describe literal in
// file declares, with "" for a literal that declares none.
func describeLiterals(file *ast.File) []string {
	var values []string
	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.CompositeLit); ok && isSelector(lit.Type, "checkkit", "Describe") {
			values = append(values, describeComments(lit))
		}
		return true
	})
	return values
}

// describeComments is the treatment a checkkit.Describe literal sets in its
// Comments field, or "" when it sets no checkkit Comments constant.
func describeComments(lit *ast.CompositeLit) string {
	value := ""
	ast.Inspect(lit, func(n ast.Node) bool {
		if kv, ok := n.(*ast.KeyValueExpr); ok && isIdent(kv.Key, "Comments") {
			value = treatment(kv.Value)
		}
		return value == ""
	})
	return value
}

// treatment is the value of the checkkit Comments constant expr names, or
// "" when it names none.
func treatment(expr ast.Expr) string {
	if sel, ok := expr.(*ast.SelectorExpr); ok && isIdent(sel.X, "checkkit") {
		return commentTreatments[sel.Sel.Name]
	}
	return ""
}

// isSelector reports whether expr is pkg.name.
func isSelector(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && sel.Sel.Name == name
}

// synopsis is the first doc sentence as plain text, stripping brackets from
// links Go cuts before reading their definitions.
func synopsis(pkg *doc.Package, text string) string {
	summary := pkg.Synopsis(text)
	for _, def := range pkg.Parser().Parse(text).Links {
		summary = strings.ReplaceAll(summary, "["+def.Text+"]", def.Text)
	}
	return summary
}

// markdown renders doc comment text with level-2 headings and no IDs, since
// GitHub derives them, below the title a README puts above.
func markdown(pkg *doc.Package, text string) string {
	printer := pkg.Printer()
	printer.HeadingLevel = 2
	printer.HeadingID = func(*comment.Heading) string { return "" }
	return blocks(codeSpans(string(printer.Markdown(pkg.Parser().Parse(text)))))
}

// blocks fences the printer's code blocks so the prose checks mask them, and
// strips the indent the printer puts before a list marker.
func blocks(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "\t") {
			out = append(out, unindentItem(lines[i]))
			continue
		}
		end := codeEnd(lines, i)
		out = append(out, "```")
		for _, line := range lines[i:end] {
			out = append(out, strings.TrimPrefix(line, "\t"))
		}
		out = append(out, "```")
		i = end - 1
	}
	return strings.Join(out, "\n")
}

// codeEnd is the index past the code block at lines[start], after its last
// tab-indented line so trailing blanks stay out.
func codeEnd(lines []string, start int) int {
	end := start + 1
	for j := start + 1; j < len(lines); j++ {
		if strings.HasPrefix(lines[j], "\t") {
			end = j + 1
		} else if lines[j] != "" {
			break
		}
	}
	return end
}

// unindentItem drops the indent the printer puts before a list marker:
// two spaces before "-", one before a number.
func unindentItem(line string) string {
	if strings.HasPrefix(line, "  - ") || numberedItem.MatchString(line) {
		return strings.TrimLeft(line, " ")
	}
	return line
}

// numberedItem matches a numbered list item as the printer writes it.
var numberedItem = regexp.MustCompile(`^ [0-9]+\. `)

// codeSpans turns the backticks a doc comment writes around code into
// Markdown code spans, undoing the escapes the printer adds inside them. Indented
// code blocks keep their text.
func codeSpans(md string) string {
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") {
			continue
		}
		lines[i] = spanLine(line)
	}
	return strings.Join(lines, "\n")
}

// spanLine turns each pair of escaped backticks on one line into a code
// span. A last backtick with no partner stays escaped, so it opens no span.
func spanLine(line string) string {
	parts := strings.Split(line, "\\`")
	for j := 1; j+1 < len(parts); j += 2 {
		parts[j] = unescape(parts[j])
	}
	if len(parts)%2 == 1 {
		return strings.Join(parts, "`")
	}
	last := len(parts) - 1
	return strings.Join(parts[:last], "`") + "\\`" + parts[last]
}

// unescape drops the backslash before each escaped character in s.
func unescape(s string) string {
	var b strings.Builder
	escaped := false
	for _, r := range s {
		if r == '\\' && !escaped {
			escaped = true
			continue
		}
		escaped = false
		b.WriteRune(r)
	}
	return b.String()
}

// parseDoc reads the documentation of the one non-test package in dir.
func parseDoc(dir string) (*doc.Package, error) {
	fset := token.NewFileSet()
	files, err := parseSources(fset, dir, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s holds no Go package", dir)
	}
	return doc.NewFromFiles(fset, files, "github.com/may-journal/fitness-runner/go/cmd/"+filepath.Base(dir))
}

// parseSources parses each package source in dir in mode, such as only the
// package clause and doc comment.
func parseSources(fset *token.FileSet, dir string, mode parser.Mode) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []*ast.File
	for _, entry := range entries {
		if !packageSource(entry) {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, mode)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

// packageSource reports whether entry is a Go source file of the package,
// not a folder or a test.
func packageSource(entry os.DirEntry) bool {
	name := entry.Name()
	return !entry.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// capitalize upper-cases the first letter of s.
func capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
