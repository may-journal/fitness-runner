---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain -->

# Judging the cognitive complexity of prose

Can a fitness check score written paragraphs the way `go-complexity` scores functions? A research survey plus an experiment on this repo's own changelog history. Short answer: yes for coarse, document-level outlier detection with a handful of deterministic metrics; no for precise per-paragraph judgment — and the classical "readability formulas" measure far less than their names suggest.

## Summary

- Every classical readability formula is a two-variable regression on the same two surface proxies: word length (standing in for vocabulary familiarity, via Zipf's law) and sentence length (standing in for syntactic working-memory load). None measures cohesion, ordering, ambiguity, or meaning — scrambling word order inside every sentence changes no score.
- Three formulas are cleanly computable in deterministic stdlib-only Go: Coleman-Liau, ARI, and LIX need only letter, word, and sentence counts. The syllable-based formulas are implementation-defined (no two tools agree), Gunning Fog is not mechanizable as specified, and Dale-Chall requires an embedded word list that saturates on technical vocabulary.
- All eight were normed on samples of 100+ words, with standard errors of 1.5-2.5 grade levels even at that length. On 1-3 sentence paragraphs, one missed sentence boundary swings results by 2-10 grades. Per-paragraph gating is statistically meaningless; document-level outlier detection is defensible.
- The experiment on this repo's changelog (essay bullets at `HEAD~1` vs the tight rewrite at `HEAD`) confirms it: no formula works as a per-bullet gate (best AUC 0.701; a threshold catching 90% of old bullets falsely flags 71-92% of new ones), but section-level aggregation reaches AUC 0.824 (Coleman-Liau), and the only clean separator is the `changelog-bullets` check's own 365-character cap.
- The best-evidenced cognitive measures (per-word surprisal, propositional idea density, dependency length) need a language model, POS tagger, or parser. What survives with zero dependencies: structural budgets (sentence-length tails, clause chaining, nesting depth, nominalization and passive density) and cohesion proxies (connective density, adjacent-sentence word overlap) — the same family Hemingway Editor and Vale actually ship.
- No one has built a true prose analog of SonarSource's cognitive-complexity metric (increments plus nesting multipliers). The design pattern transfers; the field is open.

## The classical formulas

| Formula | Inputs | Stdlib-Go viable? | Notes |
| --- | --- | --- | --- |
| Flesch Reading Ease | words/sentences, syllables/words | heuristic only | `206.835 − 1.015(W/S) − 84.6(Syl/W)`; most syllable-sensitive of all (84.6 coefficient) |
| Flesch-Kincaid Grade | same | heuristic only | `0.39(W/S) + 11.8(Syl/W) − 15.59`; rank-identical to Reading Ease — computing both adds nothing |
| Gunning Fog | words/sentences, complex-word rate | no | "complex word" excludes proper nouns, familiar jargon, and inflected third syllables — needs human judgment; every automated Fog is a house variant |
| SMOG | polysyllable count per 30 sentences | no (scale) | defined on 30-sentence samples; at 2 sentences each polysyllabic word moves the score a full grade |
| Coleman-Liau | letters, words, sentences | yes | `5.88(L/W) − 29.6(S/W) − 15.8`; designed for machine scoring, no syllables |
| ARI | characters, words, sentences | yes | `4.71(C/W) + 0.5(W/S) − 21.43`; digits count, so version strings inflate it |
| Dale-Chall | 3,000-word familiarity list, words/sentences | no (list) | best validity on general prose, useless on technical prose — every technical term is "unfamiliar," so scores pin at the ceiling |
| LIX | words/sentences, share of 7+-letter words | yes | `(W/S) + 100(long/W)`; simplest defensible construct, binary long-word test is robust to tokenization edge cases |

Validation pedigree worth knowing: Flesch-Kincaid was recalibrated in 1975 against comprehension tests on ~531 Navy enlisted personnel reading technical training material — the closest historical analog to developer docs — and still carried a standard error near two grade levels. The critique literature (Duffy and Kabance 1982; Davison and Kantor 1982) showed that rewriting text to optimize these scores did not reliably improve measured comprehension: short words and chopped sentences are not the same thing as clarity. Goodhart's law applies in full — a hard fail threshold on a formula rewards writing "use the CLI to init the cfg" over "use the command-line interface to initialize the configuration."

Three mechanical traps dominate any implementation:

1. Syllable counting. Exact counts need a pronunciation dictionary (~134k entries). Vowel-group heuristics reach 85-95% per-token accuracy but systematically undercount vowel-sparse code identifiers, so every syllable-based score is implementation-defined — Word, textstat, and GNU style all disagree on identical text. Character-based formulas dodge this entirely.
2. Sentence segmentation. Naive `[.!?]` splitting breaks on "e.g.", version numbers, and paths, and markdown headings and list items carry no terminal punctuation, so naive flattening merges them into mega-sentences. One boundary error that doubles average sentence length adds 7.8 grades to Flesch-Kincaid and 10 to ARI. Segmentation policy affects scores more than formula choice does.
3. Short texts. The formulas become ratio estimators with tiny denominators below ~100 words. The literature's answer is simply: do not score short passages; report "insufficient text" instead.

## What cognitive science offers beyond the formulas

- Per-word surprisal (Hale 2001, Levy 2008) — processing effort as prediction error, `−log P(word | context)` — is the best-evidenced difficulty predictor in the field (logarithmic effect confirmed across 11 languages; PNAS 2024). It requires a language model, with an odd twist: small models like GPT-2 fit human reading times better than large ones. The only zero-dependency proxy is word-frequency rarity from an embedded list.
- Propositional idea density (the Kintsch and Keenan result: reading time tracks the number of atomic ideas, not words; famous from Snowdon's Nun Study) is computable at ~0.97 agreement with human raters by CPIDR — but only given a POS tagger. Closed-class counts (prepositions and conjunctions per 10 words) are the exact-computable slice.
- Dependency length and embedding depth (Yngve 1960; Gibson's dependency locality theory) explain why center-embedding and long subject-verb separations hurt: the reader holds unfinished structure in working memory. True parse metrics need a parser; deterministic proxies exist — words before the first verb, subordinator counters, bracket and em-dash nesting depth.
- Cohesion (Coh-Metrix lineage: Graesser, McNamara, Louwerse, and colleagues) is what the formulas miss entirely: two paragraphs with identical word and sentence lengths differ enormously in difficulty depending on whether sentences share referents and signal their relations. This is also the most tractable target: connective density is a pure word list, and referential cohesion is well approximated by adjacent-sentence content-word overlap after stemming — both fully stdlib-implementable.
- The fully-deterministic heuristic family — sentence-length distribution tails (the one 60-word sentence, which means hide), clauses per sentence, parenthetical nesting depth, nominalization suffix density (`-tion`, `-ment`, `-ance`), passives inside subordinate clauses, lexical density, windowed type-token ratio — is individually weak but jointly proxies the constructs above, at full fidelity, with zero dependencies.
- A direct prose analog of SonarSource's cognitive complexity (Campbell's increment-plus-nesting scheme) does not exist in the literature or in tooling. A defensible mapping: nesting increments for subordinate clauses and parenthetical asides, fundamental increments for each clause beyond the first and each passive in a subordinate clause, flow-break increments for adversative-connective chains and topic shifts, referent-switch increments for pronouns with no nearby antecedent. Like Sonar's metric it would be argued rather than derived, and would need validation against human difficulty ratings.

## Prior art in tooling

- Vale (Go, markdown-aware) is the direct precedent: its readability style ships seven document-level metric rules — Flesch-Kincaid > 8, Reading Ease < 70, Fog > 10, SMOG > 10, LIX > 35, ARI > 8, Coleman-Liau > 9 — as editable YAML conditions. Code fences and inline code are excluded by scope before scoring. Alerts are document-level, which makes poor PR annotations.
- Hemingway Editor is exactly: ARI for the grade badge; per-sentence flags only for sentences of 14+ words (per-sentence ARI 10-13 is "hard," 14+ is "very hard"); adverbs as `-ly` minus a whitelist; passive as be-verb plus participle; a phrase-substitution dictionary — each budgeted per 100 words. No parsing, no ML. Its entire mechanism is reproducible in a fitness check.
- proselint, write-good, and alex are regex-plus-word-list detectors (usage, weasel words, insensitive language) with no scoring; textlint is the best framework precedent — a typed markdown AST where rules subscribe to node types and code nodes never reach prose rules.
- The academic instruments (Coh-Metrix, TAALES, TAACO) demonstrate that cohesion and word familiarity outpredict sentence length, but they emit hundreds of unthresholded features from GUI tools with heavyweight NLP models — structurally unusable as CI gates.
- LLM-as-judge is the only approach that measures actual clarity rather than proxies, and it is not deterministic even at temperature zero. Practice that works: pinned model snapshots, rubric anchoring with few-shot band definitions, coarse pass/warn/fail bands rather than numeric scores, and caching verdicts by content hash so unchanged prose can never newly fail. For a runner whose contract is reproducibility, an LLM judge stays advisory or verdict-frozen.

## The experiment: this repo's changelog as a labeled corpus

The 2026-07-18 changelog rewrite created a natural experiment: `HEAD~1` holds 217 bullets across 61 sections including the essay style (single bullets up to 1,514 characters), `HEAD` holds 238 tight bullets (all under 365 characters, by construction of the `changelog-bullets` check). Same facts, two prose styles, only 7 bullets shared. We scored both populations with ten metrics, masking inline code spans, links, and version tokens first (13-15% of characters).

Per-bullet discrimination (AUC = probability a random old bullet scores worse than a random new one):

| Metric | Old mean | New mean | AUC |
| --- | --- | --- | --- |
| Coleman-Liau | 16.4 | 12.2 | 0.701 |
| Flesch Reading Ease | 34.3 | 49.4 | 0.638 |
| ARI | 15.4 | 13.3 | 0.596 |
| Flesch-Kincaid Grade | 13.2 | 11.7 | 0.577 |
| LIX | 48.9 | 45.7 | 0.547 |
| Gunning Fog | 15.1 | 14.6 | 0.522 |
| Words per sentence | 18.9 | 21.4 | 0.453 (inverted) |

Findings, stated plainly:

- No metric works as a per-bullet gate. Setting each metric's threshold to flag 90% of old bullets falsely flags 71-92% of the tight rewrite. Gunning Fog and LIX are coin flips here. Words-per-sentence discriminates in the wrong direction under period-only splitting, because a tight periodless one-liner counts as one long "sentence."
- Aggregation rescues the signal. Scoring each section's bullets as one document lifts Coleman-Liau to AUC 0.824 (old sections mean 17.8, new 12.4 — the new median section sits at the old 10th percentile) and Reading Ease to 0.759. Sample size, not formula choice, was the binding constraint — exactly as the norming literature predicts.
- The only clean separator is trivial: raw length. 15.7% of old bullets exceed 365 characters; zero new ones do (the worst new bullet is 360). A size cap out-discriminates every readability formula on this corpus.
- Masking matters more than it seems. Unmasked, the old/new Flesch-Kincaid gap nearly vanishes (15.4 vs 15.0) because tight bullets are denser in backticked identifiers; masked, the gap triples. Any prose scorer for this repo must strip code spans first or it will punish precision.
- Formulas explode on notation. The single worst bullet by grade level in either corpus was a 149-character old bullet — a comma list of unbackticked hyphenated identifiers scoring Reading Ease −101. The formulas were reading notation density, not prose difficulty.
- Treating semicolons and em-dashes as sentence boundaries (the essay style hides sentences behind semicolons) lifts sentence-based metrics by 0.05-0.09 AUC but does not change any conclusion — the tight style chains clauses with em-dashes almost as often.
- Reference point: the five longest README paragraphs score almost exactly like the new bullets (Reading Ease 44.8 vs 49.4) and clearly easier than the old ones (34.3). The essay bullets were measurably harder than this repo's ordinary prose; the rewrite restored the baseline.

## Implications for a fitness check

What the evidence supports, in order of defensibility:

1. Structural budgets, not readability scores, at fine granularity. The mechanisms that actually separate tight from sloppy at bullet/paragraph scale are the ones `changelog-bullets` already uses — hard size caps and counts — plus Hemingway-style per-sentence rules (flag sentences of 30+ words, length-gated so short sentences are never scored) and depth counters (parenthetical nesting, clause chaining). All exact, all stdlib.
2. Document-level outlier detection with a character-based formula. If a grade-style score is wanted, use Coleman-Liau or LIX over a whole markdown file, after a frozen masking spec (strip fences, placeholder inline code and URLs, treat block boundaries as sentence boundaries), with a 100-word minimum below which the check reports insufficient text rather than a score. Warn-level bands, not hard fails: the formulas are gameable, and optimizing them can make prose worse.
3. Cohesion proxies are the unexplored high ground. Connective density and adjacent-sentence word overlap are deterministic, cheap, and target what formulas miss — whether sentences connect. No mainstream CI tool ships them.
4. A Sonar-style prose cognitive-complexity score is buildable and would be novel — an additive increment scheme over nesting, clause load, flow breaks, and referent switches — but it should be validated against human judgments on this repo's docs before it gates anything.
5. Thresholds must be calibrated in-house. Scores are implementation-defined once a tokenizer and splitter are frozen; literature thresholds (grade 8, LIX 35) do not transfer to technical prose, which legitimately runs grade 11-15 here even when well-written.

## Sources

- Formula primaries: Flesch 1948; Kincaid et al. 1975 (Navy recalibration); Gunning 1952; McLaughlin 1969 (SMOG); Coleman and Liau 1975; Senter and Smith 1967 (ARI); Dale and Chall 1948/1995; Björnsson 1968 and Anderson 1983 (LIX/RIX). Critiques: Duffy and Kabance 1982; Davison and Kantor 1982; DuBay 2004 survey.
- Cognitive measures: Kintsch and Keenan 1973 (propositions); Brown et al. 2008 (CPIDR, Behavior Research Methods); Sirts et al. 2017 (DEPID); Gibson 1998/2000 (dependency locality); Hale 2001 and Levy 2008 (surprisal); Wilcox et al. 2023 (TACL); Shain et al. 2024 (PNAS); Graesser et al. 2004 (Coh-Metrix); Campbell 2018 (Cognitive Complexity, SonarSource whitepaper).
- Tooling: Vale readability style (github.com/errata-ai/readability); Hemingway mechanics (freeCodeCamp deconstruction); textlint TxtAST architecture; textstat; retext-readability (per-sentence ensemble voting).
- Experiment artifacts: scoring script and corpus extracts in the session scratchpad (`readability.py`); corpus is this repo's `CHANGELOG.md` at `dcb8376` (new) and its parent (old).
