package gocoverage

import (
	"math"
	"strings"
	"testing"
)

func TestProfileUnionAndExactThreshold(t *testing.T) {
	p, err := ReadProfile(strings.NewReader("mode: atomic\nm/main.go:1.1,2.2 99999 2\nm/main.go:3.1,3.2 1 0\nm/main.go:1.1,2.2 99999 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := summarize([]Package{{ImportPath: "m", GoFiles: []string{"main.go"}}}, p, map[string]bool{"m": true})
	requireReport(t, r, err, false)
	assertCount(t, r, "package", "m", 99999, 100000)
	other := Profile{}
	for b := range p {
		other[b] = true
	}
	if err := p.Merge(other); err != nil {
		t.Fatal(err)
	}
	r, err = summarize(nil, p, map[string]bool{"m": true})
	requireReport(t, r, err, true)
}
func TestBadProfiles(t *testing.T) {
	records := []string{"", "mode: wrong\n", "mode: set\nbroken\n", "mode: set\nm 1 1\n", "mode: set\nm:1,2 1 1\n", "mode: set\nm:0.1,2.1 1 1\n", "mode: set\nm:2.1,1.1 1 1\n", "mode: set\nm:1.1,1.2 -1 1\n", "mode: set\nm:1.1,1.2 1 -1\n"}
	for _, raw := range records {
		if _, err := ReadProfile(strings.NewReader(raw)); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
func TestConflictingProfiles(t *testing.T) {
	b := Block{File: "m/a.go", StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 2, Statements: 1}
	p := Profile{b: true}
	b.Statements = 2
	if err := p.Merge(Profile{b: false}); err == nil {
		t.Fatal("conflict accepted")
	}
}
func TestOverflowAndEmptyCounts(t *testing.T) {
	p := Profile{{File: "m/a.go", StartLine: 1, Statements: math.MaxInt64}: true, {File: "m/a.go", StartLine: 2, Statements: 1}: true}
	if _, err := summarize(nil, p, nil); err == nil {
		t.Fatal("overflow accepted")
	}
	if (Count{}).Percent() != "no executable statements" {
		t.Fatal("empty denominator")
	}
}
func TestInventoryRejectsMissingAndUnknownData(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package m\nfunc Value() int { return 1 }\n")
	packages := []Package{{ImportPath: "m", Dir: root, GoFiles: []string{"a.go"}}}
	if err := validateInventory(packages, Profile{}); err == nil {
		t.Fatal("missing coverage accepted")
	}
	write(t, root, "a.go", "package m\nconst Value=1\n")
	if err := validateInventory(packages, Profile{}); err != nil {
		t.Fatal(err)
	}
	if err := validateInventory(packages, Profile{{File: "unknown/a.go"}: true}); err == nil {
		t.Fatal("unknown coverage accepted")
	}
}
func TestEntriesCannotOmitCommands(t *testing.T) {
	packages := []Package{{ImportPath: "m/cmd", Name: "main"}, {ImportPath: "m/lib", Name: "lib"}}
	got, err := entries(packages, []string{"m/lib"})
	if err != nil || !got["m/cmd"] || !got["m/lib"] {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := entries(packages, []string{"unknown"}); err == nil {
		t.Fatal("unknown entry accepted")
	}
}
func TestUnstableAuditIsNotRedundancy(t *testing.T) {
	b := Block{File: "m/a.go", Statements: 1}
	groups := []testGroup{{Name: "stable", Stable: true, Covered: map[Block]bool{b: true}}, {Name: "unstable", Stable: false, Covered: map[Block]bool{b: true}}}
	text := strings.Join(overlap(groups), "\n")
	if strings.Contains(text, "no unique") || !strings.Contains(text, "unstable coverage") {
		t.Fatal(text)
	}
}
func TestAuditSubtractsSharedSetup(t *testing.T) {
	shared := Block{File: "m/a.go", StartLine: 1, Statements: 1}
	body := Block{File: "m/a.go", StartLine: 2, Statements: 1}
	got := contribution(Profile{shared: true, body: true}, Profile{shared: true})
	if len(got) != 1 || !got[body] {
		t.Fatal(got)
	}
}

func requireReport(t *testing.T, r Report, err error, pass bool) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if (len(r.Failures) == 0) != pass {
		t.Fatalf("%+v", r)
	}
}
