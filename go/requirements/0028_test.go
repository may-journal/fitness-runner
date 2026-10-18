package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// goTestCoverage runs `fitness-install -- go-test-coverage <args>` on
// happyRepo with files written over it.
func goTestCoverage(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"go-test-coverage"}, args...)...)
}

// gtcCommand is a Go module with one command package of one statement.
func gtcCommand(test string) map[string]string {
	return map[string]string{
		"go.mod":       "module example.test/cli\n\ngo 1.24\n",
		"main.go":      "package main\n\nfunc main() { println(42) }\n",
		"main_test.go": test,
	}
}

// gtcEntryTest is a command test that runs every statement.
const gtcEntryTest = "package main\n\nimport \"testing\"\n\nfunc TestEntry(t *testing.T) { main() }\n"

// gtcLibrary is a Go module with one library package of two statements, a
// config written over happyRepo's, and a test that runs only One.
func gtcLibrary(config string) map[string]string {
	return map[string]string{
		".fitnessrc.json": config,
		"go.mod":          "module example.test/lib\n\ngo 1.24\n",
		"lib.go":          "package lib\n\nfunc One() int { return 1 }\n\nfunc Two() int { return 2 }\n",
		"lib_test.go":     "package lib\n\nimport \"testing\"\n\nfunc TestOne(t *testing.T) { One() }\n",
	}
}

func Test0028_3(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcLibrary("{}\n"))
	sees(t, out, code, 1, "no entry packages; declare library entries with --entry=import/path")
}

func Test0028_5(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcCommand(gtcEntryTest), "--entry=example.test/nope")
	sees(t, out, code, 1, `unknown entry package "example.test/nope"`)
}

func Test0028_6(t *testing.T) {
	t.Parallel()
	wrapped := "package main\n\nimport \"testing\"\n\nfunc TestAll(t *testing.T) { t.Run(\"entry\", runsMain) }\n\nfunc runsMain(t *testing.T) { main() }\n"
	out, code := goTestCoverage(t, gtcCommand(wrapped))
	sees(t, out, code, 1, "main_test.go:5: TestAll passes named helper runsMain to t.Run")
}

func Test0028_7(t *testing.T) {
	t.Parallel()
	out, _ := goTestCoverage(t, nil)
	passedWithNoFiles(t, out, "go-test-coverage")
}

func Test0028_8(t *testing.T) {
	t.Parallel()
	claims := filepath.Join(t.TempDir(), "claims.json")
	out, code := goTestCoverage(t, gtcCommand(gtcEntryTest), "--audit-map="+claims)
	sees(t, out, code, 0, "All 1 checks passed")
	if _, err := os.Stat(claims); err != nil {
		t.Errorf("source map was not written: %v", err)
	}
}

func Test0028_9(t *testing.T) {
	t.Parallel()
	lib := gtcLibrary(`{"goTestCoverage": {"entries": ["example.test/cli/lib"]}}` + "\n")
	files := gtcCommand("")
	files[".fitnessrc.json"] = lib[".fitnessrc.json"]
	for _, name := range []string{"lib.go", "lib_test.go"} {
		files["lib/"+name] = lib[name]
	}
	out, code := goTestCoverage(t, files)
	sees(t, out, code, 1, "entry example.test/cli: 0/1 statements; required 100%", "entry example.test/cli/lib: 1/2 statements; required 100%")
}

// gtcFlip is a command source file whose flip takes one branch for even
// counts and the other for odd ones.
const gtcFlip = "package main\n\nfunc flip(n int) int {\n\tif n%2 == 0 {\n\t\treturn 1\n\t}\n\treturn 2\n}\n"

// gtcFlipTest covers the command fully in TestEntry, while TestFlip takes the
// other branch of flip each time it runs, counted in a file it appends to.
const gtcFlipTest = `package main

import (
	"os"
	"testing"
)

func TestEntry(t *testing.T) { main(); flip(0); flip(1) }

func TestFlip(t *testing.T) {
	runs, _ := os.ReadFile("runs")
	if err := os.WriteFile("runs", append(runs, 'x'), 0o600); err != nil {
		t.Fatal(err)
	}
	flip(len(runs))
}
`

func Test0028_10(t *testing.T) {
	t.Parallel()
	files := gtcCommand(gtcFlipTest)
	files["flip.go"] = gtcFlip
	out, code := goTestCoverage(t, files)
	sees(t, out, code, 0, "example.test/cli/TestFlip (", "unstable coverage; excluded from overlap advice")
}

// gtcSlowTests adds sixteen one-second tests: eight call t.Parallel() in one
// package, and eight run serially, two in each of four more packages. Run one
// test at a time, either half alone takes sixteen seconds.
func gtcSlowTests() map[string]string {
	files := gtcCommand(gtcEntryTest)
	gtcSlowPackage(files, "parallel", 8, "\tt.Parallel()\n")
	for i := range 4 {
		gtcSlowPackage(files, fmt.Sprintf("serial%d", i), 2, "")
	}
	return files
}

// gtcSlowPackage writes package pkg with count one-second tests, each
// starting with first.
func gtcSlowPackage(files map[string]string, pkg string, count int, first string) {
	files[pkg+"/wait.go"] = "package " + pkg + "\n\nimport \"time\"\n\nfunc Wait() { time.Sleep(time.Second) }\n"
	test := "package " + pkg + "\n\nimport \"testing\"\n"
	for i := range count {
		test += fmt.Sprintf("\nfunc TestWait%d(t *testing.T) {\n%s\tWait()\n}\n", i, first)
	}
	files[pkg+"/wait_test.go"] = test
}

func Test0028_11(t *testing.T) {
	t.Parallel()
	start := time.Now()
	out, code := goTestCoverage(t, gtcSlowTests())
	elapsed := time.Since(start)
	sees(t, out, code, 0, "example.test/cli/parallel/TestWait7 (", "example.test/cli/serial3/TestWait1 (", "All 1 checks passed")
	if elapsed > 15*time.Second {
		t.Errorf("go-test-coverage took %s with sixteen one-second tests, want within 15s", elapsed.Round(time.Millisecond))
	}
}
