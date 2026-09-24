---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Prior art in tooling

## Vale

Vale (Go, markdown-aware) is the direct precedent. Its readability style ships seven document-level metric rules as editable YAML conditions. The thresholds are Flesch-Kincaid > 8, Reading Ease < 70, Fog > 10, SMOG > 10, LIX > 35, ARI > 8, and Coleman-Liau > 9. Code fences and inline code are excluded by scope before scoring. Alerts are document-level, which makes poor PR annotations.

## Hemingway Editor

Hemingway is exactly this: ARI for the grade badge, plus per-sentence flags only for sentences of 14+ words. Adverbs are `-ly` minus a whitelist. Passive is a be-verb plus participle. Complex phrases come from a substitution dictionary. Each detector is budgeted per 100 words, with no parsing and no ML.

Its entire mechanism is reproducible in a fitness check.

## Regex and word-list linters

proselint, write-good, and alex are regex-plus-word-list detectors with no scoring. They cover usage, weasel words, and insensitive language. textlint is the best framework precedent: a typed markdown AST where rules subscribe to node types, so code nodes never reach prose rules.

## Academic instruments and LLM judges

The academic instruments (Coh-Metrix, TAALES, TAACO) demonstrate that cohesion and word familiarity outpredict sentence length. But they emit hundreds of unthresholded features from heavyweight NLP models. As CI gates they are structurally unusable.

LLM-as-judge is the only approach that measures actual clarity rather than proxies. It is not deterministic even at temperature zero. What works: pinned model snapshots, rubric anchoring with few-shot bands, coarse pass/warn/fail verdicts, and verdicts cached by content hash. For a runner whose contract is reproducibility, an LLM judge stays advisory or verdict-frozen.
