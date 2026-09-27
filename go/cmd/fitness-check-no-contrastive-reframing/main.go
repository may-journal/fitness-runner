// Command fitness-check-no-contrastive-reframing flags the "not X, it's Y"
// reframing in markdown prose: a sentence rejects a claim, then restates the
// real point ("It's not a workout. It's a lifestyle."). It favors precision
// over recall — it fires only when the sentence opens with a demonstrative
// copula (It's, That's, This is) and carries both a negation and a contrast,
// so ordinary technical negations are left alone. It reuses the mdx prose
// masking, so fenced code, headings, tables, and inline code never trip it,
// and it drops quoted spans so documenting the pattern does not self-flag.
package main

import (
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-contrastive-reframing"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errs, n, err := walkfs.ScanFiles(root, []string{".md"}, fileErrors)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errs) > 0 {
		return checkkit.Fail(n, errs...), nil
	}
	return checkkit.Pass(n), nil
}

var (
	// leadRe matches a sentence that opens with a demonstrative copula — the
	// shape the pattern almost always takes. This is the precision lever: a
	// mid-paragraph technical negation rarely opens this way.
	leadRe = regexp.MustCompile(`(?i)^(it|that|this|they|these|those)(['’]s|['’]re| is| are| was| were| [a-z]+n['’]t)\b`)
	// negRe matches a negation anywhere in the sentence: "not" or any n't
	// contraction (matched generically so no word fragment is spelled out).
	negRe = regexp.MustCompile(`(?i)\b(not|[a-z]+n['’]t)\b`)
	// pivotRe matches the single-sentence contrast pivot after a comma or
	// semicolon: "…, but Y", "…; it's Y".
	pivotRe = regexp.MustCompile(`(?i)[,;]\s+(but|rather|it['’]s|it is|that['’]s)\b`)
	// quotedRe and curlyRe drop quoted spans, which may cross a sentence
	// boundary, so an example in quotes is never scanned as prose.
	quotedRe = regexp.MustCompile(`"[^"]*"`)
	curlyRe  = regexp.MustCompile(`“[^”]*”`)
)

// fileErrors returns the reframing errors for one markdown file's prose.
func fileErrors(rel, content string) []string {
	sentences := mdx.Sentences(stripQuoted(mdx.Prose(content)))
	var errs []string
	for i := range sentences {
		if snippet := reframeAt(sentences, i); snippet != "" {
			errs = append(errs, msgFor(rel, snippet))
		}
	}
	return errs
}

// reframeAt returns the offending snippet at sentence i — the single-sentence
// form, or the split form spanning i and i+1 — or "" when neither matches.
func reframeAt(sentences []string, i int) string {
	if isSingle(sentences[i]) {
		return sentences[i]
	}
	if i+1 < len(sentences) && isSplit(sentences[i], sentences[i+1]) {
		return sentences[i] + " " + sentences[i+1]
	}
	return ""
}

// isSingle reports the one-sentence form: a demonstrative opening that both
// negates and pivots — "It's not a workout, but a lifestyle."
func isSingle(s string) bool {
	return leadRe.MatchString(s) && negRe.MatchString(s) && pivotRe.MatchString(s)
}

// isSplit reports the two-sentence form: a demonstrative negation followed by
// a demonstrative assertion — "It's not a workout. It's a lifestyle."
func isSplit(a, b string) bool {
	return leadRe.MatchString(a) && negRe.MatchString(a) &&
		leadRe.MatchString(b) && !negRe.MatchString(b)
}

// stripQuoted blanks straight- and curly-quoted spans so a quoted example is
// not read as prose.
func stripQuoted(s string) string {
	s = quotedRe.ReplaceAllString(s, " ")
	return curlyRe.ReplaceAllString(s, " ")
}

// msgFor builds the failure message, quoting a trimmed snippet.
func msgFor(rel, snippet string) string {
	if len(snippet) > 80 {
		snippet = snippet[:80] + "…"
	}
	return rel + `: rewrite as a direct statement, not the "not X, it's Y" pattern: "` + snippet + `"`
}
