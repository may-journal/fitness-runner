package architecture

import (
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// technologyWords are the words a container's technology string may use
// for each language the evidence proves, from the language modules.
var technologyWords = lang.TechnologyWords(langs.All)

// namesTechnology reports whether a container's technology string, such
// as "Go" or "Swift 6, SwiftUI", names the evidence's language.
func namesTechnology(technology, language string) bool {
	for _, word := range technologyTokens(technology) {
		if contains(technologyWords[language], word) {
			return true
		}
	}
	return false
}

func technologyTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(".-+", r)
	})
}
