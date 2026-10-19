package spell

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

// WriteWords rewrites words.txt in the spell folder dir from the lists in
// its dict folder: a header naming them, then every word they hold, sorted
// by byte order and unique, one per line. It writes nothing, and fails,
// when a language module names a list dict/ lacks.
func WriteWords(dir string) error {
	lists, err := dictLists(dir)
	if err != nil {
		return err
	}
	words, err := readWords(dir, lists)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# words.txt — written by `fitness-release words`; do not edit.\n")
	fmt.Fprintf(&b, "# Merged from dict/%s.\n", strings.Join(lists, ", dict/"))
	fmt.Fprintf(&b, "# %d words, sorted by byte order, one per line.\n", len(words))
	for _, w := range words {
		b.WriteString(w + "\n")
	}
	return os.WriteFile(filepath.Join(dir, "words.txt"), []byte(b.String()), 0o644)
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
// header lines and blank lines are not words.
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
			if line != "" && line[0] != '#' && !yield(line) {
				return
			}
		}
	}
}
