// cspell:ignore Liau unbackticked scoreable

package check

import (
	"fmt"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Defaults are calibrated to this repo's baseline of grade 11-15 and LIX 40-48, so only genuine outliers fire; conf.Defaults holds them.
var (
	defaultMaxGrade = conf.Defaults.TextReadability.MaxGrade
	defaultMaxLix   = conf.Defaults.TextReadability.MaxLix
	defaultMinWords = conf.Defaults.TextReadability.MinWords
)

// thresholds are the resolved alarm bands for one run.
type thresholds struct {
	maxGrade float64
	maxLix   float64
	minWords int
}

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "text-readability", Scope: checkkit.ScopeFiles, Blob: true},
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
	return walkFiles(root, loadThresholds(root), hasFlag(args, "--report")), nil
}

// walkFiles scores every in-scope markdown file, then each comment block
// the run judges. A file it cannot read is skipped and named as info.
func walkFiles(root string, th thresholds, report bool) checkkit.Result {
	judge := func(rel, content string) []string {
		return judgeContent(rel, content, th, report)
	}
	result := comments.Judge(walkfs.ScanFiles(root, []string{".md"}, judge), root, judge)
	if len(result.Errors) > 0 {
		result.Errors = append(result.Errors, guidance()...)
	}
	return result
}

// guidance follows the per-file alarms once per failing run, saying what each formula measures and which edits lower it.
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

// judgeContent scores already-read markdown under name and returns its alarm, if any, for the file walk and body mode.
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

// formulas computes Coleman-Liau, ARI and LIX from counts, with alphanumeric character counts, so thresholds are calibrated to this implementation.
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
