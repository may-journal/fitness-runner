---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# Judging the cognitive complexity of prose

Can a fitness check score written paragraphs the way `go-complexity` scores functions? This is a research survey plus an experiment on this repo's own plan documents.

## Verdict

Yes for coarse, document-level outlier detection with a handful of deterministic metrics. No for precise per-paragraph judgment. The classical readability formulas measure far less than their names suggest.

Every classical formula is a two-variable regression on the same two surface proxies. Word length stands in for vocabulary familiarity via Zipf's law. Sentence length stands in for syntactic working-memory load.

None measures cohesion, ordering, ambiguity, or meaning. Scrambling word order inside every sentence changes no score.

## Key findings

- Three formulas compute cleanly in deterministic stdlib-only Go: Coleman-Liau, ARI, and LIX. The rest need dictionaries, word lists, or human judgment.
- The formulas were normed on 100+ word samples. Per-paragraph gating is statistically meaningless. Document-level outlier detection is defensible.
- On this repo's archived plans the formulas invert the house judgment. They score the template style as harder than the rejected free-form style.
- The best cognitive measures need a model, tagger, or parser. What survives with zero dependencies is structural budgets and cohesion proxies.
- No one has built a true prose analog of SonarSource's cognitive-complexity metric. The design pattern transfers. The field is open.

## Detail documents

- The classical formulas: [0001-formulas.md](./0001-formulas.md).
- Cognitive science beyond the formulas: [0001-cognitive-science.md](./0001-cognitive-science.md).
- Prior art in tooling: [0001-prior-art.md](./0001-prior-art.md).
- The experiment on this repo: [0001-calibration.md](./0001-calibration.md).
- Implications and limits: [0001-limits.md](./0001-limits.md).
- Sources: [0001-sources.md](./0001-sources.md).
