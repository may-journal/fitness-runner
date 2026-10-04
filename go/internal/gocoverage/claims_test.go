package gocoverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sampleBlock(line, column, end int) Block {
	return Block{File: "m/code.go", StartLine: line, StartColumn: column, EndLine: line, EndColumn: end, Statements: 1}
}
func stableGroup(name string, blocks ...Block) testGroup {
	group := testGroup{Name: name, Stable: true, Covered: map[Block]bool{}}
	for _, b := range blocks {
		group.Covered[b] = true
	}
	return group
}
func TestLineOwnershipAndPartialOverlap(t *testing.T) {
	shared := sampleBlock(10, 2, 8)
	aOnly := sampleBlock(11, 2, 9)
	bOnly := sampleBlock(12, 2, 9)
	claims := claimMap([]testGroup{stableGroup("B", shared, bOnly), stableGroup("A", shared, aOnly)})
	if len(claims.Overlaps) != 1 {
		t.Fatal(claims.Overlaps)
	}
	pair := claims.Overlaps[0]
	if ([2]string{pair.A, pair.B} != [2]string{"A", "B"}) || !reflect.DeepEqual(pair.SharedBlocks, []int{1}) {
		t.Fatal(pair)
	}
	if !reflect.DeepEqual(claims.Lines[0].Tests, []string{"A", "B"}) {
		t.Fatal(claims.Lines)
	}
	requireClaim(t, claims.Tests[0], []int{1, 2})
	requireClaim(t, claims.Tests[1], []int{1, 3})
}
func requireClaim(t *testing.T, claim TestClaim, blocks []int) {
	t.Helper()
	if !reflect.DeepEqual(claim.Blocks, blocks) {
		t.Fatal(claim)
	}
}
func TestSameLineDifferentBlocks(t *testing.T) {
	a, b := sampleBlock(4, 1, 9), sampleBlock(4, 12, 22)
	claims := claimMap([]testGroup{stableGroup("A", a), stableGroup("B", b)})
	if len(claims.Overlaps) != 1 {
		t.Fatal(claims)
	}
	pair := claims.Overlaps[0]
	if len(pair.SharedBlocks) != 0 || !reflect.DeepEqual(pair.SharedLines, []int{1}) {
		t.Fatal(pair)
	}
	if !reflect.DeepEqual(claims.Lines[0].Blocks, []int{1, 2}) {
		t.Fatal(claims.Lines)
	}
}
func TestSetupAndUnstableClaimsAreSeparate(t *testing.T) {
	body, setup := sampleBlock(1, 1, 5), sampleBlock(2, 1, 5)
	a := stableGroup("A", body)
	a.Setup = map[Block]bool{setup: true}
	b := stableGroup("B")
	b.Setup = map[Block]bool{setup: true}
	unstable := stableGroup("C", body)
	unstable.Stable = false
	claims := claimMap([]testGroup{a, b, unstable})
	if len(claims.Overlaps) != 0 || len(claims.Tests[2].Blocks) != 0 {
		t.Fatal(claims)
	}
	if !reflect.DeepEqual(claims.Lines[1].SetupFor, []string{"A", "B"}) {
		t.Fatal(claims.Lines)
	}
	if len(claims.Lines[1].Tests) != 0 {
		t.Fatal(claims.Lines)
	}
}
func TestLineRangesAndFileBoundaries(t *testing.T) {
	a := sampleBlock(2, 2, 1)
	a.EndLine = 5
	b := sampleBlock(3, 2, 8)
	b.File = "m/other.go"
	claims := claimMap([]testGroup{stableGroup("A", a), stableGroup("B", b)})
	if len(claims.Lines) != 4 || len(claims.Overlaps) != 0 {
		t.Fatal(claims)
	}
	for _, line := range claims.Lines {
		if line.Line == 5 {
			t.Fatal("exclusive end column counted")
		}
	}
}
func TestClaimMapDeterministic(t *testing.T) {
	a, b := sampleBlock(1, 1, 8), sampleBlock(2, 1, 8)
	first := []testGroup{stableGroup("B", b, a), stableGroup("A", a)}
	second := []testGroup{stableGroup("A", a), stableGroup("B", a, b)}
	if !reflect.DeepEqual(claimMap(first), claimMap(second)) {
		t.Fatal("order changed source map")
	}
}
func TestFreshExportOutsideRepository(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(t.TempDir(), "claims.json")
	report := Report{Maps: []ModuleMap{claimMap([]testGroup{stableGroup("A", sampleBlock(1, 1, 3))})}}
	if err := report.WriteAudit(root, destination); err != nil {
		t.Fatal(err)
	}
	verifyExport(t, destination)
	if err := report.WriteAudit(root, destination); err == nil {
		t.Fatal("overwrote prior report")
	}
	if err := report.WriteAudit(root, filepath.Join(root, "claims.json")); err == nil {
		t.Fatal("wrote to repository")
	}
}
func verifyExport(t *testing.T, file string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var report AuditMap
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != 1 || len(report.Modules) != 1 {
		t.Fatal(report)
	}
}
func TestExportRejectsSymlinkIntoRepo(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	err := (Report{}).WriteAudit(root, filepath.Join(link, "claims.json"))
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatal(err)
	}
}
func TestExportRequiresExistingParent(t *testing.T) {
	root := t.TempDir()
	if err := (Report{}).WriteAudit(root, filepath.Join(t.TempDir(), "missing", "claims.json")); err == nil {
		t.Fatal("missing parent accepted")
	}
}
func TestSourcePathsAreRelativeToModule(t *testing.T) {
	root := t.TempDir()
	write(t, root, "api/api.go", "package api\n")
	got, err := sourceFiles(root, []Package{{ImportPath: "m", Dir: root, GoFiles: []string{"entry.go"}}, {ImportPath: "m/api", Dir: filepath.Join(root, "api"), GoFiles: []string{"api.go"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"m/entry.go": "entry.go", "m/api/api.go": "api/api.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestSourcePathsResolveDirectoryAliases(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sourceFiles(link, []Package{{ImportPath: "m", Dir: real, GoFiles: []string{"main.go"}}})
	if err != nil || got["m/main.go"] != "main.go" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestSourcePathsAcceptRelativeModule(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	got, err := sourceFiles(".", []Package{{ImportPath: "m", Dir: root, GoFiles: []string{"main.go"}}})
	if err != nil || got["m/main.go"] != "main.go" {
		t.Fatalf("%v %v", got, err)
	}
}
