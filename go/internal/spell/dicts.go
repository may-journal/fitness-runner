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
		embeddedWords = make(map[string]struct{}, 220000)
		entries, err := dictFiles.ReadDir("dict")
		if err != nil {
			return
		}
		for _, entry := range entries {
			data, err := dictFiles.ReadFile("dict/" + entry.Name())
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				word := strings.TrimSpace(scanner.Text())
				if word != "" && !strings.HasPrefix(word, "#") {
					embeddedWords[word] = struct{}{}
				}
			}
		}
	})
	return embeddedWords
}
