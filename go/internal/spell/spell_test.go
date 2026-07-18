package spell

// cspell:ignore packagejson deadbeefcafe wouldn'tx XQRZVs

import (
	"fmt"
	"reflect"
	"testing"
)

// checker builds a Checker over just the given words — engine tests never
// depend on the embedded dictionaries.
func checker(words ...string) *Checker {
	c := NewChecker()
	c.AddWords(words)
	return c
}

func issueStrings(issues []Issue) []string {
	var out []string
	for _, is := range issues {
		out = append(out, fmt.Sprintf("%s@%d:%d", is.Word, is.Line, is.Col))
	}
	return out
}

func TestCheckText(t *testing.T) {
	cases := []struct {
		name string
		dict []string
		text string
		want []string
	}{
		{"clean text passes", []string{"good", "words"}, "good words", nil},
		{"short words never flagged", nil, "teh ab c", nil},
		{"unknown word flagged with position", []string{"good"}, "good borwn", []string{"borwn@1:6"}},
		{"case-insensitive lookup", []string{"fitness"}, "Fitness FITNESS fitness", nil},
		{"camelCase splits and flags the part", []string{"word"}, "myBorwnWord", []string{"Borwn@1:3"}},
		{"upper-run split HTMLElement", []string{"html", "element"}, "HTMLElement", nil},
		{"upper-run split flags unknown tail", []string{"html"}, "HTMLBorwn", []string{"Borwn@1:5"}},
		{"caps plural with known base", []string{"url"}, "URLs URLs", nil},
		{"caps plural with short base", nil, "CLIs IDs", nil},
		{"caps plural with unknown base flags whole", nil, "XQRZVs", []string{"XQRZVs@1:1"}},
		{"all-caps unknown flags whole", nil, "BORWN", []string{"BORWN@1:1"}},
		{"possessive of known word", []string{"css"}, "CSS's", nil},
		{"contraction in dictionary", []string{"don't"}, "don't", nil},
		{"unknown contraction flags whole token", nil, "wouldn'tx", []string{"wouldn'tx@1:1"}},
		{"compound dictionary entry wins whole", []string{"use", "prettier-plugin-packagejson"},
			"use `prettier-plugin-packagejson`", nil},
		{"digits split words", nil, "borwn256 sha256", []string{"borwn@1:1"}},
		{"snake and kebab split", []string{"snake", "case"}, "snake_case-borwn", []string{"borwn@1:12"}},
		{"url masked", []string{"see"}, "see https://example.com/borwnpage.html", nil},
		{"email masked", nil, "bob@borwnmail.com", nil},
		{"c-style hex masked", nil, "0xdeadbeef", nil},
		{"letters-only hex flagged", nil, "deadbeefcafe", []string{"deadbeefcafe@1:1"}},
		{"digit-bearing hex run masked", nil, "deadbeef0123456789abcdef0123456789abcdef", nil},
		{"repeated char never flagged", nil, "xxxx XXXXX", nil},
		{"line and column across lines", nil, "ok\nbad borwn", []string{"borwn@2:5"}},
		{"rune columns after accents", []string{"café"}, "café borwn", []string{"borwn@1:6"}},
		{"accented word case folds", []string{"naïve"}, "Naïve", nil},
		{"escape sequence splits words", []string{"word"}, `say \nword`, nil},
		{"ignore directive adds doc words", nil, "<!-- cspell:ignore borwn -->\nborwn", nil},
		{"words directive adds doc words", nil, "// cspell:words borwn\nborwn", nil},
		{"disable-line masks its line", nil, "borwn // cspell:disable-line", nil},
		{"disable-next masks next line", nil, "// cspell:disable-next\nborwn\nborwn", []string{"borwn@3:1"}},
		{"disable enable block", nil, "borwn\n<!-- cspell:disable -->\nborwn\n<!-- cspell:enable -->\nborwn",
			[]string{"borwn@1:1", "borwn@5:1"}},
		{"unterminated disable masks to end", nil, "cspell:disable\nborwn", nil},
		{"prose colon is not a directive", []string{"cspell", "staged", "paths"}, "cspell: staged paths borwn",
			[]string{"borwn@1:22"}},
		{"project word beats nothing else", []string{"borwn"}, "borwn", nil},
		{"empty text", nil, "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := issueStrings(checker(tc.dict...).CheckText(tc.text))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("issues = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSplitSubWords(t *testing.T) {
	cases := []struct {
		token string
		want  []string
	}{
		{"plain", []string{"plain"}},
		{"myBorwnWord", []string{"my", "Borwn", "Word"}},
		{"HTMLElement", []string{"HTML", "Element"}},
		{"XMLHttpRequest", []string{"XML", "Http", "Request"}},
		{"URLs", []string{"URLs"}},
		{"LSTMs", []string{"LSTMs"}},
		{"parseURL", []string{"parse", "URL"}},
		{"BORWN", []string{"BORWN"}},
	}
	for _, tc := range cases {
		t.Run(tc.token, func(t *testing.T) {
			var got []string
			for _, s := range splitSubWords(tc.token) {
				got = append(got, tc.token[s.start:s.end])
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitSubWords(%q) = %v, want %v", tc.token, got, tc.want)
			}
		})
	}
}

func TestAddWordsNormalizes(t *testing.T) {
	c := checker("EOTP", "Wardley’s")
	if !c.known("eotp", nil) {
		t.Fatal("expected EOTP to be known lowercased")
	}
	if !c.known("wardley's", nil) {
		t.Fatal("expected curly apostrophe to normalize")
	}
}
