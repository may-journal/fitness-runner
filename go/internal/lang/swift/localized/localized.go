// Package localized judges the localized-text check: in an app that
// ships a String Catalog, every word a SwiftUI view shows comes from the
// catalog, and the requirement tests read the same entries. A view that
// passes a string literal to Text, Label, Button, Section, or a title,
// accessibility, alert or dialog modifier fails. So does a test literal
// that types an entry's words. Each finding names the file, line and
// literal, and the reference to use instead.
package localized

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Name is the check's name.
const Name = "localized-text"

// shown are the SwiftUI views and modifiers whose first argument a
// literal turns into a LocalizedStringKey, so its words are on screen.
var shown = map[string]bool{
	"Text": true, "Label": true, "Button": true, "Section": true,
	"navigationTitle": true, "accessibilityLabel": true, "accessibilityHint": true,
	"alert": true, "confirmationDialog": true,
}

// Run judges the repo at root. A repo without a String Catalog has not
// opted in, so it passes with no files judged.
func Run(root string, _ []string) (checkkit.Result, error) {
	catalogs := walkfs.FilesByExt(root, catalogExt)
	if len(catalogs) == 0 {
		return checkkit.Pass(0), nil
	}
	known, cfg, problems, err := prepare(root, catalogs)
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	files := walkfs.InScope(walkfs.FilesByExt(root, swift.Ext))
	findings, err := judgeFiles(root, files, known, cfg)
	if err != nil {
		return checkkit.Result{}, err
	}
	if findings = append(problems, findings...); len(findings) > 0 {
		return checkkit.Fail(len(files), findings...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// prepare reads the config and every catalog. An error is one the check
// fails with; problems are the config's allow patterns it cannot use.
func prepare(root string, catalogs []string) (entries, config, []string, error) {
	cfg, problems, err := loadConfig(root)
	if err != nil {
		return nil, cfg, nil, err
	}
	known, err := readCatalogs(root, catalogs)
	return known, cfg, problems, err
}

// judgeFiles reads and judges each Swift file in turn.
func judgeFiles(root string, files []string, known entries, cfg config) ([]string, error) {
	var findings []string
	for _, file := range files {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return nil, err
		}
		findings = append(findings, judge(file, string(src), known, cfg)...)
	}
	return findings, nil
}

// judge returns one file's findings: a test file's literals that type a
// catalog entry, or an app file's literals shown on screen.
func judge(file, src string, known entries, cfg config) []string {
	code, literals := swift.Literals(src)
	find := shownFinding
	if isTestFile(file) {
		find = typedFinding
	}
	var out []string
	for _, lit := range literals {
		if cfg.allowed(lit.Text) {
			continue
		}
		if finding := find(code, lit, known, cfg); finding != "" {
			out = append(out, fmt.Sprintf("%s:%d - %s", file, lit.Line, finding))
		}
	}
	return out
}

// typedFinding is a test literal's finding when it types an entry's
// words, or "".
func typedFinding(_ string, lit swift.Literal, known entries, cfg config) string {
	key, ok := known[lit.Text]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s types catalog entry %q; read it through %s", lit.Source, key, cfg.reference)
}

// shownFinding is an app literal's finding when a view shows it, or "".
func shownFinding(code string, lit swift.Literal, known entries, cfg config) string {
	call := shownBy(code, lit.Start)
	if call == "" || !hasWords(lit.Text) {
		return ""
	}
	fix := fmt.Sprintf("add it to the String Catalog and use %s", cfg.reference)
	if key, ok := known[lit.Text]; ok {
		fix = fmt.Sprintf("use %s for catalog entry %q", cfg.reference, key)
	}
	return fmt.Sprintf("%s passed to %s skips the String Catalog; %s", lit.Source, call, fix)
}

// shownBy returns the view or modifier a literal at start is the first
// argument of, unlabeled or labeled verbatim, or "" when it is none.
// code has its comments and literals blanked, so only code is read.
func shownBy(code string, start int) string {
	i, ok := argumentStart(code, start)
	if !ok || !opensCall(code, i) {
		return ""
	}
	name, at := nameBack(code, skipSpaceBack(code, i-1))
	before, _ := nameBack(code, skipSpaceBack(code, at))
	if !shown[name] || before == "func" {
		return ""
	}
	return name
}

// opensCall reports a parenthesis just before i, opening a call.
func opensCall(code string, i int) bool { return i > 0 && code[i-1] == '(' }

// argumentStart returns where the argument holding a literal at start
// begins, before a verbatim label; ok is false for another label.
func argumentStart(code string, start int) (int, bool) {
	i := skipSpaceBack(code, start)
	if i == 0 || code[i-1] != ':' {
		return i, true
	}
	label, at := nameBack(code, skipSpaceBack(code, i-1))
	return skipSpaceBack(code, at), label == "verbatim"
}

// skipSpaceBack moves i back over spaces and line breaks.
func skipSpaceBack(code string, i int) int {
	for i > 0 && strings.IndexByte(" \t\r\n", code[i-1]) >= 0 {
		i--
	}
	return i
}

// nameBack returns the name ending at i and the offset it starts at.
func nameBack(code string, i int) (string, int) {
	end := i
	for i > 0 && isNameByte(code[i-1]) {
		i--
	}
	return code[i:end], i
}

// isNameByte reports a byte of a Swift name: an ASCII letter, digit or
// underscore, or any byte of a non-ASCII character.
func isNameByte(c byte) bool {
	return c == '_' || c >= 0x80 || isASCIIAlnum(c)
}

// isASCIIAlnum reports an ASCII digit or letter, of either case.
func isASCIIAlnum(c byte) bool {
	return c >= '0' && c <= '9' || (c|0x20) >= 'a' && (c|0x20) <= 'z'
}

// isTestFile reports a file of a test target: one in a folder whose name
// ends in Tests, as Xcode and SwiftPM name test targets, or one named
// like a test file.
func isTestFile(file string) bool {
	if strings.HasSuffix(path.Base(file), "Tests.swift") {
		return true
	}
	for _, dir := range strings.Split(path.Dir(file), "/") {
		if strings.HasSuffix(dir, "Tests") {
			return true
		}
	}
	return false
}
