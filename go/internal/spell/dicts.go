package spell

// cspell:ignore wordlists

import (
	"bufio"
	"embed"
	"strings"
	"sync"
)

// The embedded base dictionaries are committed plain-text wordlists (sorted,
// lowercased, one word per line) generated from the installed cspell
// dictionary packages; see dict/generate.mjs for sources, licenses, and the
// regeneration command.
//
//go:embed dict/*.txt
var dictFiles embed.FS

var (
	embeddedOnce  sync.Once
	embeddedWords map[string]struct{}
)

// EmbeddedWords returns the union of the embedded base dictionaries, loaded
// once per process. Lines starting with '#' are provenance comments.
func EmbeddedWords() map[string]struct{} {
	embeddedOnce.Do(func() {
		embeddedWords = loadEmbeddedDicts()
	})
	return embeddedWords
}

// loadEmbeddedDicts reads every embedded dict/*.txt wordlist into one set; an
// unreadable directory or file contributes nothing, never an error.
func loadEmbeddedDicts() map[string]struct{} {
	words := make(map[string]struct{}, 220000)
	entries, err := dictFiles.ReadDir("dict")
	if err != nil {
		return words
	}
	for _, entry := range entries {
		data, err := dictFiles.ReadFile("dict/" + entry.Name())
		if err != nil {
			continue
		}
		addDictLines(string(data), words)
	}
	return words
}

// addDictLines adds each non-empty, non-comment ('#'-prefixed) wordlist line
// to words, trimmed of surrounding whitespace.
func addDictLines(data string, words map[string]struct{}) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") {
			words[word] = struct{}{}
		}
	}
}
