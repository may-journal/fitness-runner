// Package clonedetect is the native duplicate-code engine: a generic lexer and
// rolling-hash window detector whose line accounting is modeled on jscpd.
package clonedetect

import "github.com/may-journal/fitness-runner/go/internal/par"

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
	// Files lists each file holding duplicated lines, in file order.
	Files []FileLines
}

// FileLines is how many duplicated lines one file holds.
type FileLines struct {
	Path  string
	Lines int
}

// Percentage returns duplicated lines as a percentage of total lines.
func (s Stats) Percentage() float64 {
	if s.TotalLines == 0 {
		return 0
	}
	return float64(s.DuplicatedLines) * 100 / float64(s.TotalLines)
}

// Detect returns the duplicated-line accounting across files in slice order,
// where each window's first occurrence stays free.
func Detect(files []File, opt Options) Stats {
	k := opt.MinTokens
	hashes := par.Map(len(files), 0, func(i int) []uint64 {
		return windowHashes(files[i].Tokens, k)
	})
	var stats Stats
	counts := countWindows(hashes)
	for fi, f := range files {
		stats.TotalLines += f.Lines
		dup := dupWindows(hashes[fi], counts)
		if lines := dupLineCount(f.Tokens, dup, k, opt.MinLines); lines > 0 {
			stats.DuplicatedLines += lines
			stats.Files = append(stats.Files, FileLines{Path: f.Path, Lines: lines})
		}
	}
	return stats
}

// countWindows counts each window hash across every file, in a map sized
// for every window up front so it never grows while filling.
func countWindows(hashes [][]uint64) map[uint64]int {
	windows := 0
	for _, hs := range hashes {
		windows += len(hs)
	}
	counts := make(map[uint64]int, windows)
	for _, hs := range hashes {
		for _, h := range hs {
			counts[h]++
		}
	}
	return counts
}

// dupWindows returns the ascending positions in hs of repeated hashes, marking
// each first sighting with count -1 so the original copy stays free.
func dupWindows(hs []uint64, counts map[uint64]int) []int {
	var dup []int
	for pos, h := range hs {
		switch c := counts[h]; {
		case c < 0:
			dup = append(dup, pos)
		case c >= 2:
			counts[h] = -1
		}
	}
	return dup
}

// hashBase is both the polynomial rolling-hash base and the FNV-64 prime.
const hashBase = 1099511628211

// fnvOffset is the FNV-64 offset basis.
const fnvOffset uint64 = 14695981039346656037

// windowHashes returns the rolling hash of every k-token window, or nil when
// the file is shorter, using no shared state.
func windowHashes(tokens []Token, k int) []uint64 {
	if len(tokens) < k {
		return nil
	}
	vals := tokenHashes(tokens)
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

// tokenHashes maps each token to the FNV-1a 64 hash of its value.
func tokenHashes(tokens []Token) []uint64 {
	vals := make([]uint64, len(tokens))
	for i, t := range tokens {
		vals[i] = fnv1a(t.Val)
	}
	return vals
}

// fnv1a is FNV-1a 64 over s, inlined rather than via hash/fnv to avoid a
// per-token allocation and interface call on this hot path.
func fnv1a(s string) uint64 {
	h := fnvOffset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= hashBase
	}
	return h
}

// dupLineCount returns the distinct lines covered by merged duplicate spans of
// at least minLines lines, so no line counts twice per file.
func dupLineCount(tokens []Token, wins []int, k, minLines int) int {
	lines := make(map[int]bool)
	for i := 0; i < len(wins); {
		next, end := mergeSpan(wins, i, k)
		startLine, endLine := tokens[wins[i]].Line, tokens[end].Line
		if endLine-startLine+1 >= minLines {
			for l := startLine; l <= endLine; l++ {
				lines[l] = true
			}
		}
		i = next
	}
	return len(lines)
}

// mergeSpan grows the duplicated span opening at wins[i]: contiguous
// windows — each starting no later than one past the current span end —
// extend it. It returns the index of the first window past the span and
// the last token index the span covers.
func mergeSpan(wins []int, i, k int) (next, end int) {
	end = wins[i] + k - 1
	next = i + 1
	for next < len(wins) && wins[next] <= end+1 {
		end = wins[next] + k - 1
		next++
	}
	return next, end
}
