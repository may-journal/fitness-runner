package release

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

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

// firstSentence is the first sentence of a comment's first paragraph, on one
// line. It ends at a period, question or exclamation mark before a space.
func firstSentence(text string) string {
	text, _, _ = strings.Cut(text, "\n\n")
	text = strings.Join(strings.Fields(text), " ")
	for i := 0; i < len(text)-1; i++ {
		if strings.IndexByte(".!?", text[i]) >= 0 && text[i+1] == ' ' {
			return text[:i+1]
		}
	}
	return text
}
