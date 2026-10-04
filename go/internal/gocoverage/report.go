package gocoverage

import (
	"fmt"
	"math"
	"path"
	"sort"
	"strings"
)

type Count struct{ Covered, Total int64 }
type Row struct {
	Kind, Name string
	Count
	Entry bool
}
type Report struct {
	Rows           []Row
	Gaps, Findings []string
	Failures       []string
}

func summarize(packages []Package, p Profile, entry map[string]bool) (Report, error) {
	counts := map[string]Count{}
	var report Report
	for b, hit := range p {
		if err := addBlock(counts, b, hit); err != nil {
			return report, err
		}
		report.addGap(b, hit)
	}
	for _, pkg := range packages {
		seedCounts(counts, pkg)
	}
	report.addRows(counts, entry)
	report.sort()
	return report, nil
}
func entryRow(kind, name string, entries map[string]bool) bool {
	if kind == "file" {
		return entries[packageFor(name)]
	}
	return kind == "package" && entries[name]
}
func seedCounts(counts map[string]Count, p Package) {
	for _, file := range append(p.GoFiles, p.CgoFiles...) {
		key := "file " + p.ImportPath + "/" + file
		counts[key] = counts[key]
	}
	key := "package " + p.ImportPath
	counts[key] = counts[key]
}
func addBlock(counts map[string]Count, b Block, hit bool) error {
	keys := blockKeys(b)
	for _, key := range keys {
		c := counts[key]
		if b.Statements > math.MaxInt64-c.Total {
			return fmt.Errorf("statement count overflow in %s", b.File)
		}
		c.Total += b.Statements
		if hit {
			c.Covered += b.Statements
		}
		counts[key] = c
	}
	return nil
}
func (r *Report) sort() {
	sort.Slice(r.Rows, func(i, j int) bool { return r.Rows[i].Name+" "+r.Rows[i].Kind < r.Rows[j].Name+" "+r.Rows[j].Kind })
	sort.Strings(r.Gaps)
	sort.Strings(r.Failures)
	sort.Strings(r.Findings)
}
func (r Report) Lines() []string {
	var lines []string
	for _, row := range r.Rows {
		target := "target 100%"
		if row.Entry {
			target = "required 100%"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %d/%d statements (%s); %s", row.Kind, row.Name, row.Covered, row.Total, row.Percent(), target))
	}
	lines = append(lines, r.Gaps...)
	return append(lines, r.Findings...)
}
func (c Count) Percent() string {
	if c.Total == 0 {
		return "no executable statements"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(c.Covered)/float64(c.Total))
}

func (report *Report) addRows(counts map[string]Count, entry map[string]bool) {
	for key, c := range counts {
		kind, name, _ := strings.Cut(key, " ")
		row := Row{Kind: kind, Name: name, Count: c, Entry: entryRow(kind, name, entry)}
		report.Rows = append(report.Rows, row)
		if row.Entry && c.Covered != c.Total {
			report.Failures = append(report.Failures, fmt.Sprintf("entry %s: %d/%d statements; required 100%%", name, c.Covered, c.Total))
		}
	}
}

func blockKeys(b Block) []string {
	keys := []string{"file " + b.File, "package " + packageFor(b.File)}
	for folder := path.Dir(b.File); folder != "." && folder != "/"; folder = path.Dir(folder) {
		keys = append(keys, "folder "+folder)
	}
	return keys
}

func (report *Report) addGap(b Block, hit bool) {
	if !hit && b.Statements > 0 {
		report.Gaps = append(report.Gaps, fmt.Sprintf("%s:%d.%d-%d.%d: %d uncovered statements", b.File, b.StartLine, b.StartColumn, b.EndLine, b.EndColumn, b.Statements))
	}
}

func (r Report) FileCount() int {
	n := 0
	for _, row := range r.Rows {
		if row.Kind == "file" {
			n++
		}
	}
	return n
}
