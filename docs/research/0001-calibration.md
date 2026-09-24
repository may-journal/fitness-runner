---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Calibration on this repo's plans

`docs/plans/archive/` holds two generations of the same document type. Six free-form plans from the TypeScript era are goal-and-checklist hybrids in varied shapes. The two later plans follow the may-journals template: numbered title, one-line blockquote, a Goal paragraph, numbered checkbox sections.

The house judgment is on record, since the template replaced the free-form style. That makes the corpus labeled, and the question becomes whether the metrics agree with the judgment. We scored prose paragraphs, blockquotes, and list items separately, with code spans, links, and version tokens masked, treating each markdown block end as a sentence boundary.

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

The formulas invert the house judgment. Every formula scores the template plans as harder on average than the free-form plans they replaced. That holds at document level and at unit level, with list-item Coleman-Liau mean 15.6 template versus 9.9 free-form. A gate built on them would have defended the rejected style.

Genuine paragraphs score sanely. The two Goal paragraphs are real 72-100 word prose, the formulas' native sample size. They land at grade 11.6-13.5, exactly where well-written technical prose belongs.

Per-paragraph scores are noise. `plan-checks-abstractions.md` has 26 paragraphs in one register. Flesch-Kincaid swings from 2.3 to 19.7 inside it, a 17-grade spread in a uniformly written document.

## The markdown artifact

The markdown artifact dwarfs everything. Scored without the block-boundary rule, `01-go-rewrite.md` jumps from grade 11.8 to 46.7, roughly 104 words per "sentence". Its 45 punctuation-free checkbox items merge into pseudo-sentences. Segmentation policy is worth 35 grades; no real quality difference here is worth 5.

What separates the generations is structure, not sentence statistics: one blockquote, one Goal paragraph, numbered checkbox sections. That shape is deterministically checkable by a template gate. It is precisely the thing readability formulas cannot see.

## The earlier changelog round

An earlier round used the changelog rewrite as its corpus, and reached the same negative conclusions. Per-unit gating failed (best AUC 0.701). Aggregation helped (0.824 at section level). Masking code spans was essential, and formulas exploded on notation-dense text (Reading Ease −101 on one bullet).

But that corpus was weaker: changelog bullets are notation rather than paragraphs, and its one clean separator, the character cap, was true by construction. The plans corpus supersedes it. The changelog numbers survive only as corroboration.
