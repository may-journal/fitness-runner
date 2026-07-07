/** @type {import('@mayjournal/fitness').FitnessConfig} */
const { defaultChecks } = require('@mayjournal/fitness-checks');

// Opt-in mermaid diagram + callout-table checks live in this repo, so we run them on our
// own architecture docs (dogfooding). They only activate on diagrams paired with a callout
// table, so unrelated diagrams are untouched.
const mermaidChecks = [
  'mermaid-callouts',
  'mermaid-callout-why',
  'mermaid-diagram-prose',
  'mermaid-legend',
  'mermaid-level-bleed',
];

module.exports = {
  // Every default check plus the opt-in mermaid + dependency-currency checks — the repo runs
  // the full suite on itself (dogfooding). Nothing is disabled.
  checks: [
    ...defaultChecks,
    ...mermaidChecks,
    'dependency-currency',
    'gitignore-why',
    // This repo's docs are kebab-case, so we enable the kebab-case flavor here (dogfooding);
    // the camelCase flavor stays opt-in (it would fail our hyphenated/numbered docs).
    'markdown-filename-kebab-case',
  ],
};
