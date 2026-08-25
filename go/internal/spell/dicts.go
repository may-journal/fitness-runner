package spell

// cspell:ignore wordlists

import (
	_ "embed"
	"strings"
	"sync"
)

// The embedded base dictionaries are committed plain-text wordlists derived
// from the cspell dictionary packages; each file's '#'-prefixed provenance
// header names its sources and licenses. The Node generator was removed in the
// npm-free migration — a Go regeneration tool is deferred (see
// docs/plans/archive/02-npm-free.md). Each file is that header followed by the
// words: lowercased, deduplicated, sorted, one per line — the committed lists
// are pre-sorted, which is the guarantee the binary search below relies on (and
// TestEmbeddedDictsSearchable re-verifies byte-wise).
//
// Each file embeds as its own string so Lookup binary-searches the embedded
// bytes in place: no parsing, no index build, and no per-word allocation at
// process start — the startup cost of the old eager 217k-entry map.

//go:embed dict/en-us.txt
var dictEnUS string

//go:embed dict/software-terms.txt
var dictSoftwareTerms string

//go:embed dict/golang.txt
var dictGolang string

//go:embed dict/node.txt
var dictNode string

//go:embed dict/npm.txt
var dictNpm string

//go:embed dict/typescript.txt
var dictTypescript string

//go:embed dict/html.txt
var dictHTML string

//go:embed dict/companies.txt
var dictCompanies string

//go:embed dict/aws.txt
var dictAWS string

//go:embed dict/css.txt
var dictCSS string

//go:embed dict/fullstack.txt
var dictFullstack string

//go:embed dict/filetypes.txt
var dictFiletypes string

//go:embed dict/shell.txt
var dictShell string

//go:embed dict/git.txt
var dictGit string

// embeddedDicts lists every embedded wordlist, biggest hit rates first
// (en-us resolves most natural-language lookups). TestEmbeddedDictNames pins
// this table to dict/*.txt on disk so a newly generated list cannot be
// silently left out.
var embeddedDicts = []struct{ name, data string }{
	{"en-us.txt", dictEnUS},
	{"software-terms.txt", dictSoftwareTerms},
	{"golang.txt", dictGolang},
	{"node.txt", dictNode},
	{"npm.txt", dictNpm},
	{"typescript.txt", dictTypescript},
	{"html.txt", dictHTML},
	{"companies.txt", dictCompanies},
	{"aws.txt", dictAWS},
	{"css.txt", dictCSS},
	{"fullstack.txt", dictFullstack},
	{"filetypes.txt", dictFiletypes},
	{"shell.txt", dictShell},
	{"git.txt", dictGit},
}

// embeddedDict looks words up directly in the embedded wordlists, memoizing
// results because natural language repeats heavily. The memo is a sync.Map:
// the engine runs lookups concurrently under par.Map, and the read-mostly,
// write-once-per-key pattern is exactly its sweet spot — a mutex-guarded map ties
// it serially but measured ~40x slower under parallel lookups.
type embeddedDict struct {
	memo sync.Map // word -> bool
}

// embeddedBase is the process-wide embedded dictionary; its zero value is
// ready, so referencing it costs nothing at startup.
var embeddedBase embeddedDict

// NewEmbeddedChecker returns a Checker backed by the embedded base
// dictionaries, ready before any dictionary byte has been touched.
func NewEmbeddedChecker() *Checker {
	c := NewChecker()
	c.base = &embeddedBase
	return c
}

// Lookup reports whether word is an embedded dictionary entry. Words are
// matched exactly as stored (lowercase; callers fold first).
func (d *embeddedDict) Lookup(word string) bool {
	if hit, ok := d.memo.Load(word); ok {
		return hit.(bool)
	}
	found := searchEmbedded(word)
	d.memo.Store(word, found)
	return found
}

// searchEmbedded binary-searches every embedded wordlist for word.
func searchEmbedded(word string) bool {
	for _, ed := range embeddedDicts {
		if searchLines(ed.data, word) {
			return true
		}
	}
	return false
}

// searchLines reports whether target is exactly one of data's sorted lines:
// classic binary search over byte offsets — probe the middle, snap to the
// containing line, and narrow. lo is always a line start, so the containing
// line never begins before lo and the range always shrinks.
func searchLines(data, target string) bool {
	lo, hi := 0, len(data)
	for lo < hi {
		start := lineStart(data, (lo+hi)/2)
		end := lineEnd(data, start)
		if lineBelow(data[start:end], target) {
			lo = end + 1
		} else {
			hi = start
		}
	}
	return lo < len(data) && data[lo:lineEnd(data, lo)] == target
}

// lineBelow reports whether a probed line sorts strictly before target.
// Header comments ('#'-prefixed) appear only before the first word and words
// never start with '#', so ranking them below every target keeps the
// predicate monotone; blank lines never occur but would rank below too.
func lineBelow(line, target string) bool {
	return line == "" || line[0] == '#' || line < target
}

// lineStart returns the offset of the first byte of the line containing i.
func lineStart(data string, i int) int {
	for i > 0 && data[i-1] != '\n' {
		i--
	}
	return i
}

// lineEnd returns the offset just past the last byte of the line starting at
// start (its '\n', or end of data for an unterminated final line).
func lineEnd(data string, start int) int {
	if n := strings.IndexByte(data[start:], '\n'); n >= 0 {
		return start + n
	}
	return len(data)
}
