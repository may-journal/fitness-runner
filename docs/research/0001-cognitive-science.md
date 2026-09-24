---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Cognitive science beyond the formulas

The best-evidenced measures need heavy tooling. Each needs a language model, POS tagger, or parser. Each also has a zero-dependency proxy. Those proxies are structural budgets and cohesion signals.

## Per-word surprisal

Surprisal treats effort as prediction error (Hale 2001; Levy 2008). The formula is `−log P(word | context)`. It is the field's best difficulty predictor. The logarithmic effect holds across 11 languages (PNAS 2024).

It needs a language model. Oddly, small models like GPT-2 fit reading times better than large ones. The only proxy is word-frequency rarity from a list.

## Propositional idea density

Reading time tracks atomic ideas, not words. This is the Kintsch and Keenan 1973 result. It is famous from Snowdon's Nun Study.

CPIDR matches human raters at ~0.97, but needs a POS tagger. The exact slice is closed-class counts. Count prepositions and conjunctions per 10 words.

## Dependency length

Long dependencies raise load (Yngve 1960; Gibson). Center-embedding and long subject-verb gaps hurt. The reader holds unfinished structure in memory. True parse metrics need a parser.

| Proxy | Signal |
| --- | --- |
| Words before the first verb | a delayed head |
| Subordinator counts | clause chaining |
| Bracket and em-dash depth | nesting |

## Cohesion

Cohesion is what the formulas miss entirely. It comes from the Coh-Metrix lineage (Graesser, McNamara, Louwerse). Two paragraphs can share lengths yet differ in difficulty. The split is whether sentences share referents and signal relations. This target is also the most tractable.

Connective density is a pure word list. Referential cohesion needs adjacent-sentence content-word overlap after stemming. Both are stdlib.

## Deterministic heuristics

The heuristic family is individually weak but jointly strong. Together these proxy the constructs above. They need zero dependencies.

| Heuristic | What it counts |
| --- | --- |
| Sentence-length tails | the one very long sentence |
| Clause load | clauses per sentence |
| Nesting depth | parenthetical and bracket depth |
| Nominalization density | `-tion`, `-ment`, `-ance` suffixes |
| Passive density | passives in subordinate clauses |
| Lexical density | content words over the total |
| Type-token ratio | windowed vocabulary variety |

## A Sonar analog

No direct prose analog of SonarSource's cognitive complexity exists yet. It would follow Campbell's increment-plus-nesting scheme. A defensible mapping does exist.

| Increment | Trigger |
| --- | --- |
| Nesting | subordinate clauses and asides |
| Fundamental | each extra clause, each passive |
| Flow break | adversative connective chains |
| Referent switch | a pronoun with no near antecedent |
