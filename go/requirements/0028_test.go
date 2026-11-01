package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// gtcLibrary is a Go module with a two-statement library package, a config
// written over happyRepo's, and a test running only One.
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
	repo := example(t, "happyRepo", gtcCommand(gtcEntryTest))
	out, code := fitness(t, repo, []string{pathWithout(t, "go")}, "go-test-coverage")
	showsInfo(t, out, code, "go-test-coverage", "Go not installed: see https://go.dev/dl")
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

// gtcFlipTest covers the command in TestEntry, while TestFlip takes the other
// branch of flip on each run, counted in a file.
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
// package, and eight run serially, two in each of four more packages. Each
// run appends its package, test, process, start and end to runs.log at the
// module root.
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
	test := "package " + pkg + "\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"testing\"\n\t\"time\"\n)\n" + gtcRecord
	for i := range count {
		test += fmt.Sprintf("\nfunc TestWait%d(t *testing.T) {\n%s\trecord(t, %q)\n}\n", i, first, pkg)
	}
	files[pkg+"/wait_test.go"] = test
}

// gtcRecord is the slow tests' helper: it waits a second and logs the run.
const gtcRecord = `
func record(t *testing.T, pkg string) {
	start := time.Now().UnixNano()
	Wait()
	f, err := os.OpenFile("../runs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s %d %d %d\n", pkg, t.Name(), os.Getpid(), start, time.Now().UnixNano())
}
`

// gtcRun is one logged test run.
type gtcRun struct {
	pkg, test  string
	start, end int64
}

// gtcAuditRuns reads runs.log and keeps the audit runs whose process ran one
// test, since the module-wide run logs several per process.
func gtcAuditRuns(t *testing.T, repo string) []gtcRun {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, "runs.log"))
	if err != nil {
		t.Fatalf("the slow tests logged no runs: %v", err)
	}
	perProcess := map[string][]gtcRun{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		pid, run := gtcParseRun(t, line)
		perProcess[pid] = append(perProcess[pid], run)
	}
	var runs []gtcRun
	for _, group := range perProcess {
		runs = append(runs, gtcSingle(group)...)
	}
	return runs
}

// gtcParseRun reads one runs.log line as its process ID and run.
func gtcParseRun(t *testing.T, line string) (string, gtcRun) {
	t.Helper()
	var run gtcRun
	var pid string
	if _, err := fmt.Sscan(line, &run.pkg, &run.test, &pid, &run.start, &run.end); err != nil {
		t.Fatalf("bad runs.log line %q: %v", line, err)
	}
	return pid, run
}

// gtcSingle keeps a process's run only when it ran exactly one test.
func gtcSingle(group []gtcRun) []gtcRun {
	if len(group) != 1 {
		return nil
	}
	return group
}

// gtcOverlaps counts overlapping pairs of runs by gtcPairKind.
func gtcOverlaps(runs []gtcRun) map[string]int {
	counts := map[string]int{}
	for i, a := range runs {
		for _, b := range runs[i+1:] {
			if a.start < b.end && b.start < a.end {
				counts[gtcPairKind(a, b)]++
			}
		}
	}
	return counts
}

// gtcPairKind names what an overlap of runs a and b shows: "forbidden"
// within one serial package or test, else the runs' sides.
func gtcPairKind(a, b gtcRun) string {
	if a.pkg == b.pkg && (a.test == b.test || a.pkg != "parallel") {
		return "forbidden"
	}
	return gtcSide(a) + "-" + gtcSide(b)
}

// gtcSide is "parallel" for the parallel package's runs, else "serial".
func gtcSide(run gtcRun) string {
	if run.pkg == "parallel" {
		return "parallel"
	}
	return "serial"
}

func Test0028_11(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", gtcSlowTests())
	out, code := fitness(t, repo, nil, "go-test-coverage")
	sees(t, out, code, 0, "All 1 checks passed")
	runs := gtcAuditRuns(t, repo)
	if len(runs) != 32 {
		t.Fatalf("audit ran the sixteen tests %d times, want 32", len(runs))
	}
	counts := gtcOverlaps(runs)
	if counts["parallel-parallel"] == 0 {
		t.Errorf("no two runs of the parallel tests overlapped; go test runs them side by side")
	}
	if counts["serial-serial"] == 0 {
		t.Errorf("no two serial packages' runs overlapped; go test runs packages side by side")
	}
	if counts["forbidden"] > 0 {
		t.Errorf("%d pairs of runs overlapped within a serial package or one test; go test runs them in turn", counts["forbidden"])
	}
}
