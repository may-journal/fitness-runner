package spell

// cspell:ignore wordlists

import (
	"cmp"
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// The embedded base dictionaries are committed plain-text wordlists derived
// from the cspell dictionary packages; each file's '#'-prefixed provenance
// header names its sources and licenses. The Node generator was removed in the
// npm-free migration — a Go regeneration tool is deferred (see
// docs/plans/archive/02-npm-free.md). Each file is that header followed by the
// words: lowercased, deduplicated, sorted, one per line — the committed lists
// are pre-sorted, which is the guarantee the binary search below relies on.
//
// Every file in dict/ is embedded, so a new wordlist needs no list entry.
// Lookup binary-searches each list's text in place: no parsing, no index
// build, and no per-word allocation. The lists are read once, on the first
// lookup, so a process that spells nothing never touches them.

//go:embed dict/*.txt
var dictFiles embed.FS

// embeddedDicts reads the wordlists the language registry names: the
// repo-wide lists no module claims, and each module's. They are searched
// biggest first, so en-us, which resolves most natural-language lookups,
// leads. A word found in any list is known, so the order changes speed,
// never a verdict.
var embeddedDicts = sync.OnceValue(func() []string {
	matches, err := fs.Glob(dictFiles, "dict/*.txt")
	if err != nil {
		panic("spell: " + err.Error())
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, path.Base(m))
	}
	var lists []string
	for _, name := range lang.Dicts(names, langs.All) {
		data, err := dictFiles.ReadFile("dict/" + name)
		if err != nil {
			panic("spell: a language module claims " + name + ", which dict/ lacks")
		}
		lists = append(lists, string(data))
	}
	slices.SortStableFunc(lists, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return lists
})

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
	for _, data := range embeddedDicts() {
		if searchLines(data, word) {
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
