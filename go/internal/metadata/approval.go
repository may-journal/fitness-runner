// cspell:ignore approv aproved dont doesnt didnt cant wont isnt wasnt arent
package metadata

import (
	"slices"
	"strings"
	"unicode"
)

// negations are the words that, just before an approving word, turn it into a refusal.
var negations = []string{"not", "no", "never", "yet", "cannot", "dont", "doesnt", "didnt", "cant", "wont", "isnt", "wasnt", "arent"}

// stems are the words within one edit of "approv" that hold a "v", so "approach" is none.
var stems = oneEdit("approv")

// saysApproved reports whether body has a word that is a form of approve, typos allowed.
// A word starting "un" or "dis", or just after a negation, does not count.
func saysApproved(body string) bool {
	clean := strings.NewReplacer("'", "", "’", "").Replace(strings.ToLower(body))
	words := strings.FieldsFunc(clean, func(r rune) bool { return !unicode.IsLetter(r) })
	for i, w := range words {
		if approving(w) && (i == 0 || !slices.Contains(negations, words[i-1])) {
			return true
		}
	}
	return false
}

// approving reports whether w, lowercase, starts with a stem: its first five to seven letters.
func approving(w string) bool {
	if strings.HasPrefix(w, "un") || strings.HasPrefix(w, "dis") {
		return false
	}
	return hasStem(w)
}

// hasStem reports whether one of w's first five to seven letters is a stem.
func hasStem(w string) bool {
	for n := 5; n <= 7 && n <= len(w); n++ {
		if stems[w[:n]] {
			return true
		}
	}
	return false
}

// oneEdit holds the words one deletion, swap, substitution or insertion from word that hold a "v".
func oneEdit(word string) map[string]bool {
	set := map[string]bool{word: true}
	for i := 0; i <= len(word); i++ {
		add(set, word[:i]+word[min(i+1, len(word)):])
		if i+1 < len(word) {
			add(set, word[:i]+word[i+1:i+2]+word[i:i+1]+word[i+2:])
		}
		for c := 'a'; c <= 'z'; c++ {
			add(set, word[:i]+string(c)+word[i:])
			add(set, word[:i]+string(c)+word[min(i+1, len(word)):])
		}
	}
	return set
}

// add puts w in set when it holds a "v".
func add(set map[string]bool, w string) {
	if strings.Contains(w, "v") {
		set[w] = true
	}
}
