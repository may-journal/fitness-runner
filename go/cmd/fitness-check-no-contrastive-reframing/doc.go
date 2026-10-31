// Command fitness-check-no-contrastive-reframing flags the "not X, it's Y"
// reframing in markdown prose, where a sentence rejects a claim and then
// restates the real point. An example is `It's not a workout. It's a
// lifestyle.` It favors precision over recall, so it fires narrowly and
// leaves ordinary negations alone.
//
// # What it flags
//
//   - The split form: a demonstrative negation followed by a demonstrative assertion (`It's not a workout. It's a lifestyle.`).
//   - The single-sentence form: a demonstrative negation with a contrast pivot (`It's not a workout, but a lifestyle.`).
//
// # What it leaves alone
//
//   - A sentence that does not open with `It's`, `That's`, `This is`, or a kindred copula.
//   - An ordinary negation with no restatement, and the additive `not only X but also Y`.
//   - Fenced code, headings, tables, inline code, and quoted spans, via the shared `mdx` prose masking, so documenting the pattern does not self-flag.
//
// # Behavior
//
//   - Pass: no prose sentence matches, or the match sits in code or quotes.
//   - Fail: a sentence matches one of the two forms — one error per match, quoting the snippet.
//
// The detection is deliberately narrow; widen it only with tests that pin
// the false-positive line.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
