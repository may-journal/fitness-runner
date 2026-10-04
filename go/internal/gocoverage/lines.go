package gocoverage

import (
	"slices"
	"sort"
)

type sourceLine struct {
	file string
	line int
}

func lineClaims(blocks []BlockClaim) []LineClaim {
	lines := map[sourceLine]*LineClaim{}
	for _, b := range blocks {
		addBlockLines(lines, b)
	}
	result := make([]LineClaim, 0, len(lines))
	for _, line := range lines {
		result = append(result, *line)
	}
	sort.Slice(result, func(i, j int) bool { return beforeLine(result[i], result[j]) })
	for i := range result {
		result[i].ID = i + 1
		result[i].Tests = uniqueNames(result[i].Tests)
		result[i].SetupFor = uniqueNames(result[i].SetupFor)
	}
	return result
}
func addBlockLines(lines map[sourceLine]*LineClaim, b BlockClaim) {
	end := b.EndLine
	if b.EndColumn == 1 && end > b.StartLine {
		end--
	}
	for number := b.StartLine; number <= end; number++ {
		key := sourceLine{b.File, number}
		if lines[key] == nil {
			lines[key] = &LineClaim{File: b.File, Line: number, Blocks: []int{}, Tests: []string{}, SetupFor: []string{}}
		}
		line := lines[key]
		line.Blocks = append(line.Blocks, b.ID)
		line.Tests = append(line.Tests, b.Tests...)
		line.SetupFor = append(line.SetupFor, b.SetupFor...)
	}
}
func beforeLine(a, b LineClaim) bool {
	if a.File == b.File {
		return a.Line < b.Line
	}
	return a.File < b.File
}
func uniqueNames(names []string) []string { slices.Sort(names); return slices.Compact(names) }

type testPair struct{ a, b string }

func pairClaims(lines []LineClaim, tests []TestClaim) []PairClaim {
	pairs := map[testPair][]int{}
	for _, line := range lines {
		linePairs(pairs, line)
	}
	blocks := map[string][]int{}
	for _, test := range tests {
		blocks[test.Name] = test.Blocks
	}
	result := make([]PairClaim, 0, len(pairs))
	for pair, shared := range pairs {
		result = append(result, PairClaim{A: pair.a, B: pair.b, SharedLines: shared, SharedBlocks: intersection(blocks[pair.a], blocks[pair.b])})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].A+"\x00"+result[i].B < result[j].A+"\x00"+result[j].B })
	return result
}
func linePairs(pairs map[testPair][]int, line LineClaim) {
	for i, a := range line.Tests {
		for _, b := range line.Tests[i+1:] {
			pair := testPair{a, b}
			pairs[pair] = append(pairs[pair], line.ID)
		}
	}
}
func intersection(a, b []int) []int {
	result := []int{}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			result = append(result, a[i])
			i++
			j++
		}
	}
	return result
}
