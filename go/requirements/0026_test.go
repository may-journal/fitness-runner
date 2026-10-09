package requirements

import (
	"strings"
	"testing"
)

// goComplexity runs `fitness-install -- go-complexity` on happyRepo with
// files written over it.
func goComplexity(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "go-complexity")
}

// goComplexityIfs returns n `if` branches, one point each.
func goComplexityIfs(n int) string {
	return strings.Repeat("\tif ok {\n\t}\n", n)
}

// goComplexityFile is a Go file holding decl and its body of n branches.
func goComplexityFile(decl string, n int) string {
	return "package p\n\nvar ok bool\n\n" + decl + " {\n" + goComplexityIfs(n) + "}\n"
}

func Test0026_1(t *testing.T) {
	out, code := goComplexity(t, map[string]string{"a.go": goComplexityFile("func f()", 6)})
	sees(t, out, code, 1, "a.go:5: func f has a complexity of 7; maximum allowed is 5")
}

func Test0026_2(t *testing.T) {
	body := "package p\n\nfunc f(n int, ch chan int, a, b bool) bool {\n" +
		"\tswitch n {\n\tcase 1:\n\tcase 2:\n\tdefault:\n\t}\n" +
		"\tselect {\n\tcase <-ch:\n\tdefault:\n\t}\n" +
		"\tfor n > 0 {\n\t\tn--\n\t}\n" +
		"\tfor range ch {\n\t}\n" +
		"\treturn a && b || a\n}\n"
	out, code := goComplexity(t, map[string]string{"a.go": body})
	sees(t, out, code, 1, "a.go:3: func f has a complexity of 8; maximum allowed is 5")
}

func Test0026_3(t *testing.T) {
	body := "package p\n\nvar ok bool\n\nfunc outer() {\n\tg := func() {\n" + goComplexityIfs(5) + "\t}\n\tg()\n}\n"
	out, code := goComplexity(t, map[string]string{"a.go": body})
	sees(t, out, code, 1, "a.go:6: function literal has a complexity of 6; maximum allowed is 5")
	if strings.Contains(out, "func outer") {
		t.Errorf("outer must not carry the literal's branches:\n%s", out)
	}
}

func Test0026_4(t *testing.T) {
	file := "package p\n\ntype T struct{}\n\n" + strings.TrimPrefix(goComplexityFile("func (T) m()", 5), "package p\n\n")
	out, code := goComplexity(t, map[string]string{"a.go": file})
	sees(t, out, code, 1, "a.go:7: method m has a complexity of 6; maximum allowed is 5")
}

func Test0026_5(t *testing.T) {
	out, code := goComplexity(t, map[string]string{"a_test.go": goComplexityFile("func TestF()", 5)})
	sees(t, out, code, 1, "a_test.go:5: func TestF has a complexity of 6; maximum allowed is 5")
}

func Test0026_6(t *testing.T) {
	out, code := goComplexity(t, map[string]string{
		".fitnessrc.json": `{"goComplexity": {"max": 2}}` + "\n",
		"a.go":            goComplexityFile("func f()", 2),
	})
	sees(t, out, code, 1, "a.go:5: func f has a complexity of 3; maximum allowed is 2")
}

func Test0026_7(t *testing.T) {
	out, code := goComplexity(t, map[string]string{"bad.go": "package p\nfunc {"})
	sees(t, out, code, 1, "✖ bad.go:")
	if !strings.Contains(goComplexityJoined(out), goComplexityJoined("bad.go:2:6: expected 'IDENT', found '{'")) {
		t.Errorf("output is missing the parser's error:\n%s", out)
	}
}

func Test0026_8(t *testing.T) {
	out, code := goComplexity(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-complexity")
}

// goComplexityJoined drops the table borders and every space, so a long
// message the table wrapped mid-word reads whole.
func goComplexityJoined(out string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(out, "│", " ")), "")
}
