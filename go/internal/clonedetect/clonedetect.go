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
// and never contributes duplicated lines. Window hashing is pure per-file
// work (tokens hash directly, no shared intern table), so it fans out on
// the par pool; the cheap counting and marking passes below run serially
// over the in-order results, keeping the outcome deterministic.
func Detect(files []File, opt Options) Stats {
	k := opt.MinTokens
	hashes := par.Map(len(files), 0, func(i int) []uint64 {
		return windowHashes(files[i].Tokens, k)
	})
	var stats Stats
	counts := make(map[uint64]int)
	for fi, f := range files {
		stats.TotalLines += f.Lines
		for _, h := range hashes[fi] {
			counts[h]++
		}
	}
	seen := make(map[uint64]bool)
	for fi, f := range files {
		dup := dupWindows(hashes[fi], counts, seen)
		stats.DuplicatedLines += dupLineCount(f.Tokens, dup, k, opt.MinLines)
	}
	return stats
}

// dupWindows returns the positions (ascending) of windows in hs whose hash
// occurs at least twice across all files, skipping each hash's first
// occurrence — the "original" copy stays free. It records first sightings
// in seen, which persists across the per-file calls.
func dupWindows(hs []uint64, counts map[uint64]int, seen map[uint64]bool) []int {
	var dup []int
	for pos, h := range hs {
		if counts[h] < 2 {
			continue
		}
		if !seen[h] {
			seen[h] = true
			continue
		}
		dup = append(dup, pos)
	}
	return dup
}

// hashBase is both the polynomial rolling-hash base and the FNV-64 prime.
const hashBase = 1099511628211

// fnvOffset is the FNV-64 offset basis.
const fnvOffset uint64 = 14695981039346656037

// windowHashes hashes each token value independently and returns the
// rolling polynomial hash of every k-token window, or nil when the file
// is shorter than one window. It touches no shared state, so files hash
// in parallel.
//
// Only hash equality matters to detection: equal token sequences always
// produce equal window hashes, exactly as under the old intern table. A
// spurious match now additionally requires two distinct token values
// sharing one FNV-1a 64 value (previously intern ids were distinct by
// construction), but that is birthday-bounded — even a million distinct
// token values collide with probability under 10^-7 — so, like the
// polynomial hash itself, token collisions stay effectively impossible.
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

// tokenHashes maps each token to the FNV-1a 64 hash of its value; equal
// values hash equal everywhere, so no cross-file intern table is needed.
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

// dupLineCount merges contiguous duplicated windows (ascending positions)
// into spans, keeps spans of at least minLines lines, and returns how many
// distinct lines those spans cover — no line counts twice per file.
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
