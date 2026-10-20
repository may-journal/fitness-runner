package spell

import (
	_ "embed"
	"sort"
	"strings"
	"sync"
)

// The base dictionaries are committed plain-text wordlists in dict/, derived
// from the cspell dictionary packages; each file's '#'-prefixed provenance
// header names its sources and licenses. The Node generator was removed in
// the npm-free migration — a Go regeneration tool is deferred (see
// docs/plans/archive/02-npm-free.md).
//
// The words generator (package wordlist) merges every list the language
// registry names into words.txt: one sorted, unique list, so a lookup is one
// binary search instead of one per list. Run `make words` after changing a
// list in dict/; `make words-check` in CI fails while words.txt is stale.

//go:generate go run gen_words.go

//go:embed words.txt
var words string

// wordStarts is the offset table of words.txt, built on the first lookup:
// the start of each word line, then the end of the text. wordlist.Write ends
// every line with a newline and leads with '#' header lines, which are left out.
// A process that spells nothing never builds it.
var wordStarts = sync.OnceValue(func() []int32 {
	starts := make([]int32, 0, strings.Count(words, "\n")+1)
	for i := 0; i < len(words); {
		if words[i] != '#' {
			starts = append(starts, int32(i))
		}
		nl := strings.IndexByte(words[i:], '\n')
		if nl < 0 {
			panic("spell: words.txt must end with a newline; rerun go generate")
		}
		i += nl + 1
	}
	return append(starts, int32(len(words)))
})

// embeddedDict looks words up directly in the embedded wordlist, memoizing
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

// searchEmbedded binary-searches the merged wordlist for word through its
// offset table.
func searchEmbedded(word string) bool {
	starts := wordStarts()
	n := len(starts) - 1
	i := sort.Search(n, func(i int) bool { return wordAt(starts, i) >= word })
	return i < n && wordAt(starts, i) == word
}

// wordAt returns word i of the merged list, without its newline.
func wordAt(starts []int32, i int) string {
	return words[starts[i] : starts[i+1]-1]
}
