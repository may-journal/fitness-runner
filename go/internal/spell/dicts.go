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

// The embedded dictionaries are committed cspell wordlists: a '#' header, then
// sorted lowercase words that the binary search relies on.
//
// Every file in dict/ is embedded, so a new wordlist needs no list entry.

//go:embed dict/*.txt
var dictFiles embed.FS

// embeddedDicts reads the wordlists the language registry names, biggest first.
// A word found in any list is known, so the order changes only speed.
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

// embeddedDict looks words up in the embedded wordlists, memoizing in a
// sync.Map, since a mutex-guarded map measured ~40x slower in parallel.
type embeddedDict struct {
	memo sync.Map // maps each word to whether the lists know it
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

// searchLines reports whether target is one of data's sorted lines, by binary
// search over byte offsets snapped to line starts.
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
// Header lines starting with '#' rank below every target, which keeps the
// predicate monotone because they precede the words.
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

// lineEnd returns the offset of the '\n' ending the line at start, or the end
// of data for an unterminated final line.
func lineEnd(data string, start int) int {
	if n := strings.IndexByte(data[start:], '\n'); n >= 0 {
		return start + n
	}
	return len(data)
}
