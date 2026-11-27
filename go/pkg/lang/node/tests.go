package node

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/comments"
	"github.com/may-journal/fitness-runner/go/pkg/lang"
	"github.com/may-journal/fitness-runner/go/pkg/lang/node/javascript"
)

// Tests finds vitest and jest tests whose title opens with an acceptance ID, as
// in test("0001.1 saves offline", …).
var Tests = lang.Tests{Grouped: true, Scan: scanTests, Name: func(id string) string { return `test("` + id + ` …")` }, Reads: ScriptTest}

// scanTests reads every script test file's tests; a repo with any script
// is written in TypeScript or JavaScript.
func scanTests(root string) ([]lang.Test, []string, bool, error) {
	scripts := Scripts(root)
	files := scriptTestFiles(scripts)
	tests, err := scanScripts(root, files)
	return tests, files, len(scripts) > 0, err
}

var (
	// scriptTestID matches a vitest title that opens with an acceptance ID:
	// "0001.1 adds two numbers" is 0001.1.
	scriptTestID = regexp.MustCompile(`^(\d{4})\.([1-9]\d*)(?:\s|$)`)
	// testFuncs are the vitest and jest functions that declare one test.
	testFuncs = map[string]bool{"it": true, "test": true}
	// groupFuncs are the vitest and jest functions that declare one group.
	groupFuncs = map[string]bool{"describe": true}
	// testModifiers may follow it, test or describe before its call, as in
	// it.skip(…).
	testModifiers = map[string]bool{"skip": true, "only": true, "todo": true, "concurrent": true, "sequential": true, "fails": true, "shuffle": true}
	// tableModifiers take their own arguments first, as in it.each(rows)(…).
	tableModifiers = map[string]bool{"each": true, "for": true, "skipIf": true, "runIf": true}
	// rowModifiers are the table modifiers that run one declaration per row.
	rowModifiers = map[string]bool{"each": true, "for": true}
)

// declared is one it, test or describe call.
type declared struct {
	// title is the call's first argument.
	title javascript.Token
	// open is the index of the parenthesis that opens the arguments.
	open int
	// table says each or for declares the call once for many rows.
	table bool
}

// openGroup is a describe call the reader is inside.
type openGroup struct {
	lang.Group
	// end is the index of the parenthesis that closes the call.
	end int
}

// scanScripts reads the code view of every vitest or jest test file and
// returns its tests.
func scanScripts(root string, files []string) ([]lang.Test, error) {
	var tests []lang.Test
	for _, f := range files {
		data, err := comments.CodeFile(root, f)
		if err != nil {
			return nil, err
		}
		tests = append(tests, scriptFileTests(f, string(data))...)
	}
	return tests, nil
}

// scriptTestFiles lists the scripts named as test files, such as
// adder.test.ts.
func scriptTestFiles(scripts []string) []string {
	var out []string
	for _, f := range scripts {
		if ScriptTest(f) {
			out = append(out, f)
		}
	}
	return out
}

// scriptFileTests returns each it or test call in one file, with the
// describe calls around it. Comments and strings are tokens of their own, so
// a call inside them is no test, and describe titles never name a test.
func scriptFileTests(file, src string) []lang.Test {
	tokens := javascript.Lex(src)
	loc := func(i int) string { return fmt.Sprintf("%s:%d", file, strings.Count(src[:tokens[i].Pos], "\n")+1) }
	var tests []lang.Test
	var open []openGroup
	for i := range tokens {
		open = slices.DeleteFunc(open, func(g openGroup) bool { return g.end < i })
		if d, ok := declaration(tokens, i, groupFuncs); ok {
			open = append(open, openGroup{Group: lang.Group{Title: groupTitle(tokens, d), Loc: loc(i), Table: d.table}, end: closing(tokens, d.open)})
		}
		if d, ok := testCall(tokens, i); ok {
			name := tokens[i].Text + "(" + shownTitle(d.title) + ")"
			tests = append(tests, lang.Test{Name: name, ID: scriptIDOf(d.title), Loc: loc(i), File: file, Groups: groupsOf(open), Table: d.table})
		}
	}
	return tests
}

// groupsOf lists the open groups, outermost first, or nil for none.
func groupsOf(open []openGroup) []lang.Group {
	var groups []lang.Group
	for _, g := range open {
		groups = append(groups, g.Group)
	}
	return groups
}

// groupTitle is a describe call's title, or "" when its first argument is
// no whole string.
func groupTitle(tokens []javascript.Token, d declared) string {
	title, whole := titleAt(tokens, d.open+1)
	if !whole {
		return ""
	}
	return title.Text
}

