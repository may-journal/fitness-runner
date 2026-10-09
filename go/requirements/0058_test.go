package requirements

import (
	"strings"
	"testing"
)

// textReadability runs `fitness-install -- text-readability <args>` on
// happyRepo with files written over it.
func textReadability(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"text-readability"}, args...)...)
}

// textReadabilityPlain is 140 prose words of short, plain sentences.
var textReadabilityPlain = strings.Repeat("The runner builds each check and then runs the full suite on every commit. ", 10)

// textReadabilityDense is 141 long words in one sentence.
var textReadabilityDense = strings.Repeat("extraordinary bureaucratic infrastructure considerations regarding administrative complexity ", 20) + "conclude.\n"

func Test0058_1(t *testing.T) {
	t.Parallel()
	out, code := textReadability(t, map[string]string{"dense.md": textReadabilityDense})
	sees(t, out, code, 1,
		"dense.md: readability alarm on 141 prose words",
		"(grade bands, alarm at 18)", "(alarm at 60)",
		"to fix: split sentences over ~25 words",
		"methodology: go/cmd/fitness-check-text-readability/README.md")
}

func Test0058_2(t *testing.T) {
	t.Parallel()
	out, code := textReadability(t, map[string]string{"plain.md": textReadabilityPlain})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0058_3(t *testing.T) {
	t.Parallel()
	short := strings.Repeat("extraordinary bureaucratic infrastructure considerations ", 20) + "conclude.\n"
	out, code := textReadability(t, map[string]string{"short.md": short})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0058_4(t *testing.T) {
	t.Parallel()
	// Half seven-letter words in 21-word sentences: only the long-word score is high.
	lix := strings.Repeat(strings.Repeat("journey to ", 10)+"end. ", 6)
	out, code := textReadability(t, map[string]string{"lix.md": lix}, "--report")
	sees(t, out, code, 0, "LIX 68.6", "All 1 checks passed")
}

func Test0058_5(t *testing.T) {
	t.Parallel()
	doc := textReadabilityPlain + "\n\n```text\n" + textReadabilityDense + "```\n"
	out, code := textReadability(t, map[string]string{"notation.md": doc})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0058_6(t *testing.T) {
	t.Parallel()
	out, code := textReadability(t, map[string]string{"body.md": textReadabilityDense}, "--body-file", "body.md")
	sees(t, out, code, 1, "(description): readability alarm on 141 prose words")
}

func Test0058_7(t *testing.T) {
	t.Parallel()
	out, code := textReadability(t, map[string]string{
		".fitnessrc.json": `{"textReadability": {"maxGrade": 5, "minWords": 10}}` + "\n",
		"plain.md":        textReadabilityPlain,
	})
	sees(t, out, code, 1, "plain.md: readability alarm on 140 prose words", "(grade bands, alarm at 5)")
}

func Test0058_8(t *testing.T) {
	t.Parallel()
	out, code := textReadability(t, map[string]string{"plain.md": textReadabilityPlain}, "--report")
	sees(t, out, code, 0, "README.md 3 words (below 100 — not judged)", "plain.md 140 words", "All 1 checks passed")
}
