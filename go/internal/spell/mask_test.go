package spell

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// referenceMaskPatterns preserves the whole-document matching contract.
func referenceMaskPatterns(text string, masked []bool) {
	for _, pattern := range ignorePatterns {
		for _, match := range pattern.re.FindAllStringIndex(text, -1) {
			mark(masked, match[0], match[1])
		}
	}
	for _, match := range hexRun.FindAllStringIndex(text, -1) {
		if strings.ContainsAny(text[match[0]:match[1]], "0123456789") {
			mark(masked, match[0], match[1])
		}
	}
}

func assertSameMask(t *testing.T, text string) {
	t.Helper()
	got, want := make([]bool, len(text)), make([]bool, len(text))
	maskPatterns(text, got)
	referenceMaskPatterns(text, want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mask differs for %q", text)
	}
}

func TestMaskPatternEquivalence(t *testing.T) {
	t.Run("randomized", testMaskPatternRandomizedEquivalence)
	t.Run("embedded word lists", testMaskEmbeddedWordlistEquivalence)
	samples := []string{
		"", "\n", "word\n", "\nword", "plain prose without special marks",
		"HTTPS://example.com/a\nbadword", "<name+tag@example.com>\r\nnext",
		"[abcdef123] 0XFACE123 #ABCDEF sha256-abcdefghijklmnopqrstuvwxyz0123456789",
		"md5:abcdefghijklmnopqrstuvwxyz0123456789 U+ABCD-EF01",
		"01234567-89ab-cdef-0123-456789abcdef \\nword \\xABCD",
		"abcdefg deadbeef abcdef1", "https://example.com/a\u2028b\tend",
		"name\n@example.com U\n+ABCD 0\nxABCDEF [abc\ndef123]",
		"λ\rhttps://example.com/λ\fending", "SHA256-abcdefghijklmnopqrstuvwxyz0123456789",
	}
	for _, sample := range samples {
		assertSameMask(t, sample)
	}
}

func testMaskPatternRandomizedEquivalence(t *testing.T) {
	random := rand.New(rand.NewSource(175))
	tokens := []string{"plain", "\n", "\r\n", " ", "\t", "@", "https://example.com", "[abc123def]", "\\xABCDE", "U+ABCD", "#ABCDEF", "0x0123face", "deadbeef", "λ", "\u2028", "name@example.com", "sha256-abcdefghijklmnopqrstuvwxyz0123456789"}
	for range 500 {
		var text strings.Builder
		for range 20 {
			text.WriteString(tokens[random.Intn(len(tokens))])
		}
		assertSameMask(t, text.String())
	}
}

func testMaskEmbeddedWordlistEquivalence(t *testing.T) {
	for _, dictionary := range embeddedDicts {
		t.Run(dictionary.name, func(t *testing.T) { assertSameMask(t, dictionary.data) })
	}
}

func FuzzMaskPatternEquivalence(f *testing.F) {
	for _, seed := range []string{"hello\nhttps://example.com", "name@example.com", "0Xabcdef123\nU+ABCD"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) { assertSameMask(t, text) })
}

func BenchmarkIgnorePatterns(b *testing.B) {
	for _, test := range []struct {
		name string
		mask func(string, []bool)
	}{
		{"whole-document", referenceMaskPatterns}, {"line-markers", maskPatterns},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.SetBytes(int64(len(dictEnUS)))
			b.ReportAllocs()
			for b.Loop() {
				test.mask(dictEnUS, make([]bool, len(dictEnUS)))
			}
		})
	}
}
