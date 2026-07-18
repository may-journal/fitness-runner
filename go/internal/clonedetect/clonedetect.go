// Package clonedetect is the native engine behind the jscpd check: a
// generic code lexer, a rolling-hash window detector with jscpd's
// min-lines/min-tokens semantics, and the scan-scope helpers the check
// needs (doublestar ignore globs, batched .gitignore filtering).
//
// The lexer is deliberately generic, not per-language. Its boundaries:
//   - Comments stripped: // and # line comments, /* */ and <!-- --> block
//     comments (C-family, shell/Python, and HTML/XML syntaxes). SQL/Lua
//     "--" line comments are NOT stripped — they collide with the
//     decrement operator.
//   - String literals ('…', "…", `…`) normalize to one $str token whose
//     value keeps the literal's content but drops the quote style; content
//     is deliberately NOT collapsed away — real jscpd distinguishes string
//     values, and full collapse manufactures clones it never reports
//     (e.g. Go import blocks differing only in import paths). Single- and
//     double-quoted strings implicitly close at end of line, backtick
//     strings span lines, backslash escapes are honored but stay raw.
//   - Numbers collapse to a single $num token (a digit then any run of
//     [0-9A-Za-z_.], which covers hex, exponents, and digit separators).
//   - Identifiers and punctuation survive verbatim; whitespace vanishes.
//   - Regions between jscpd:ignore-start and jscpd:ignore-end comment
//     markers produce no tokens (the escape hatch).
//
// Duplicated-line accounting follows jscpd: every window of MinTokens
// normalized tokens is hashed; a window whose hash was seen earlier (same
// or another file) marks its occurrence as duplicated, contiguous marked
// windows merge into spans, and spans of at least MinLines lines
// contribute their lines — each line at most once per file, with each
// clone's first occurrence staying free, exactly like jscpd counts a
// clone pair's lines once.
package clonedetect

// File is one lexed source file ready for detection.
type File struct {
	Path   string
	Tokens []Token
	Lines  int
}

// Options are the clone thresholds, mirroring jscpd's --min-lines and
// --min-tokens flags: a clone must span at least MinLines lines and
// MinTokens normalized tokens.
type Options struct {
	MinLines  int
	MinTokens int
}

// Stats is the outcome of a detection pass over a set of files.
type Stats struct {
	TotalLines      int
	DuplicatedLines int
}

// Percentage returns duplicated lines as a percentage of total lines.
func (s Stats) Percentage() float64 {
	if s.TotalLines == 0 {
		return 0
	}
	return float64(s.DuplicatedLines) * 100 / float64(s.TotalLines)
}

// Detect finds duplicated regions across files (same-file and cross-file
// clones both count) and returns the line accounting. Files are processed
// in slice order; the first occurrence of any window is the "original"
// and never contributes duplicated lines.
func Detect(files []File, opt Options) Stats {
	k := opt.MinTokens
	var stats Stats
	ids := make(map[string]uint64)
	hashes := make([][]uint64, len(files))
	counts := make(map[uint64]int)
	for fi, f := range files {
		stats.TotalLines += f.Lines
		if len(f.Tokens) < k {
			continue
		}
		hashes[fi] = windowHashes(f.Tokens, k, ids)
		for _, h := range hashes[fi] {
			counts[h]++
		}
	}
	seen := make(map[uint64]bool)
	for fi, f := range files {
		var dup []int
		for pos, h := range hashes[fi] {
			if counts[h] < 2 {
				continue
			}
			if !seen[h] {
				seen[h] = true
				continue
			}
			dup = append(dup, pos)
		}
		stats.DuplicatedLines += dupLineCount(f.Tokens, dup, k, opt.MinLines)
	}
	return stats
}

// hashBase is the polynomial rolling-hash base (the FNV-64 prime).
const hashBase = 1099511628211

// windowHashes interns each token value to a mixed 64-bit id and returns
// the rolling polynomial hash of every k-token window.
func windowHashes(tokens []Token, k int, ids map[string]uint64) []uint64 {
	vals := make([]uint64, len(tokens))
	for i, t := range tokens {
		id, ok := ids[t.Val]
		if !ok {
			id = mix(uint64(len(ids)) + 1)
			ids[t.Val] = id
		}
		vals[i] = id
	}
	pow := uint64(1)
	for i := 0; i < k-1; i++ {
		pow *= hashBase
	}
	out := make([]uint64, len(tokens)-k+1)
	var h uint64
	for i := 0; i < k; i++ {
		h = h*hashBase + vals[i]
	}
	out[0] = h
	for i := k; i < len(tokens); i++ {
		h = (h-vals[i-k]*pow)*hashBase + vals[i]
		out[i-k+1] = h
	}
	return out
}

// mix is splitmix64: it spreads sequential intern ids across the 64-bit
// space so polynomial collisions are vanishingly unlikely.
func mix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// dupLineCount merges contiguous duplicated windows (ascending positions)
// into spans, keeps spans of at least minLines lines, and returns how many
// distinct lines those spans cover — no line counts twice per file.
func dupLineCount(tokens []Token, wins []int, k, minLines int) int {
	lines := make(map[int]bool)
	for i := 0; i < len(wins); {
		end := wins[i] + k - 1
		j := i + 1
		for j < len(wins) && wins[j] <= end+1 {
			end = wins[j] + k - 1
			j++
		}
		startLine, endLine := tokens[wins[i]].Line, tokens[end].Line
		if endLine-startLine+1 >= minLines {
			for l := startLine; l <= endLine; l++ {
				lines[l] = true
			}
		}
		i = j
	}
	return len(lines)
}
