---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Calibration on this repo's plans

`docs/plans/archive/` holds two generations of one document type. Six free-form TypeScript-era plans are goal-and-checklist hybrids. Two later plans follow the may-journals template: numbered title, one-line blockquote, a Goal paragraph, numbered checkbox sections.

The house judgment is on record: the template replaced the free-form style. That labels the corpus; the question is whether metrics agree. We scored paragraphs, blockquotes, and list items separately, masking code spans, links, and version tokens. Each markdown block end was a sentence boundary.

| Plan | Style | Words | FKGL | Coleman-Liau | ARI | LIX |
| --- | --- | --- | --- | --- | --- | --- |
| `01-go-rewrite.md` | template | 726 | 11.8 | 15.4 | 12.4 | 44.5 |
| `02-npm-free.md` | template | 373 | 10.5 | 13.1 | 11.3 | 42.8 |
| `plan-checks-abstractions.md` | free-form | 796 | 8.6 | 12.4 | 9.6 | 37.1 |
| `plan-deps-vs-devdeps-check.md` | free-form | 200 | 9.9 | 12.6 | 9.0 | 40.1 |
| `plan-issue-13-bundle-size-performance.md` | free-form | 151 | 6.4 | 10.8 | 6.8 | 31.0 |
| `plan-issue-23-config-local-check-paths.md` | free-form | 193 | 6.3 | 9.3 | 6.1 | 30.4 |
| `plan-split-runner-check-packages.md` | free-form | 470 | 6.6 | 10.7 | 6.7 | 33.8 |
| `plan-swiftlint-jscpd-checks.md` | free-form | 400 | 9.8 | 12.7 | 10.2 | 38.0 |

## Findings

The formulas invert that judgment. Every one scores the template plans harder than the free-form plans they replaced. That holds at document and unit level (list-item Coleman-Liau mean 15.6 versus 9.9). A gate built on them would have defended the rejected style.

Genuine paragraphs score sanely. The two Goal paragraphs are real 72-100 word prose. They land at grade 11.6-13.5, where good technical prose belongs.

Per-paragraph scores are noise. Inside `plan-checks-abstractions.md` (26 paragraphs, one register) Flesch-Kincaid swings from 2.3 to 19.7 — a 17-grade spread in uniform text.

## The markdown artifact

The markdown artifact dwarfs everything. Without the block-boundary rule, `01-go-rewrite.md` jumps from grade 11.8 to 46.7, ~104 words per "sentence". Its 45 punctuation-free checkbox items merge into pseudo-sentences. Segmentation policy is worth 35 grades; no quality difference is worth 5.

What separates the generations is structure, not sentence statistics: one blockquote, one Goal paragraph, numbered checkbox sections. That shape is deterministically checkable by a template gate, precisely what readability formulas cannot see.

## The earlier changelog round

An earlier round on the changelog rewrite reached the same conclusions. Per-unit gating failed (AUC 0.701); aggregation helped (0.824 at section level). Masking code spans was essential, and formulas exploded on notation-dense text (Reading Ease −101 on one bullet).

But that corpus was weaker: bullets are notation, not paragraphs, and its clean separator, the character cap, held by construction. The plans corpus supersedes it; the changelog numbers survive as corroboration.
