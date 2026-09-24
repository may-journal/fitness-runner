// Command fitness-check-prose-budget is a hard-cap brevity linter for markdown
// prose. Per .md file — with front matter, fenced code, tables, and headings
// masked out, the same masking text-readability scores — it enforces six
// limits: words per sentence, sentences per paragraph, paragraphs per section,
// words per list item, items per list, and total prose words per file. Each
// limit falls back to a default and is overridable in .fitnessrc.json.
//
// CHANGELOG.md is always exempt; configured exempt paths (an exact path or a
// `dir/**` prefix) union with it.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
// paragraphs and sections, and bounded lists.
var defaults = limits{
	sentenceWords:      30,
	paragraphSentences: 5,
	sectionParagraphs:  4,
	listItemWords:      30,
	listItems:          10,
	words:              400,
}

// builtinExempt is exempt regardless of config, because it grows by design.
const builtinExempt = "CHANGELOG.md"

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

func run(root string, _ []string) (checkkit.Result, error) {
	lim, exempt := settings(root)
	files := walkfs.FilesByExt(root, ".md")
	var errs []string
	for _, rel := range files {
		if isExempt(rel, exempt) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, err))
			continue
		}
		errs = append(errs, check(rel, string(raw), lim)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// settings resolves the limits and exempt set: defaults with every positive
// config value overlaid, and the built-in exemption combined with configured
// paths.
func settings(root string) (limits, []string) {
	lim := defaults
	exempt := []string{builtinExempt}
	cfg, err := conf.Load(root)
	if err != nil || cfg == nil {
		return lim, exempt
	}
	applyLimits(&lim, cfg)
	return lim, append(exempt, cfg.ProseBudget.Exempt...)
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

// isExempt reports whether rel matches an exempt entry: an exact path, or a
// `dir/**` prefix matching any file under dir.
func isExempt(rel string, exempt []string) bool {
	for _, e := range exempt {
		if rel == e {
			return true
		}
		if dir, ok := strings.CutSuffix(e, "/**"); ok && strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

// check returns every limit violation in one file, in a stable order: the
// file-word fallback, then section, paragraph, and list violations.
func check(rel, content string, lim limits) []string {
	blocks := parse(content)
	var errs []string
	if msg := wordsError(rel, blocks, lim); msg != "" {
		errs = append(errs, msg)
	}
	errs = append(errs, sectionErrors(rel, blocks, lim)...)
	errs = append(errs, paragraphErrors(rel, blocks, lim)...)
	errs = append(errs, listErrors(rel, blocks, lim)...)
	return errs
}

// wordsError reports the total-prose-words fallback violation, if any.
func wordsError(rel string, blocks []block, lim limits) string {
	if n := totalWords(blocks); n >= lim.words {
		return fmt.Sprintf("%s: %d prose words over the %d-word budget", rel, n, lim.words)
	}
	return ""
}

// totalWords sums the prose words across every block.
func totalWords(blocks []block) int {
	n := 0
	for _, bl := range blocks {
		n += bl.words()
	}
	return n
}

// sectionErrors reports each section whose paragraph count exceeds the limit.
func sectionErrors(rel string, blocks []block, lim limits) []string {
	var errs []string
	for _, s := range countSections(blocks) {
		if s.paras > lim.sectionParagraphs {
			errs = append(errs, fmt.Sprintf(
				"%s: section %q has %d paragraphs (max %d)", rel, s.name, s.paras, lim.sectionParagraphs))
		}
	}
	return errs
}

// section is one heading-delimited section's paragraph tally.
type section struct {
	name  string
	paras int
}

// countSections tallies paragraph blocks per section, in document order.
func countSections(blocks []block) []section {
	var secs []section
	idx := map[int]int{}
	for _, bl := range blocks {
		if bl.isList {
			continue
		}
		i, ok := idx[bl.section]
		if !ok {
			idx[bl.section] = len(secs)
			secs = append(secs, section{name: bl.sectionName})
			i = len(secs) - 1
		}
		secs[i].paras++
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
	if len(bl.items) > lim.listItems {
		errs = append(errs, fmt.Sprintf("%s: a list has %d items (max %d)", rel, len(bl.items), lim.listItems))
	}
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
