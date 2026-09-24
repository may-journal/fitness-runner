---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore Liau Flesch Kincaid Gunning Björnsson Yngve Kintsch Keenan Snowdon Vasishth Frazier Demberg Schuler Rajkumar Graesser McNamara Louwerse Kulikowich Crossley Hemingway proselint textlint reviewdog retext remark nlcst mdast textstat cmudict Pyphen Campbell's Goodhart's surprisal polysyllabic polysyllable nominalization nominalizations subordinator subordinators relativizer relativizers Zipf cloze CPIDR DEPID Coh Metrix TAACO TAALES SUBTLEX stopword stemmer appositives Levy's -->
<!-- cspell:ignore Chall recalibrated Kabance Davison Kantor undercount PNAS tion ment ance outpredict unthresholded periodless norming backticked unbackticked gameable Senter Sirts TACL Shain blockquote blockquotes devdeps FKGL -->

# The classical formulas

Every classical formula is a two-variable regression on the same surface proxies. Three compute cleanly in stdlib-only Go: Coleman-Liau, ARI, and LIX. They need only letter, word, and sentence counts. The syllable formulas are implementation-defined, and no two tools agree. Gunning Fog is not mechanizable, and Dale-Chall needs an embedded word list.

| Formula | Inputs | Stdlib-Go viable? | Notes |
| --- | --- | --- | --- |
| Flesch Reading Ease | words/sentences, syllables/words | heuristic only | `206.835 − 1.015(W/S) − 84.6(Syl/W)`; most syllable-sensitive (84.6 coefficient) |
| Flesch-Kincaid Grade | same | heuristic only | `0.39(W/S) + 11.8(Syl/W) − 15.59`; rank-identical to Reading Ease |
| Gunning Fog | words/sentences, complex-word rate | no | "complex word" needs human judgment; every automated Fog is a house variant |
| SMOG | polysyllable count per 30 sentences | no (scale) | defined on 30-sentence samples; at 2 sentences each polysyllabic word moves a full grade |
| Coleman-Liau | letters, words, sentences | yes | `5.88(L/W) − 29.6(S/W) − 15.8`; designed for machine scoring, no syllables |
| ARI | characters, words, sentences | yes | `4.71(C/W) + 0.5(W/S) − 21.43`; digits count, so version strings inflate it |
| Dale-Chall | 3,000-word list, words/sentences | no (list) | best on general prose, useless on technical prose; scores pin at the ceiling |
| LIX | words/sentences, share of 7+-letter words | yes | `(W/S) + 100(long/W)`; simplest defensible construct, robust to tokenization |

## Validation pedigree

Flesch-Kincaid was recalibrated in 1975. The test used ~531 Navy enlisted personnel. They read technical training material. That is the closest historical analog to developer docs. Even there the standard error neared two grade levels.

The critique literature attacked the scores. See Duffy and Kabance 1982, and Davison and Kantor 1982. Rewriting text to optimize scores did not reliably improve comprehension. Short words and chopped sentences are not clarity. Goodhart's law applies in full.

A hard threshold rewards terse jargon. It prefers "use the CLI to init the cfg". That beats "use the command-line interface to initialize the configuration".

## Three mechanical traps

Three implementation traps dominate any build.

| Trap | Why it bites |
| --- | --- |
| Syllable counting | exact counts need a ~134k-entry dictionary; vowel heuristics hit 85-95% but undercount code identifiers; every syllable score is implementation-defined |
| Sentence segmentation | naive `[.!?]` breaks on "e.g.", versions, and paths; punctuation-free headings and list items merge into mega-sentences; one boundary error adds 7.8 grades to Flesch-Kincaid and 10 to ARI |
| Short texts | below ~100 words the formulas are ratio estimators with tiny denominators; the literature says do not score them; report "insufficient text" instead |
