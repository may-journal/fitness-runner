package gocoverage

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, name, body string) {
	t.Helper()
	file := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n")
	write(t, root, "main.go", `package main
import "example.test/app/service"
func main() { service.Value(1) }
`)
	write(t, root, "service/service.go", `package service
func Value(n int) int {
 if n < 0 { return -1 }
 return n+1
}
`)
	write(t, root, "main_test.go", `package main
import "testing"
func TestEntry(t *testing.T) { main() }
`)
	return root
}
func execute(t *testing.T, root string, options Options) Report {
	t.Helper()
	report, err := Run(context.Background(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
func TestEntryCreditsChildrenAndReportsUntested(t *testing.T) {
	root := fixture(t)
	write(t, root, "unused/unused.go", "package unused\nfunc Value() int { return 42 }\n")
	report := execute(t, root, Options{})
	if len(report.Failures) != 0 {
		t.Fatal(report.Failures)
	}
	assertCount(t, report, "package", "example.test/app/service", 2, 3)
	assertCount(t, report, "package", "example.test/app/unused", 0, 1)
	if len(report.Gaps) != 2 {
		t.Fatal(report.Gaps)
	}
}
func assertCount(t *testing.T, r Report, kind, name string, covered, total int64) {
	t.Helper()
	for _, row := range r.Rows {
		if row.Kind == kind && row.Name == name {
			if row.Count != (Count{Covered: covered, Total: total}) {
				t.Fatalf("%+v", row)
			}
			return
		}
	}
	t.Fatalf("missing %s %s", kind, name)
}
func TestEntryFailureCannotHideInFolderTotal(t *testing.T) {
	root := fixture(t)
	write(t, root, "extra.go", "package main\nfunc uncovered() { println(42) }\n")
	report := execute(t, root, Options{})
	if !strings.Contains(strings.Join(report.Failures, "\n"), "extra.go") {
		t.Fatal(report.Failures)
	}
}
func TestOverlapPreservesDifferentAssertions(t *testing.T) {
	root := fixture(t)
	write(t, root, "service/service_test.go", `package service
import "testing"
func TestOne(t *testing.T) { if Value(1)!=2 { t.Fatal("one") } }
func TestTwo(t *testing.T) { if Value(2)!=3 { t.Fatal("two") } }
`)
	report := execute(t, root, Options{Audit: true})
	if len(report.Failures) != 0 {
		t.Fatal(report.Failures)
	}
	findings := strings.Join(report.Findings, "\n")
	if !strings.Contains(findings, "identical coverage") || !strings.Contains(findings, "assertions may differ") {
		t.Fatal(findings)
	}
}
func TestCompiledChildCoverage(t *testing.T) {
	root := fixture(t)
	write(t, root, "main_test.go", `package main
import ("os";"os/exec";"path/filepath";"testing")
func TestCLI(t *testing.T) {
 bin:=filepath.Join(t.TempDir(),"app")
 if out,err:=exec.Command("go","build","-cover","-o",bin,".").CombinedOutput(); err!=nil { t.Fatalf("%s: %v",out,err) }
 cmd:=exec.Command(bin)
 cmd.Env=append(os.Environ(),"GOCOVERDIR="+os.Getenv("FITNESS_GO_COVER_DIR"))
 if out,err:=cmd.CombinedOutput(); err!=nil { t.Fatalf("%s: %v",out,err) }
}
`)
	report := execute(t, root, Options{})
	if len(report.Failures) != 0 {
		t.Fatal(report.Failures)
	}
	assertCount(t, report, "package", "example.test/app", 1, 1)
}
func TestLibraryEntriesAndMultipleModules(t *testing.T) {
	root := fixture(t)
	write(t, root, "lib/go.mod", "module example.test/lib\n\ngo 1.24\n")
	write(t, root, "lib/lib.go", "package lib\nfunc Value() int { return 1 }\n")
	write(t, root, "lib/lib_test.go", "package lib\nimport \"testing\"\nfunc TestValue(t *testing.T) { if Value()!=1 { t.Fatal(\"value\") } }\n")
	report := execute(t, root, Options{Entries: []string{"example.test/lib"}})
	if len(report.Failures) != 0 {
		t.Fatal(report.Failures)
	}
	_, err := Run(context.Background(), root, Options{})
	if err == nil || !strings.Contains(err.Error(), "no entry packages") {
		t.Fatal(err)
	}
}
func TestTestFailureIsAnError(t *testing.T) {
	root := fixture(t)
	write(t, root, "main_test.go", "package main\nimport \"testing\"\nfunc TestBroken(t *testing.T) { t.Fatal(\"expected failure\") }\n")
	_, err := Run(context.Background(), root, Options{})
	if err == nil || !strings.Contains(err.Error(), "expected failure") {
		t.Fatal(err)
	}
}
func TestConcurrentRunsDoNotWriteToRepo(t *testing.T) {
	root := fixture(t)
	before := files(t, root)
	done := make(chan error, 2)
	for range 2 {
		go func() { _, err := Run(context.Background(), root, Options{}); done <- err }()
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(before, "\n") != strings.Join(files(t, root), "\n") {
		t.Fatal("coverage wrote to repository")
	}
}
func files(t *testing.T, root string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error { names = append(names, path); return err })
	if err != nil {
		t.Fatal(err)
	}
	return names
}
func TestNoModulesAndMissingToolchain(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := Run(context.Background(), t.TempDir(), Options{}); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), fixture(t), Options{})
	if err == nil || !strings.Contains(err.Error(), "Go not installed") {
		t.Fatal(err)
	}
}
func TestCanceledRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Run(ctx, fixture(t), Options{})
	if err == nil {
		t.Fatal("canceled run passed")
	}
}
func TestInvalidGoAndUnknownEntry(t *testing.T) {
	root := fixture(t)
	_, err := Run(context.Background(), root, Options{Entries: []string{"missing"}})
	if err == nil || !strings.Contains(err.Error(), "unknown entry") {
		t.Fatal(err)
	}
	write(t, root, "main.go", "not valid Go")
	_, err = Run(context.Background(), root, Options{})
	if err == nil {
		t.Fatal("invalid source passed")
	}
}
func TestMissingProfileAndChildData(t *testing.T) {
	bin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	r := runner{ctx: context.Background(), bin: bin, dir: t.TempDir()}
	if _, err := r.profiles(filepath.Join(r.dir, "missing"), r.dir); err == nil {
		t.Fatal("missing profile passed")
	}
	write(t, r.dir, "profile", "mode: set\n")
	if _, err := r.profiles(filepath.Join(r.dir, "profile"), filepath.Join(r.dir, "missing")); err == nil {
		t.Fatal("missing child directory passed")
	}
	write(t, r.dir, "children/bad", "broken")
	if _, err := r.profiles(filepath.Join(r.dir, "profile"), filepath.Join(r.dir, "children")); err == nil {
		t.Fatal("invalid child data passed")
	}
}

