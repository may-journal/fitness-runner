---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../../.fitnessrc.json']
---

<!-- cspell:ignore Liau unbackticked -->

# text-readability

A document-level readability smoke detector for markdown prose. Each `.md` file is scored with the three character-based readability formulas — Coleman-Liau, ARI, and LIX. A file fails only when at least two of the three sit in alarm territory. The bands, the ensemble vote, and the masking rules come from this repo's own corpus in [docs/research/0001-prose-cognitive-complexity.md](../../../docs/research/0001-prose-cognitive-complexity.md). Well-written technical prose here runs grade 11-15 and LIX 40-48. The alarms fire on genuine outliers — a 60-word run-on sentence, notation masquerading as prose — never on ordinary dense technical writing.

Lower scores are not better. Readability formulas reward chatty filler and punish information density. On this repo's plan documents they rank the preferred style as the harder one. This check is a ceiling on absurdity, not a target to optimize.

## Behavior

- Whole documents only, never paragraphs: files with fewer than 100 prose words are counted but not judged (below that the formulas are statistically meaningless).
- Frozen masking spec before scoring: front matter and HTML comments dropped; fenced code, headings, and tables skipped; inline code spans become a placeholder word; link text kept, URLs and version tokens dropped; every block end (heading, list item, blank line) counts as a sentence boundary.
- The three formulas use only letter, word, and sentence counts — no syllable guessing, no word lists — so the check is deterministic. Scores are implementation-defined; the thresholds are calibrated against this implementation, not the literature.
- Alarm requires 2 of 3: Coleman-Liau >= 18, ARI >= 18, LIX >= 60 (defaults).
- `--report` (passthrough arg) prints every file's scores to stderr without changing the verdict.

## Errors

Per-file alarms come first. A failing run then appends three guidance lines: the formulas, the edits that lower them, and where the methodology lives. An LLM (or a human) can fix the prose from the error output alone:

```text
docs/foo.md: readability alarm on 412 prose words — Coleman-Liau 19.2 and ARI 18.6 (grade bands, alarm at 18), LIX 63.0 (alarm at 60)
formulas: Coleman-Liau = 5.88(letters/word) - 29.6(sentences/word) - 15.8; ARI = 4.71(letters/word) + 0.5(words/sentence) - 21.43; LIX = words/sentence + 100(share of 7+ letter words); all three rise with long words and long sentences, and a file fails when 2 of 3 exceed their band
to fix: split sentences over ~25 words (semicolon and em-dash chains count as ONE sentence — end them with periods); wrap identifiers, file names, and check names in backticks (code spans are masked to one short word before scoring, so unbackticked notation reads as very long words); move quoted tool output and name lists into fenced code blocks, which are never scored
methodology: go/cmd/fitness-check-text-readability/README.md (behavior, masking spec, bands) and docs/research/0001-prose-cognitive-complexity.md (why these formulas, calibration on this repo, their limits)
```

## Configure

Optional bands in `.fitnessrc.json` (any positive value overrides its default):

```json
{
  "textReadability": { "maxGrade": 18, "maxLix": 60, "minWords": 100 }
}
```

## Enable

Opt-in — add the name to your `checks` list in `.fitnessrc.json`:

```json
{
  "checks": ["text-readability"]
}
```
