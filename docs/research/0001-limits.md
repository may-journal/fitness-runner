---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Implications and limits for a fitness check

What the evidence supports, in order of defensibility.

## Structural budgets first

Use structural budgets, not readability scores, at fine granularity. Both of this repo's quality interventions were structural gates: `changelog-bullets` (size caps and counts) and the plan template. In both corpora, structure separated good from bad where formulas failed or inverted. Add Hemingway-style per-sentence rules and depth counters, all exact and stdlib.

## Document-level outliers only

If a grade-style score is wanted, use Coleman-Liau or LIX over a whole file. Apply a frozen masking spec first: strip fences, placeholder inline code and URLs, treat block boundaries as sentence boundaries. Refuse to score below a 100-word minimum, reporting insufficient text instead. Use warn-level bands, not hard fails, because the formulas are gameable.

## Cohesion is unexplored high ground

Connective density and adjacent-sentence word overlap are deterministic, cheap, and target what formulas miss. They measure whether sentences connect. No mainstream CI tool ships them.

## A Sonar-style score is buildable

A Sonar-style prose cognitive-complexity score would be novel: an additive increment scheme over nesting, clause load, flow breaks, and referent switches. It should be validated against human judgments on this repo's docs before it gates anything.

## Thresholds must be in-house

Scores are implementation-defined once a tokenizer and splitter are frozen. Literature thresholds like grade 8 and LIX 35 do not transfer to technical prose. Well-written technical prose legitimately runs grade 11-15 here.
