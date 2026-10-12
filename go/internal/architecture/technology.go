package architecture

import (
	"strings"
	"unicode"
)

// technologyWords are the words a container's technology string may use
// for each language the evidence proves.
var technologyWords = map[string][]string{
	"Go":    strings.Fields("go golang"),
	"Swift": strings.Fields("swift swiftui objective-c ios macos watchos tvos visionos xcode"),
	"Node":  strings.Fields("node node.js nodejs javascript typescript bun deno"),
}

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
