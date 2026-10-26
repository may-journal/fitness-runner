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
	"regexp"
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
	return checkDoc{Name: name, Folder: folder, Summary: synopsis(pkg, rest), Body: markdown(pkg, rest)}, nil
}

// synopsis is the first sentence of doc comment text as plain text. Go cuts
// the sentence before it reads the link definitions at the comment's end,
// so a link there would keep its brackets; synopsis shows its text alone.
func synopsis(pkg *doc.Package, text string) string {
	summary := pkg.Synopsis(text)
	for _, def := range pkg.Parser().Parse(text).Links {
		summary = strings.ReplaceAll(summary, "["+def.Text+"]", def.Text)
	}
	return summary
}

// markdown renders doc comment text as Markdown, its headings at level 2,
// with no IDs, which GitHub derives from the heading text itself,
// below the title an explanation or README puts above them.
func markdown(pkg *doc.Package, text string) string {
	printer := pkg.Printer()
	printer.HeadingLevel = 2
	printer.HeadingID = func(*comment.Heading) string { return "" }
	return blocks(codeSpans(string(printer.Markdown(pkg.Parser().Parse(text)))))
}

// blocks writes the printer's code blocks and lists the way the repo's own
// Markdown writes them. A run of tab-indented code lines, with any blank
// lines inside it, becomes a fenced block, which the prose checks mask as
// code. A list item loses the indent the printer puts before its marker.
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

// codeEnd is the index just past the code block starting at lines[start]:
// past its last tab-indented line, so the blank lines after it stay out.
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
// Markdown code spans. The printer escapes them, since a doc comment has no
// code spans, and escapes punctuation inside them, which a code span would
// show. Indented code blocks keep their text.
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

// parseSources parses each package source in dir in mode, such as the
// package clause and its doc comment alone when the generator needs only
// the doc comment.
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
