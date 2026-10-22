package release

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
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
	return checkDoc{Name: name, Folder: folder, Summary: pkg.Synopsis(rest), Body: markdown(pkg, rest)}, nil
}

// markdown renders doc comment text as Markdown, its headings at level 2,
// below the title an explanation puts above them.
func markdown(pkg *doc.Package, text string) string {
	printer := pkg.Printer()
	printer.HeadingLevel = 2
	printer.HeadingID = func(*comment.Heading) string { return "" }
	return codeSpans(string(printer.Markdown(pkg.Parser().Parse(text))))
}

// codeSpans turns the backticks a doc comment writes around code into
// Markdown code spans. The printer escapes them, since a doc comment has no
// code spans, and escapes punctuation inside them, which a code span would
// show. Indented code blocks keep their text.
func codeSpans(md string) string {
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") {
			continue
		}
		parts := strings.Split(line, "\\`")
		for j := 1; j < len(parts); j += 2 {
			parts[j] = unescape(parts[j])
		}
		lines[i] = strings.Join(parts, "`")
	}
	return strings.Join(lines, "\n")
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
	files, err := parseSources(fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s holds no Go package", dir)
	}
	return doc.NewFromFiles(fset, files, "github.com/may-journal/fitness-runner/go/cmd/"+filepath.Base(dir))
}

// parseSources parses each package source in dir, with its comments.
func parseSources(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []*ast.File
	for _, entry := range entries {
		if !packageSource(entry) {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.ParseComments)
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
