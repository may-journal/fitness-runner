package spell

// cspell:ignore packagejson borwn quik wordlists

import (
	"bufio"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
)

// legacyEmbeddedWords rebuilds the embedded dictionaries the way the engine
// used to at startup — every trimmed, non-comment line into one set. It is
// the reference the binary-search Lookup is differentially tested against.
func legacyEmbeddedWords(t *testing.T) map[string]struct{} {
	t.Helper()
	words := make(map[string]struct{}, 220000)
	for _, ed := range embeddedDicts {
		scanner := bufio.NewScanner(strings.NewReader(ed.data))
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word != "" && !strings.HasPrefix(word, "#") {
				words[word] = struct{}{}
			}
		}
	}
	return words
}

func TestEmbeddedWords(t *testing.T) {
	words := legacyEmbeddedWords(t)
	if len(words) < 150000 {
		t.Fatalf("embedded dictionary suspiciously small: %d words", len(words))
	}
	present := []string{
		"the", "fitness", "typescript", "readme", "changelog", "goroutine",
		"npm", "don't", "prettier-plugin-packagejson",
	}
	for _, w := range present {
		if !embeddedBase.Lookup(w) {
			t.Errorf("expected embedded word %q", w)
		}
	}
	absent := []string{"borwn", "", "# comment"}
	for _, w := range absent {
		if embeddedBase.Lookup(w) {
			t.Errorf("did not expect embedded word %q", w)
		}
	}
}

func TestEmbeddedWordsFeedChecker(t *testing.T) {
	c := NewEmbeddedChecker()
	if issues := c.CheckText("The quick brown fox jumps over the lazy dog."); len(issues) != 0 {
		t.Fatalf("expected clean pangram, got %v", issues)
	}
	issues := c.CheckText("teh quik borwn fox")
	if len(issues) != 2 || issues[0].Word != "quik" || issues[1].Word != "borwn" {
		t.Fatalf("expected quik and borwn flagged, got %v", issues)
	}
}

// TestEmbeddedDictNames pins the embeddedDicts table to the dict/*.txt files
// on disk: a new wordlist added under dict/ without a matching //go:embed
// would otherwise silently vanish from the dictionary.
func TestEmbeddedDictNames(t *testing.T) {
	entries, err := os.ReadDir("dict")
	if err != nil {
		t.Fatalf("reading dict dir: %v", err)
	}
	onDisk := map[string]bool{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".txt") {
			onDisk[e.Name()] = true
		}
	}
	for _, ed := range embeddedDicts {
		if !onDisk[ed.name] {
			t.Errorf("embedded %q has no dict/*.txt on disk", ed.name)
		}
		delete(onDisk, ed.name)
	}
	for name := range onDisk {
		t.Errorf("dict/%s on disk but not embedded", name)
	}
}

// TestEmbeddedDictsSearchable verifies the shape searchLines depends on:
// header comments only before the first word, then non-empty, whitespace-trim
// stable lines in byte-wise sorted order (the committed lists are pre-sorted).
func TestEmbeddedDictsSearchable(t *testing.T) {
	for _, ed := range embeddedDicts {
		t.Run(ed.name, func(t *testing.T) {
			verifySearchable(t, ed.data)
		})
	}
}

func verifySearchable(t *testing.T, data string) {
	t.Helper()
	prev, inBody := "", false
	for i, line := range strings.Split(data, "\n") {
		if !inBody && strings.HasPrefix(line, "#") {
			continue
		}
		inBody = true
		checkBodyLine(t, i, prev, line, strings.HasSuffix(data, "\n"+line))
		prev = line
	}
}

// checkBodyLine asserts one post-header line is searchable: no comment or
// whitespace, non-empty except a final terminator, and not before prev.
func checkBodyLine(t *testing.T, i int, prev, line string, last bool) {
	t.Helper()
	if line == "" && last {
		return // the final "" after a trailing newline
	}
	if line == "" || strings.HasPrefix(line, "#") || line != strings.TrimSpace(line) {
		t.Fatalf("line %d: unsearchable body line %q", i+1, line)
	}
	if line < prev {
		t.Fatalf("line %d: %q sorts before %q — not byte-wise sorted", i+1, line, prev)
	}
}

// TestEmbeddedLookupDifferential sweeps the new binary-search Lookup against
// the legacy map: every embedded word must hit, and near-miss mutations plus
// seeded random strings must agree exactly with map membership.
func TestEmbeddedLookupDifferential(t *testing.T) {
	legacy := legacyEmbeddedWords(t)
	sorted := make([]string, 0, len(legacy))
	for w := range legacy {
		sorted = append(sorted, w)
	}
	sort.Strings(sorted)
	for _, w := range sorted {
		if !embeddedBase.Lookup(w) {
			t.Fatalf("legacy word %q not found by Lookup", w)
		}
	}
	agreeOnProbes(t, legacy, differentialProbes(sorted))
}

// differentialProbes builds the negative-leaning probe set: mutations of a
// deterministic word sample (boundary near-misses) plus 1000 seeded random
// strings.
func differentialProbes(sorted []string) []string {
	var probes []string
	for i := 0; i < len(sorted); i += 97 {
		w := sorted[i]
		probes = append(probes, w+"x", "x"+w, strings.TrimSuffix(w, w[len(w)-1:]), w+"'s")
	}
	rng := rand.New(rand.NewSource(1))
	const alphabet = "abcdefghijklmnopqrstuvwxyz'-."
	for i := 0; i < 1000; i++ {
		var b strings.Builder
		for n := rng.Intn(14) + 1; n > 0; n-- {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}
		probes = append(probes, b.String())
	}
	return probes
}

func agreeOnProbes(t *testing.T, legacy map[string]struct{}, probes []string) {
	t.Helper()
	for _, p := range probes {
		_, inLegacy := legacy[p]
		if got := embeddedBase.Lookup(p); got != inLegacy {
			t.Fatalf("Lookup(%q) = %v, legacy map = %v", p, got, inLegacy)
		}
	}
}
