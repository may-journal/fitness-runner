package gocoverage

import (
	"cmp"
	"maps"
	"slices"
	"sort"
)

// AuditMap retains block identity while projecting ranges onto source lines.
// Line claims locate Go coverage blocks; they are not per-line execution counts.
type AuditMap struct {
	Version int         `json:"version"`
	Basis   string      `json:"basis"`
	Modules []ModuleMap `json:"modules"`
}
type ModuleMap struct {
	SourceFiles map[string]string `json:"sourceFiles"`
	Module      string            `json:"module"`
	Blocks      []BlockClaim      `json:"blocks"`
	Lines       []LineClaim       `json:"lines"`
	Tests       []TestClaim       `json:"tests"`
	Overlaps    []PairClaim       `json:"overlaps"`
}
type BlockClaim struct {
	ID          int      `json:"id"`
	File        string   `json:"file"`
	StartLine   int      `json:"startLine"`
	StartColumn int      `json:"startColumn"`
	EndLine     int      `json:"endLine"`
	EndColumn   int      `json:"endColumn"`
	Statements  int64    `json:"statements"`
	Tests       []string `json:"tests"`
	SetupFor    []string `json:"setupFor"`
}
type LineClaim struct {
	ID       int      `json:"id"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Blocks   []int    `json:"blocks"`
	Tests    []string `json:"tests"`
	SetupFor []string `json:"setupFor"`
}
type TestClaim struct {
	Name        string `json:"name"`
	Stable      bool   `json:"stable"`
	Blocks      []int  `json:"blocks"`
	SetupBlocks []int  `json:"setupBlocks"`
}
type PairClaim struct {
	A            string `json:"a"`
	B            string `json:"b"`
	SharedBlocks []int  `json:"sharedBlocks"`
	SharedLines  []int  `json:"sharedLines"`
}

func claimMap(groups []testGroup) ModuleMap {
	groups = slices.Clone(groups)
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	blocks := claimBlocks(groups)
	ids := make(map[Block]int, len(blocks))
	result := ModuleMap{Blocks: []BlockClaim{}, Tests: []TestClaim{}}
	for i, b := range blocks {
		ids[b] = i + 1
		result.Blocks = append(result.Blocks, describeBlock(i+1, b))
	}
	for _, group := range groups {
		result.addTest(group, ids)
	}
	result.Lines = lineClaims(result.Blocks)
	result.Overlaps = pairClaims(result.Lines, result.Tests)
	return result
}
func claimBlocks(groups []testGroup) []Block {
	all := map[Block]bool{}
	for _, g := range groups {
		if !g.Stable {
			continue
		}
		maps.Copy(all, g.Covered)
		maps.Copy(all, g.Setup)
	}
	result := make([]Block, 0, len(all))
	for b := range all {
		result = append(result, b)
	}
	slices.SortFunc(result, compareBlocks)
	return result
}
func compareBlocks(a, b Block) int {
	if n := cmp.Compare(a.File, b.File); n != 0 {
		return n
	}
	left := []int{a.StartLine, a.StartColumn, a.EndLine, a.EndColumn}
	right := []int{b.StartLine, b.StartColumn, b.EndLine, b.EndColumn}
	if n := slices.Compare(left, right); n != 0 {
		return n
	}
	return cmp.Compare(a.Statements, b.Statements)
}
func describeBlock(id int, b Block) BlockClaim {
	return BlockClaim{ID: id, File: b.File, StartLine: b.StartLine, StartColumn: b.StartColumn, EndLine: b.EndLine, EndColumn: b.EndColumn, Statements: b.Statements, Tests: []string{}, SetupFor: []string{}}
}
func (m *ModuleMap) addTest(group testGroup, ids map[Block]int) {
	test := TestClaim{Name: group.Name, Stable: group.Stable, Blocks: []int{}, SetupBlocks: []int{}}
	if group.Stable {
		test.Blocks = m.assign(group.Name, group.Covered, ids, false)
		test.SetupBlocks = m.assign(group.Name, group.Setup, ids, true)
	}
	m.Tests = append(m.Tests, test)
}
func (m *ModuleMap) assign(name string, covered map[Block]bool, ids map[Block]int, setup bool) []int {
	result := make([]int, 0, len(covered))
	for b := range covered {
		id := ids[b]
		result = append(result, id)
		if setup {
			m.Blocks[id-1].SetupFor = append(m.Blocks[id-1].SetupFor, name)
		} else {
			m.Blocks[id-1].Tests = append(m.Blocks[id-1].Tests, name)
		}
	}
	slices.Sort(result)
	return result
}
