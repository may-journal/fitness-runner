// Command fitness-check-prose-budget is a hard-cap brevity linter for markdown
// prose. Per .md file — with front matter, fenced code, tables, and headings
// masked out, the same masking text-readability scores — it enforces six
// limits: words per sentence, sentences per paragraph, paragraphs per section,
// words per list item, items per list, and prose words per section. Each
// limit falls back to a default and is overridable in .fitnessrc.json.
// Every tracked .md file is judged, CHANGELOG.md included.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// limits are the six hard caps for one run.
type limits struct {
	sentenceWords      int
	paragraphSentences int
	sectionParagraphs  int
	listItemWords      int
	listItems          int
	words              int
}

// defaults are calibrated for tight technical docs: short sentences, small
// paragraphs and sections, and bounded lists — held about a quarter tighter
// than the check's first cut.
var defaults = limits{
	sentenceWords:      23,
	paragraphSentences: 4,
	sectionParagraphs:  3,
	listItemWords:      23,
	listItems:          8,
	words:              300,
}

// descriptionName labels the pseudo-file in body-mode errors, where the
// document is an Issue or PR description rather than a file on disk.
const descriptionName = "(description)"

var (
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	listItemRe    = regexp.MustCompile(`^(?:[-*+]|\d+\.)\s+(.*)$`)
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "prose-budget"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	lim := settings(root)
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return check(descriptionName, content, lim)
	}); handled || err != nil {
		return res, err
	}
	return walkFiles(root, lim), nil
}

// walkFiles applies the budget to every in-scope .md file under root.
func walkFiles(root string, lim limits) checkkit.Result {
	files := walkfs.InScope(walkfs.FilesByExt(root, ".md"))
	var errs []string
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, err))
			continue
		}
		errs = append(errs, check(rel, string(raw), lim)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...)
	}
	return checkkit.Pass(len(files))
}

// settings resolves the limits: defaults with every positive config value
// overlaid.
func settings(root string) limits {
	lim := defaults
	cfg, err := conf.Load(root)
	if err != nil || cfg == nil {
		return lim
	}
	applyLimits(&lim, cfg)
	return lim
}

// applyLimits overlays every positive proseBudget config value onto lim.
func applyLimits(lim *limits, cfg *conf.Config) {
	over(&lim.sentenceWords, cfg.ProseBudget.MaxSentenceWords)
	over(&lim.paragraphSentences, cfg.ProseBudget.MaxParagraphSentences)
	over(&lim.sectionParagraphs, cfg.ProseBudget.MaxSectionParagraphs)
	over(&lim.listItemWords, cfg.ProseBudget.MaxListItemWords)
	over(&lim.listItems, cfg.ProseBudget.MaxListItems)
	over(&lim.words, cfg.ProseBudget.MaxWords)
}

func over(dst *int, v int) {
	if v > 0 {
		*dst = v
	}
}

// check returns every limit violation in one file, in a stable order:
// section, then paragraph, then list violations.
func check(rel, content string, lim limits) []string {
	blocks := parse(content)
	errs := sectionErrors(rel, blocks, lim)
	errs = append(errs, paragraphErrors(rel, blocks, lim)...)
	errs = append(errs, listErrors(rel, blocks, lim)...)
	return errs
}

// sectionErrors reports each section over the word budget or the
// paragraph limit, in document order.
func sectionErrors(rel string, blocks []block, lim limits) []string {
	var errs []string
	for _, s := range countSections(blocks) {
		if s.words > lim.words {
			errs = append(errs, fmt.Sprintf(
				"%s: section %q has %d prose words (max %d)", rel, s.name, s.words, lim.words))
		}
		if s.paras > lim.sectionParagraphs {
			errs = append(errs, fmt.Sprintf(
				"%s: section %q has %d paragraphs (max %d)", rel, s.name, s.paras, lim.sectionParagraphs))
		}
	}
	return errs
}

// section is one heading-delimited section's tally: prose words across its
// paragraphs and lists, and its paragraph count. Prose before the first
// heading forms a section named "".
type section struct {
	name  string
	words int
	paras int
}

// countSections tallies words and paragraphs per section, in document order.
func countSections(blocks []block) []section {
	var secs []section
	idx := map[int]int{}
	for _, bl := range blocks {
		i, ok := idx[bl.section]
		if !ok {
			i = len(secs)
			idx[bl.section] = i
			secs = append(secs, section{name: bl.sectionName})
		}
		secs[i].words += bl.words()
		if !bl.isList {
			secs[i].paras++
		}
	}
	return secs
}

// paragraphErrors reports sentence-count and sentence-length violations.
func paragraphErrors(rel string, blocks []block, lim limits) []string {
	var errs []string
	for _, bl := range blocks {
		if bl.isList {
			continue
		}
		errs = append(errs, paragraphError(rel, bl, lim)...)
	}
	return errs
}

// paragraphError reports one paragraph's violations: too many sentences, or a
// sentence over the word limit.
func paragraphError(rel string, bl block, lim limits) []string {
	sentences := mdx.Sentences(mdx.MaskInline(bl.text))
	var errs []string
	if len(sentences) > lim.paragraphSentences {
		errs = append(errs, fmt.Sprintf("%s: a paragraph has %d sentences (max %d): %q",
			rel, len(sentences), lim.paragraphSentences, snippet(bl.text)))
	}
	for _, s := range sentences {
		if n := mdx.WordCount(s); n > lim.sentenceWords {
			errs = append(errs, fmt.Sprintf("%s: a sentence has %d words (max %d): %q",
				rel, n, lim.sentenceWords, snippet(s)))
		}
	}
	return errs
}

// listErrors reports item-count and item-length violations.
func listErrors(rel string, blocks []block, lim limits) []string {
	var errs []string
	for _, bl := range blocks {
		if !bl.isList {
			continue
		}
		errs = append(errs, listError(rel, bl, lim)...)
	}
	return errs
}

// listError reports one list's violations: too many items, or an item over
// the word limit.
func listError(rel string, bl block, lim limits) []string {
	var errs []string
	errs = append(errs, nestingErrors(rel, bl, lim.listItems)...)
	for _, it := range bl.items {
		if n := mdx.WordCount(mdx.MaskInline(it)); n > lim.listItemWords {
			errs = append(errs, fmt.Sprintf("%s: a list item has %d words (max %d): %q",
				rel, n, lim.listItemWords, snippet(it)))
		}
	}
	return errs
}

// snippet collapses whitespace and truncates to a readable preview.
func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 60
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// nestingErrors caps each level of a list at half the level above, rounding
// down from the top-level cap: with 8 top-level items, each item holds at
// most 4 children, each child at most 2, and each of those 1. A level whose
// cap reaches 0 allows no items.
func nestingErrors(rel string, bl block, max int) []string {
	var errs []string
	counts := map[int]int{}
	for i, d := range bl.depths {
		resetDeeper(counts, d)
		counts[d]++
		if counts[d] == (max>>d)+1 {
			errs = append(errs, fmt.Sprintf("%s: a list has more than %d items at level %d: %q", rel, max>>d, d+1, snippet(bl.items[i])))
		}
	}
	return errs
}

// resetDeeper resets the sibling counts of every level deeper than d, since an
// item at level d starts a new group of children.
func resetDeeper(counts map[int]int, d int) {
	for k := range counts {
		if k > d {
			delete(counts, k)
		}
	}
}
