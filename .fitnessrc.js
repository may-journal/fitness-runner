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
  checks: [...defaultChecks, ...mermaidChecks],
};
