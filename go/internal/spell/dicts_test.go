package spell

// cspell:ignore packagejson borwn quik

import "testing"

func TestEmbeddedWords(t *testing.T) {
	words := EmbeddedWords()
	if len(words) < 150000 {
		t.Fatalf("embedded dictionary suspiciously small: %d words", len(words))
	}
	present := []string{
		"the", "fitness", "typescript", "readme", "changelog", "goroutine",
		"npm", "don't", "prettier-plugin-packagejson",
	}
	for _, w := range present {
		if _, ok := words[w]; !ok {
			t.Errorf("expected embedded word %q", w)
		}
	}
	absent := []string{"borwn", "", "# comment"}
	for _, w := range absent {
		if _, ok := words[w]; ok {
			t.Errorf("did not expect embedded word %q", w)
		}
	}
}

func TestEmbeddedWordsFeedChecker(t *testing.T) {
	c := NewChecker(EmbeddedWords())
	if issues := c.CheckText("The quick brown fox jumps over the lazy dog."); len(issues) != 0 {
		t.Fatalf("expected clean pangram, got %v", issues)
	}
	issues := c.CheckText("teh quik borwn fox")
	if len(issues) != 2 || issues[0].Word != "quik" || issues[1].Word != "borwn" {
		t.Fatalf("expected quik and borwn flagged, got %v", issues)
	}
}
