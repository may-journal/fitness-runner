// Command fitness-check-text-readability is a document-level readability
// smoke detector for markdown prose. It scores each .md file with the three
// character-based readability formulas — Coleman-Liau, ARI, and LIX — and
// fails only when at least two of the three sit in alarm territory
// (grade >= 18 for Coleman-Liau/ARI, LIX >= 60 by default). Files with
// fewer than 100 prose words are never judged: below that, the formulas
// are noise (see docs/research/0001-prose-cognitive-complexity.md, which
// derived the bands, the ensemble vote, and the masking spec from this
// repo's own corpus). Pass --report to print every file's scores to stderr.
//
// cspell:ignore Liau unbackticked scoreable
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Defaults calibrated to this repo's baseline: well-written technical prose
// runs grade 11-15 and LIX 40-48, so the alarm bands sit far above the norm
// and only genuine outliers fire. Lower scores are NOT better — these are
// ceilings on absurdity, not targets to optimize.
const (
	defaultMaxGrade = 18.0
	defaultMaxLix   = 60.0
	defaultMinWords = 100
)

// thresholds are the resolved alarm bands for one run.
type thresholds struct {
	maxGrade float64
	maxLix   float64
	minWords int
}

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "text-readability"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(r, content string) []string {
		return judgeContent("(description)", content, loadThresholds(r), false)
	}); err != nil {
		return checkkit.Result{}, err
	} else if handled {
		return res, nil
	}
	th := loadThresholds(root)
	report := hasFlag(args, "--report")
	files := walkfs.FilesByExt(root, ".md")
	var errs []string
	for _, rel := range files {
		errs = append(errs, judgeFile(root, rel, th, report)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(files), append(errs, guidance()...)...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// guidance follows the per-file alarms once per failing run: what each
// formula measures, which edits lower it, and where the methodology lives —
// structured so an LLM (or human) can fix the prose without external context.
func guidance() []string {
	return []string{
		"formulas: Coleman-Liau = 5.88(letters/word) - 29.6(sentences/word) - 15.8; ARI = 4.71(letters/word) + 0.5(words/sentence) - 21.43; LIX = words/sentence + 100(share of 7+ letter words); all three rise with long words and long sentences, and a file fails when 2 of 3 exceed their band",
		"to fix: split sentences over ~25 words (semicolon and em-dash chains count as ONE sentence — end them with periods); wrap identifiers, file names, and check names in backticks (code spans are masked to one short word before scoring, so unbackticked notation reads as very long words); move quoted tool output and name lists into fenced code blocks, which are never scored",
		"methodology: go/cmd/fitness-check-text-readability/README.md (behavior, masking spec, bands) and docs/research/0001-prose-cognitive-complexity.md (why these formulas, calibration on this repo, their limits)",
	}
}

// loadThresholds resolves the alarm bands: .fitnessrc.json textReadability
// values when positive, else the calibrated defaults.
func loadThresholds(root string) thresholds {
	th := thresholds{maxGrade: defaultMaxGrade, maxLix: defaultMaxLix, minWords: defaultMinWords}
	cfg, err := conf.Load(root)
	if err != nil || cfg == nil {
		return th
	}
	applyConfig(&th, cfg)
	return th
}

// applyConfig overlays every positive config value onto the defaults.
func applyConfig(th *thresholds, cfg *conf.Config) {
	if cfg.TextReadability.MaxGrade > 0 {
		th.maxGrade = cfg.TextReadability.MaxGrade
	}
	if cfg.TextReadability.MaxLix > 0 {
		th.maxLix = cfg.TextReadability.MaxLix
	}
	if cfg.TextReadability.MinWords > 0 {
		th.minWords = cfg.TextReadability.MinWords
	}
}

// hasFlag reports whether the passthrough args contain flag.
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// judgeFile scores one markdown file and returns its alarm, if any.
func judgeFile(root, rel string, th thresholds, report bool) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", rel, err)}
	}
	return judgeContent(rel, string(raw), th, report)
}

// judgeContent scores already-read markdown content under name and returns its
// alarm, if any — the read-free core shared by the file walk and body mode.
func judgeContent(name, content string, th thresholds, report bool) []string {
	c := measure(mdx.Prose(content))
	if report {
		printScore(name, c, th)
	}
	return verdict(name, c, th)
}

// verdict applies the 2-of-3 ensemble vote to one file's counts.
func verdict(rel string, c counts, th thresholds) []string {
	if c.words < th.minWords {
		return nil
	}
	cli, ari, lix := formulas(c)
	if overCount(cli, ari, lix, th) < 2 {
		return nil
	}
	return []string{fmt.Sprintf(
		"%s: readability alarm on %d prose words — Coleman-Liau %.1f and ARI %.1f (grade bands, alarm at %.0f), LIX %.1f (alarm at %.0f)",
		rel, c.words, cli, ari, th.maxGrade, lix, th.maxLix)}
}

// overCount counts how many of the three formulas sit in alarm territory.
func overCount(cli, ari, lix float64, th thresholds) int {
	n := 0
	if cli >= th.maxGrade {
		n++
	}
	if ari >= th.maxGrade {
		n++
	}
	if lix >= th.maxLix {
		n++
	}
	return n
}

// printScore writes one file's scores to stderr for --report mode.
func printScore(rel string, c counts, th thresholds) {
	if c.words < th.minWords {
		fmt.Fprintf(os.Stderr, "%-64s %5d words (below %d — not judged)\n", rel, c.words, th.minWords)
		return
	}
	cli, ari, lix := formulas(c)
	fmt.Fprintf(os.Stderr, "%-64s %5d words  Coleman-Liau %5.1f  ARI %5.1f  LIX %5.1f\n",
		rel, c.words, cli, ari, lix)
}

// counts are the frozen-spec text statistics one file reduces to.
type counts struct {
	words     int
	letters   int
	longWords int
	sentences int
}

// formulas computes Coleman-Liau, ARI, and LIX from counts. Both grade
// formulas use alphanumeric character counts (the house variant — scores
// are implementation-defined, so thresholds are calibrated against this
// implementation, never copied from literature).
func formulas(c counts) (cli, ari, lix float64) {
	w, s, l := float64(c.words), float64(c.sentences), float64(c.letters)
	cli = 0.0588*(l/w*100) - 0.296*(s/w*100) - 15.8
	ari = 4.71*(l/w) + 0.5*(w/s) - 21.43
	lix = w/s + 100*float64(c.longWords)/w
	return cli, ari, lix
}

// measure reduces masked prose to the counts the formulas need.
func measure(text string) counts {
	var c counts
	for _, tok := range strings.Fields(text) {
		n := mdx.AlnumLen(tok)
		if n == 0 {
			continue
		}
		c.words++
		c.letters += n
		if n > 6 {
			c.longWords++
		}
	}
	c.sentences = sentenceCount(text)
	return c
}

// sentenceCount counts sentences via the shared splitter; a text with no
// boundary still counts as one sentence.
func sentenceCount(text string) int {
	if n := len(mdx.Sentences(text)); n > 0 {
		return n
	}
	return 1
}
