// Package wordlist merges the spell-check lists in dict/ into words.txt, the
// one sorted list the spell package embeds. It never imports spell, so
// words.txt can be rebuilt even when it is missing.
package wordlist

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// Name is the merged list's file name in the spell folder.
const Name = "words.txt"

// Write rewrites words.txt in the spell folder dir from the lists in its
// dict folder. It writes nothing, and fails, when Render does.
func Write(dir string) error {
	data, err := Render(dir)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Name), data, 0o644)
}

// Render returns words.txt for the spell folder dir: a header naming the
// lists, then every word they hold, sorted by byte order and unique, one per
// line. It fails when a language module names a list dict/ lacks.
func Render(dir string) ([]byte, error) {
	lists, err := dictLists(dir)
	if err != nil {
		return nil, err
	}
	words, err := readWords(dir, lists)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("# words.txt — written by `make words`; do not edit.\n")
	fmt.Fprintf(&b, "# Merged from dict/%s.\n", strings.Join(lists, ", dict/"))
	fmt.Fprintf(&b, "# %d words, sorted by byte order, one per line.\n", len(words))
	for _, w := range words {
		b.WriteString(w + "\n")
	}
	return []byte(b.String()), nil
}

// dictLists returns the dict/ lists the language registry names, in its
// order.
func dictLists(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "dict", "*.txt"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, filepath.Base(m))
	}
	return lang.Dicts(names, langs.All), nil
}

// readWords reads every word of the named lists, sorted and unique; '#'
// header lines and blank lines are not words. A carriage return a CRLF
// checkout left is not part of a word.
func readWords(dir string, lists []string) ([]string, error) {
	var words []string
	dict := os.DirFS(filepath.Join(dir, "dict"))
	for _, name := range lists {
		data, err := fs.ReadFile(dict, name)
		if err != nil {
			return nil, fmt.Errorf("a language module names %s, which dict/ lacks", name)
		}
		words = slices.AppendSeq(words, wordLines(string(data)))
	}
	slices.Sort(words)
	return slices.Compact(words), nil
}

// wordLines yields the lines of a list that are words.
func wordLines(data string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for line := range strings.SplitSeq(data, "\n") {
			line = strings.TrimSuffix(line, "\r")
			if line != "" && line[0] != '#' && !yield(line) {
				return
			}
		}
	}
}
