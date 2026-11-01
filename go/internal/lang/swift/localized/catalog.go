package localized

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// catalogExt ends a String Catalog file, which Xcode keeps as JSON.
const catalogExt = ".xcstrings"

// catalogFile is the part of a String Catalog the check reads.
type catalogFile struct {
	SourceLanguage string                  `json:"sourceLanguage"`
	Strings        map[string]catalogEntry `json:"strings"`
}

// catalogEntry is one key. A stale entry is one the code no longer uses,
// and an entry marked not to translate is no label.
type catalogEntry struct {
	ExtractionState string                  `json:"extractionState"`
	ShouldTranslate *bool                   `json:"shouldTranslate"`
	Localizations   map[string]localization `json:"localizations"`
}

// localization is one language's value, or its plural and device
// variations, each of which is a localization itself.
type localization struct {
	StringUnit *struct {
		Value string `json:"value"`
	} `json:"stringUnit"`
	Variations map[string]map[string]localization `json:"variations"`
}

// entries maps each source-language form of a live catalog entry to its key,
// the first when forms collide.
type entries map[string]string

// readCatalogs reads every catalog file into one entries map.
func readCatalogs(root string, files []string) (entries, error) {
	all := entries{}
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return nil, err
		}
		var catalog catalogFile
		if err := json.Unmarshal(raw, &catalog); err != nil {
			return nil, fmt.Errorf("%s is not a String Catalog: %w", file, err)
		}
		catalog.addTo(all)
	}
	return all, nil
}

// addTo adds the catalog's live entries to all, keys in order.
func (c catalogFile) addTo(all entries) {
	keys := make([]string, 0, len(c.Strings))
	for key := range c.Strings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if entry := c.Strings[key]; entry.live(key) {
			all.add(key, entry.forms(key, c.SourceLanguage))
		}
	}
}

// live reports an entry that labels something: not the empty key, not
// stale, and not marked to stay untranslated.
func (e catalogEntry) live(key string) bool {
	untranslated := e.ShouldTranslate != nil && !*e.ShouldTranslate
	return key != "" && e.ExtractionState != "stale" && !untranslated
}

// add maps each form with words to key, keeping a form's first key.
func (all entries) add(key string, forms []string) {
	for _, form := range forms {
		if _, seen := all[form]; !seen && hasWords(form) {
			all[form] = key
		}
	}
}

// forms returns the entry's source-language values; an entry with none
// reads as its key, as Xcode shows a key extracted from source.
func (e catalogEntry) forms(key, language string) []string {
	source, ok := e.Localizations[language]
	if !ok {
		return []string{key}
	}
	if forms := source.forms(); len(forms) > 0 {
		return forms
	}
	return []string{key}
}

// forms collects the localization's value and its variations' values.
func (l localization) forms() []string {
	var out []string
	if l.StringUnit != nil {
		out = append(out, l.StringUnit.Value)
	}
	for _, cases := range l.Variations {
		for _, variation := range cases {
			out = append(out, variation.forms()...)
		}
	}
	return out
}

// hasWords reports whether text holds a letter, so a reader sees a word
// a translation would change.
func hasWords(text string) bool {
	return strings.IndexFunc(text, unicode.IsLetter) >= 0
}
