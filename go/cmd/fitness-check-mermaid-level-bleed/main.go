// Command fitness-check-mermaid-level-bleed warns when a callout description
// at one C4 level repeats the previous level verbatim — lower levels should
// add level-specific rationale, not restate the parent. Compares only exact
// (normalized) matches to stay low-false-positive. The Go port of the
// mermaid-level-bleed check: unlike its four siblings it does not use the
// RunDocCheck driver — filesChecked counts the numbered `architecture/NN*.md`
// level files, whether or not they contain mermaid blocks.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-level-bleed"},
		Run:      run,
	})
}

// levelFile matches a numbered C4 level file, e.g. `architecture/02-containers.md`.
var levelFile = regexp.MustCompile(`(?:^|/)architecture/(\d+)[^/]*\.md$`)

// descriptionHeader matches a table header cell naming the description column.
var descriptionHeader = regexp.MustCompile(`(?i)^description$`)

// jsWS is the JavaScript \s character class as an RE2 class body — the
// residue of internal/mermaid's unexported twin, needed here because the TS
// check normalizes descriptions with .trim() and .replace(/\s+/g, ' ').
const jsWS = `\t\n\v\f\r \x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`

// jsSpaceRun matches one or more JavaScript whitespace characters.
var jsSpaceRun = regexp.MustCompile(`[` + jsWS + `]+`)

// isJSSpace reports whether r is in the JavaScript \s class.
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// normalizeDescription lowercases and whitespace-collapses one cell — the
// port of `.trim().toLowerCase().replace(/\s+/g, ' ')`.
func normalizeDescription(cell string) string {
	text := strings.ToLower(strings.TrimFunc(cell, isJSSpace))
	return jsSpaceRun.ReplaceAllString(text, " ")
}

// descriptionSet is a JS Set of strings: membership plus insertion order.
type descriptionSet struct {
	items []string
	seen  map[string]bool
}

func newDescriptionSet() *descriptionSet {
	return &descriptionSet{seen: map[string]bool{}}
}

func (s *descriptionSet) add(text string) {
	if s.seen[text] {
		return
	}
	s.seen[text] = true
	s.items = append(s.items, text)
}

func (s *descriptionSet) has(text string) bool { return s.seen[text] }

// addTableDescriptions adds normalized (lowercased, whitespace-collapsed)
// descriptions from one callout table.
func addTableDescriptions(table *mermaid.CalloutTableBlock, set *descriptionSet) {
	index := descriptionColumn(table.Header)
	for _, row := range table.Rows {
		cell := ""
		if index < len(row) {
			cell = row[index]
		}
		if text := normalizeDescription(cell); text != "" {
			set.add(text)
		}
	}
}

// descriptionColumn returns the index of the first header cell naming the
// description column, defaulting to 1 (the TS fallback) when none matches.
func descriptionColumn(header []string) int {
	for i, cell := range header {
		if descriptionHeader.MatchString(cell) {
			return i
		}
	}
	return 1
}

// levelDescriptions collects a doc's normalized callout descriptions, for
// exact cross-level comparison.
func levelDescriptions(content string) *descriptionSet {
	set := newDescriptionSet()
	for _, block := range mermaid.ParseDoc(content) {
		if table, ok := block.(*mermaid.CalloutTableBlock); ok {
			addTableDescriptions(table, set)
		}
	}
	return set
}

// level is one numbered C4 level file and its callout descriptions.
type level struct {
	descriptions *descriptionSet
	file         string
	level        int
}

// collectLevels reads the numbered C4 level files under root, ordered by
// level (ties keep walk order, like the JS stable sort).
func collectLevels(root string) ([]level, error) {
	var levels []level
	for _, file := range walkfs.FilesByExt(root, ".md") {
		m := levelFile.FindStringSubmatch(file)
		if m == nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return nil, err
		}
		n, _ := strconv.Atoi(m[1])
		levels = append(levels, level{
			descriptions: levelDescriptions(string(raw)),
			file:         file,
			level:        n,
		})
	}
	sort.SliceStable(levels, func(i, j int) bool { return levels[i].level < levels[j].level })
	return levels, nil
}

// bleedErrors reports where a level repeats a description from the
// immediately-previous level verbatim.
func bleedErrors(levels []level) []string {
	var errors []string
	for i := 1; i < len(levels); i++ {
		previous := levels[i-1]
		for _, text := range levels[i].descriptions.items {
			if previous.descriptions.has(text) {
				errors = append(errors, fmt.Sprintf(
					`%s: description "%s" repeats level %d verbatim — lower levels should add level-specific rationale`,
					levels[i].file, text, previous.level))
			}
		}
	}
	return errors
}

func run(root string, _ []string) (checkkit.Result, error) {
	levels, err := collectLevels(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	errors := bleedErrors(levels)
	if len(errors) > 0 {
		return checkkit.Fail(len(levels), errors...), nil
	}
	return checkkit.Pass(len(levels)), nil
}
