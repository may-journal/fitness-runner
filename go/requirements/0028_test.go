package requirements

import (
	"os"
	"path/filepath"
	"testing"
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

// gtcLibEntry declares the library package as an entry.
const gtcLibEntry = `{"goTestCoverage": {"entries": ["example.test/lib"]}}` + "\n"

func Test0028_1(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcCommand(""))
	sees(t, out, code, 1, "entry example.test/cli: 0/1 statements; required 100%")
}

func Test0028_2(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcCommand(gtcEntryTest))
	sees(t, out, code, 0, "package example.test/cli: 1/1 statements (100.0%); required 100%", "All 1 checks passed")
}

func Test0028_3(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcLibrary("{}\n"))
	sees(t, out, code, 1, "no entry packages; declare library entries with --entry=import/path")
}

func Test0028_4(t *testing.T) {
	t.Parallel()
	out, code := goTestCoverage(t, gtcLibrary(gtcLibEntry))
	sees(t, out, code, 1, "entry example.test/lib: 1/2 statements; required 100%")
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