func TestChangedFileScope(t *testing.T) {
	root := fixture(t)
	t.Setenv("FITNESS_CHANGED_FILES", "readme.md")
	report := execute(t, root, Options{})
	if report.FileCount() != 0 {
		t.Fatal(report)
	}
}
func TestTempFailureAndBadSavedProfile(t *testing.T) {
	root := fixture(t)
	t.Setenv("TMPDIR", filepath.Join(root, "missing"))
	if _, err := Run(context.Background(), root, Options{}); err == nil {
		t.Fatal("missing temp directory accepted")
	}
	write(t, root, "bad-profile", "mode: broken\n")
	r := runner{dir: root}
	if _, err := r.profiles(filepath.Join(root, "bad-profile"), root); err == nil {
		t.Fatal("bad profile accepted")
	}
	if err := mergeProfileFile(Profile{}, filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing profile accepted")
	}
}
func TestAuditPropagatesTestErrors(t *testing.T) {
	root := fixture(t)
	write(t, root, "main_test.go", "package main\nimport \"testing\"\nfunc TestBroken(t *testing.T) { t.Fatal(\"audit failure\") }\n")
	bin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	r := runner{ctx: context.Background(), bin: bin, dir: root, temp: t.TempDir()}
	_, err = r.audit([]Package{{ImportPath: "example.test/app", TestGoFiles: []string{"main_test.go"}}})
	if err == nil || !strings.Contains(err.Error(), "audit failure") {
		t.Fatal(err)
	}
}

func TestAuditDetectsChangingCoverage(t *testing.T) {
	root := fixture(t)
	write(t, root, "main.go", `package main
func main() { exercise(false) }
func exercise(flag bool) { if flag { println(1); return }; println(2) }
`)
	write(t, root, "main_test.go", `package main
import ("os";"testing")
func TestFlip(t *testing.T) {
 data,_:=os.ReadFile("marker")
 flag:=string(data)!="yes"
 exercise(flag)
 next:="yes"; if !flag { next="no" }
 if err:=os.WriteFile("marker",[]byte(next),0600); err!=nil { t.Fatal(err) }
}
`)
	report := execute(t, root, Options{Audit: true})
	if !strings.Contains(strings.Join(report.Findings, "\n"), "unstable coverage") {
		t.Fatal(report.Findings)
	}
}

func TestRealAuditRetainsSourceClaims(t *testing.T) {
	root := fixture(t)
	report := execute(t, root, Options{AuditMap: filepath.Join(t.TempDir(), "claims.json")})
	if len(report.Maps) != 1 {
		t.Fatal(report.Maps)
	}
	claims := report.Maps[0]
	if claims.Module != "." || len(claims.Lines) == 0 {
		t.Fatal(claims)
	}
	requireChildClaims(t, claims.Blocks)
}
func requireChildClaims(t *testing.T, blocks []BlockClaim) {
	t.Helper()
	found := false
	for _, block := range blocks {
		if block.File == "example.test/app/service/service.go" && len(block.Tests) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("entry test did not claim child source")
	}
}
