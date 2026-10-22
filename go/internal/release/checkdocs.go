package release

import (
	"fmt"
	"go/ast"
	"go/doc"
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
	return checkDoc{Name: name, Folder: folder, Summary: pkg.Synopsis(capitalize(rest))}, nil
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
