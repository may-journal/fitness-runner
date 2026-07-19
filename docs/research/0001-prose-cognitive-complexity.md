---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Judging the cognitive complexity of prose

Can a fitness check score written paragraphs the way `go-complexity` scores functions? A research survey plus an experiment on this repo's own plan documents. Short answer: yes for coarse, document-level outlier detection with a handful of deterministic metrics. No for precise per-paragraph judgment. And the classical "readability formulas" measure far less than their names suggest.

## Summary

- Every classical readability formula is a two-variable regression on the same two surface proxies. Word length stands in for vocabulary familiarity (via Zipf's law). Sentence length stands in for syntactic working-memory load. None measures cohesion, ordering, ambiguity, or meaning. Scrambling word order inside every sentence changes no score.
- Three formulas are cleanly computable in deterministic stdlib-only Go. Coleman-Liau, ARI, and LIX need only letter, word, and sentence counts. The syllable-based formulas are implementation-defined (no two tools agree). Gunning Fog is not mechanizable as specified. Dale-Chall requires an embedded word list that saturates on technical vocabulary.
- All eight were normed on samples of 100+ words, with standard errors of 1.5-2.5 grade levels even at that length. On 1-3 sentence paragraphs, one missed sentence boundary swings results by 2-10 grades. Per-paragraph gating is statistically meaningless. Document-level outlier detection is defensible.
- The experiment on this repo's archived plans (the two may-journals-template plans vs the six free-form plans the template replaced) is worse than inconclusive for the formulas. It inverts them. Every formula scores the preferred template style as harder on average than the rejected free-form style. Paragraphs within one uniformly written plan swing 17 grade levels. Dropping the markdown block-boundary rule inflates one plan from grade 11.8 to 46.7. What separates the generations is checkable structure — the template itself. The formulas cannot see it, and they actively penalize it.
- The best-evidenced cognitive measures (per-word surprisal, propositional idea density, dependency length) need a language model, POS tagger, or parser. What survives with zero dependencies: structural budgets and cohesion proxies. Structural budgets mean sentence-length tails, clause chaining, nesting depth, nominalization and passive density. Cohesion proxies mean connective density and adjacent-sentence word overlap. This is the same family Hemingway Editor and Vale actually ship.
- No one has built a true prose analog of SonarSource's cognitive-complexity metric (increments plus nesting multipliers). The design pattern transfers. The field is open.

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

Validation pedigree worth knowing: Flesch-Kincaid was recalibrated in 1975 against comprehension tests on ~531 Navy enlisted personnel reading technical training material. That is the closest historical analog to developer docs. Even there it carried a standard error near two grade levels. The critique literature (Duffy and Kabance 1982; Davison and Kantor 1982) showed that rewriting text to optimize these scores did not reliably improve measured comprehension. Short words and chopped sentences are not the same thing as clarity. Goodhart's law applies in full. A hard fail threshold on a formula rewards writing "use the CLI to init the cfg" over "use the command-line interface to initialize the configuration."

Three mechanical traps dominate any implementation:

1. Syllable counting. Exact counts need a pronunciation dictionary (~134k entries). Vowel-group heuristics reach 85-95% per-token accuracy but systematically undercount vowel-sparse code identifiers. Every syllable-based score is therefore implementation-defined. Word, textstat, and GNU style all disagree on identical text. Character-based formulas dodge this entirely.
2. Sentence segmentation. Naive `[.!?]` splitting breaks on "e.g.", version numbers, and paths. Markdown headings and list items carry no terminal punctuation, so naive flattening merges them into mega-sentences. One boundary error that doubles average sentence length adds 7.8 grades to Flesch-Kincaid and 10 to ARI. Segmentation policy affects scores more than formula choice does.
3. Short texts. The formulas become ratio estimators with tiny denominators below ~100 words. The literature's answer is simply: do not score short passages. Report "insufficient text" instead.

## What cognitive science offers beyond the formulas

- Per-word surprisal (Hale 2001, Levy 2008) treats processing effort as prediction error: `−log P(word | context)`. It is the best-evidenced difficulty predictor in the field (logarithmic effect confirmed across 11 languages; PNAS 2024). It requires a language model, with an odd twist: small models like GPT-2 fit human reading times better than large ones. The only zero-dependency proxy is word-frequency rarity from an embedded list.
- Propositional idea density is the Kintsch and Keenan result: reading time tracks the number of atomic ideas, not words. It is famous from Snowdon's Nun Study. CPIDR computes it at ~0.97 agreement with human raters — but only given a POS tagger. Closed-class counts (prepositions and conjunctions per 10 words) are the exact-computable slice.
- Dependency length and embedding depth (Yngve 1960; Gibson's dependency locality theory) explain why center-embedding and long subject-verb separations hurt. The reader holds unfinished structure in working memory. True parse metrics need a parser. Deterministic proxies exist: words before the first verb, subordinator counters, bracket and em-dash nesting depth.
- Cohesion (Coh-Metrix lineage: Graesser, McNamara, Louwerse, and colleagues) is what the formulas miss entirely. Two paragraphs with identical word and sentence lengths differ enormously in difficulty, depending on whether sentences share referents and signal their relations. This is also the most tractable target. Connective density is a pure word list. Referential cohesion is well approximated by adjacent-sentence content-word overlap after stemming. Both are fully stdlib-implementable.
- The fully-deterministic heuristic family is individually weak but jointly strong. It covers sentence-length distribution tails (the one 60-word sentence, which means hide), clauses per sentence, parenthetical nesting depth, nominalization suffix density (`-tion`, `-ment`, `-ance`), passives inside subordinate clauses, lexical density, and windowed type-token ratio. Together these proxy the constructs above, at full fidelity, with zero dependencies.
- A direct prose analog of SonarSource's cognitive complexity (Campbell's increment-plus-nesting scheme) does not exist in the literature or in tooling. A defensible mapping exists, though. Nesting increments for subordinate clauses and parenthetical asides. Fundamental increments for each clause beyond the first, and for each passive in a subordinate clause. Flow-break increments for adversative-connective chains and topic shifts. Referent-switch increments for pronouns with no nearby antecedent. Like Sonar's metric it would be argued rather than derived, and it would need validation against human difficulty ratings.

## Prior art in tooling

- Vale (Go, markdown-aware) is the direct precedent. Its readability style ships seven document-level metric rules — Flesch-Kincaid > 8, Reading Ease < 70, Fog > 10, SMOG > 10, LIX > 35, ARI > 8, Coleman-Liau > 9 — as editable YAML conditions. Code fences and inline code are excluded by scope before scoring. Alerts are document-level, which makes poor PR annotations.
- Hemingway Editor is exactly this: ARI for the grade badge, plus per-sentence flags only for sentences of 14+ words (per-sentence ARI 10-13 is "hard," 14+ is "very hard"). Adverbs are `-ly` minus a whitelist. Passive is be-verb plus participle. Complex phrases come from a substitution dictionary. Each detector is budgeted per 100 words. No parsing, no ML. Its entire mechanism is reproducible in a fitness check.
- proselint, write-good, and alex are regex-plus-word-list detectors (usage, weasel words, insensitive language) with no scoring. textlint is the best framework precedent: a typed markdown AST where rules subscribe to node types and code nodes never reach prose rules.
- The academic instruments (Coh-Metrix, TAALES, TAACO) demonstrate that cohesion and word familiarity outpredict sentence length. But they emit hundreds of unthresholded features from GUI tools with heavyweight NLP models. As CI gates they are structurally unusable.
- LLM-as-judge is the only approach that measures actual clarity rather than proxies. It is not deterministic even at temperature zero. Practice that works: pinned model snapshots, rubric anchoring with few-shot band definitions, coarse pass/warn/fail bands rather than numeric scores, and verdicts cached by content hash so unchanged prose can never newly fail. For a runner whose contract is reproducibility, an LLM judge stays advisory or verdict-frozen.

## The experiment: this repo's plan documents as the corpus

`docs/plans/archive/` holds two generations of the same document type. Six free-form plans from the TypeScript era are goal-and-checklist hybrids in varied shapes. The two plans written after the free-form style was rejected follow the may-journals template: numbered title, one-line blockquote, a Goal paragraph, numbered checkbox sections. The house quality judgment is on record — the template replaced the free-form style. That makes the corpus labeled. The question becomes whether the metrics agree with the judgment. We scored prose paragraphs, blockquotes, and list items separately (code spans, links, and version tokens masked), treating each markdown block end as a sentence boundary.

Document-level battery (grade-scale formulas except LIX):

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

Findings, stated plainly:

- The formulas invert the house judgment. Every formula scores the template plans as harder on average than the free-form plans they replaced. That holds at document level (table above) and at unit level (list-item Coleman-Liau mean 15.6 template vs 9.9 free-form, with every unit-level AUC on the wrong side of 0.5). The formulas reward the conversational filler the template eliminated. They punish the information density it demanded. A gate built on them would have defended the rejected style.
- Genuine paragraphs score sanely. The two Goal paragraphs are real 72-100 word prose, the formulas' native sample size. They land at grade 11.6-13.5 with the character formulas in the same band. That is exactly where well-written technical prose belongs, and stable enough to be meaningful.
- Per-paragraph scores are noise. `plan-checks-abstractions.md` has 26 paragraphs, one author, one register. Flesch-Kincaid swings from 2.3 to 19.7 inside it — a 17-grade spread in a uniformly written document.
- The markdown artifact dwarfs everything. Scored without the block-boundary rule, `01-go-rewrite.md` jumps from grade 11.8 to 46.7 (roughly 104 words per "sentence"). Its 45 punctuation-free checkbox items merge into pseudo-sentences. Segmentation policy is worth 35 grades. No real quality difference on this corpus is worth 5.
- What separates the generations is structure, not sentence statistics: one blockquote, one Goal paragraph, numbered checkbox sections. That shape is deterministically checkable by a template gate. It is precisely the thing readability formulas cannot see.

An earlier round of this experiment used the changelog rewrite (essay bullets vs the tight 365-character style) as its corpus. It reached the same negative conclusions. Per-unit gating fails (best AUC 0.701 there). Aggregation helps (0.824 at section level). Masking code spans is essential. Formulas explode on notation-dense text (Reading Ease −101 on one 149-character bullet). But the corpus was weaker: changelog bullets are notation rather than paragraphs, and its one clean separator, the character cap, was true by construction. The plans corpus supersedes it as the example. The changelog numbers survive only as corroboration.

## Implications for a fitness check

What the evidence supports, in order of defensibility:

1. Structural budgets, not readability scores, at fine granularity. Both of this repo's own quality interventions were structural gates: `changelog-bullets` (size caps and counts) and the may-journals plan template (one blockquote, Goal, numbered checkbox sections). In both corpora, structure separated good from bad where formulas failed or inverted. Add Hemingway-style per-sentence rules (flag sentences of 30+ words, length-gated so short sentences are never scored) and depth counters (parenthetical nesting, clause chaining). All exact, all stdlib.
2. Document-level outlier detection with a character-based formula. If a grade-style score is wanted, use Coleman-Liau or LIX over a whole markdown file. Apply a frozen masking spec first: strip fences, placeholder inline code and URLs, treat block boundaries as sentence boundaries. Refuse to score below a 100-word minimum — report insufficient text instead. Use warn-level bands, not hard fails: the formulas are gameable, and optimizing them can make prose worse.
3. Cohesion proxies are the unexplored high ground. Connective density and adjacent-sentence word overlap are deterministic, cheap, and target what formulas miss — whether sentences connect. No mainstream CI tool ships them.
4. A Sonar-style prose cognitive-complexity score is buildable and would be novel: an additive increment scheme over nesting, clause load, flow breaks, and referent switches. It should be validated against human judgments on this repo's docs before it gates anything.
5. Thresholds must be calibrated in-house. Scores are implementation-defined once a tokenizer and splitter are frozen. Literature thresholds (grade 8, LIX 35) do not transfer to technical prose. Well-written technical prose legitimately runs grade 11-15 here.

## Sources

- Formula primaries: Flesch 1948. Kincaid et al. 1975 (Navy recalibration). Gunning 1952. McLaughlin 1969 (SMOG). Coleman and Liau 1975. Senter and Smith 1967 (ARI). Dale and Chall 1948/1995. Björnsson 1968 and Anderson 1983 (LIX/RIX). Critiques: Duffy and Kabance 1982. Davison and Kantor 1982. DuBay 2004 survey.
- Cognitive measures: Kintsch and Keenan 1973 (propositions). Brown et al. 2008 (CPIDR, Behavior Research Methods). Sirts et al. 2017 (DEPID). Gibson 1998/2000 (dependency locality). Hale 2001 and Levy 2008 (surprisal). Wilcox et al. 2023 (TACL). Shain et al. 2024 (PNAS). Graesser et al. 2004 (Coh-Metrix). Campbell 2018 (Cognitive Complexity, SonarSource whitepaper).
- Tooling: Vale readability style (`github.com/errata-ai/readability`). Hemingway mechanics (freeCodeCamp deconstruction). textlint TxtAST architecture. textstat. retext-readability (per-sentence ensemble voting).
- Experiment artifacts: scoring scripts in the session scratchpad (`readability.py`, `plans_experiment.py`). Primary corpus is `docs/plans/archive/` at `8ea28ac`. The corroborating changelog corpus is `CHANGELOG.md` at `dcb8376` and its parent.