// testCall reports whether tokens[i] starts a call declaring one test,
// whose title must be the whole first argument.
func testCall(tokens []javascript.Token, i int) (declared, bool) {
	d, ok := declaration(tokens, i, testFuncs)
	if !ok {
		return d, false
	}
	_, whole := titleAt(tokens, d.open+1)
	return d, whole
}

// declaration reports whether tokens[i] starts a call of one of funcs with
// arguments, and returns it.
func declaration(tokens []javascript.Token, i int, funcs map[string]bool) (declared, bool) {
	if !names(tokens, i, funcs) {
		return declared{}, false
	}
	open, table, ok := callOpen(tokens, i+1)
	if !ok || !takesArguments(tokens, open) {
		return declared{}, false
	}
	return declared{title: tokens[open+1], open: open, table: table}, true
}

// names reports whether tokens[i] is one of funcs as a name of its own, not
// a property or a declaration.
func names(tokens []javascript.Token, i int, funcs map[string]bool) bool {
	t := tokens[i]
	return t.Kind == javascript.Ident && funcs[t.Text] && (i == 0 || !isMemberOrDecl(tokens[i-1]))
}

// takesArguments reports a call at open with at least one argument that is
// no method definition.
func takesArguments(tokens []javascript.Token, open int) bool {
	return open+1 < len(tokens) && !isPunct(tokens[open+1], ")") && !definesMethod(tokens, open)
}

// titleAt returns the first argument when it is a whole title string.
func titleAt(tokens []javascript.Token, i int) (javascript.Token, bool) {
	return tokens[i], tokens[i].Kind == javascript.String && isArgEnd(tokens, i+1)
}

// isMemberOrDecl reports a token that makes the next name a property, as in
// re.test(…), or a declaration, as in function test(…).
func isMemberOrDecl(t javascript.Token) bool {
	return isPunct(t, ".") || t.Kind == javascript.Ident && (t.Text == "function" || t.Text == "class")
}

// callOpen skips the modifiers after it or test and returns the index of
// the parenthesis that opens the call taking the title. table says each or
// for was among them.
func callOpen(tokens []javascript.Token, i int) (open int, table, ok bool) {
	for isModifier(tokens, i) {
		table = table || rowModifiers[tokens[i+1].Text]
		i = afterModifier(tokens, i)
	}
	return i, table, i < len(tokens) && isPunct(tokens[i], "(")
}

// isModifier reports whether tokens[i:] is a known modifier such as .skip.
func isModifier(tokens []javascript.Token, i int) bool {
	if i+1 >= len(tokens) || !isPunct(tokens[i], ".") {
		return false
	}
	name := tokens[i+1].Text
	return tokens[i+1].Kind == javascript.Ident && (testModifiers[name] || tableModifiers[name])
}

// afterModifier steps past the modifier at i and any table it takes.
func afterModifier(tokens []javascript.Token, i int) int {
	if tableModifiers[tokens[i+1].Text] {
		return skipTable(tokens, i+2)
	}
	return i + 2
}

// skipTable steps past a table modifier's own arguments: a call such as
// (rows) or a tagged template.
func skipTable(tokens []javascript.Token, i int) int {
	if i < len(tokens) && tokens[i].Kind == javascript.String {
		return i + 1
	}
	if i < len(tokens) && isPunct(tokens[i], "(") {
		return closing(tokens, i) + 1
	}
	return i
}

// closing returns the index of the parenthesis that closes the one at open,
// or the last index when it never closes.
func closing(tokens []javascript.Token, open int) int {
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch {
		case isPunct(tokens[i], "("):
			depth++
		case isPunct(tokens[i], ")"):
			depth--
		}
		if depth == 0 {
			return i
		}
	}
	return len(tokens) - 1
}

// definesMethod reports a parameter list followed by a body or return type,
// which declares no test.
func definesMethod(tokens []javascript.Token, open int) bool {
	end := closing(tokens, open)
	return end+1 < len(tokens) && (isPunct(tokens[end+1], "{") || isPunct(tokens[end+1], ":"))
}

// isArgEnd reports whether the argument ends at tokens[i]: a comma or the
// closing parenthesis.
func isArgEnd(tokens []javascript.Token, i int) bool {
	return i < len(tokens) && (isPunct(tokens[i], ",") || isPunct(tokens[i], ")"))
}

// isPunct reports whether t is the punctuation s.
func isPunct(t javascript.Token, s string) bool {
	return t.Kind == javascript.Punct && t.Text == s
}

// shownTitle is a title as an error shows it, quoted.
func shownTitle(title javascript.Token) string {
	return fmt.Sprintf("%q", title.Text)
}

// scriptIDOf returns the acceptance ID a vitest title opens with, or "".
func scriptIDOf(title javascript.Token) string {
	m := scriptTestID.FindStringSubmatch(title.Text)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}
