/** @type {import('@mayjournal/fitness').FitnessConfig} */
const { defaultChecks } = require('@mayjournal/fitness-checks');

module.exports = {
  // Every default check plus the opt-in dependency-currency check — the repo runs the full suite
  // on itself (dogfooding). Nothing is disabled.
  checks: [...defaultChecks, 'dependency-currency'],
};
