package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScoring(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"straight line", `x := 1; _ = x`, 1},
		{"if", `if true { }`, 2},
		{"if else-if", `if true { } else if false { }`, 3},
		{"for", `for i := 0; i < 1; i++ { }`, 2},
		{"range", `for range []int{} { }`, 2},
		{"switch cases count, default does not", `switch 1 { case 1: case 2: default: }`, 3},
		{"select comm clauses count, default does not", `ch := make(chan int); select { case <-ch: default: }`, 2},
		{"logical and", `_ = true && false`, 2},
		{"logical or chain", `_ = true || false || true`, 3},
		{"mixed", `if true && false { for i := 0; i < 1; i++ { } }`, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "a.go", "package p\n\nfunc f() {\n"+tc.body+"\n}\n")
			errs, err := scanFile(dir, "a.go", 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(errs) != 1 {
				t.Fatalf("want 1 finding over max 0, got %v", errs)
			}
			want := "func f has a complexity of "
			if !strings.Contains(errs[0], want) {
				t.Fatalf("error shape: %s", errs[0])
			}
			if !strings.Contains(errs[0], want+strconv.Itoa(tc.want)+";") {
				t.Fatalf("complexity: want %d in %q", tc.want, errs[0])
			}
		})
	}
}

func TestFunctionLiteralsScoreSeparately(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", `package p

func outer() {
	f := func() {
		if true {
		}
		if true {
		}
	}
	f()
}
`)
	errs, err := scanFile(dir, "a.go", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "function literal has a complexity of 3") {
		t.Fatalf("literal must score alone (outer stays 1): %v", errs)
	}
}

func TestMethodNaming(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", `package p

type T struct{}

func (T) m() {
	if true {
	}
	if true {
	}
}
`)
	errs, err := scanFile(dir, "a.go", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "method m has a complexity of 3") {
		t.Fatalf("method naming: %v", errs)
	}
}

func TestRunExemptsTestsAndHonorsConfig(t *testing.T) {
	dir := t.TempDir()
	complex := "package p\n\nfunc f() {\n\tif true {\n\t}\n\tif true {\n\t}\n}\n"
	write(t, dir, "a.go", complex)
	write(t, dir, "a_test.go", complex)
	write(t, dir, ".fitnessrc.json", `{"goComplexity": {"max": 2}}`)

	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || len(res.Errors) != 1 {
		t.Fatalf("only a.go judged (max 2): %+v", res)
	}
	if res.FilesChecked != 1 {
		t.Fatalf("test files exempt from filesChecked: %d", res.FilesChecked)
	}

	// default max 5 passes the same file
	if err := os.Remove(filepath.Join(dir, ".fitnessrc.json")); err != nil {
		t.Fatal(err)
	}
	res, err = run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("complexity 3 under default max 5 must pass: %+v", res)
	}
}

func TestUnparsableFileReportsError(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.go", "package p\nfunc {")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || len(res.Errors) == 0 || !strings.Contains(res.Errors[0], "bad.go") {
		t.Fatalf("parse failure must fail with the file named: %+v", res)
	}
}
