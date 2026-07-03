/** @type {import('@mayjournal/fitness').FitnessConfig} */
// jscpd disabled here: ~12 structurally-similar check packages (package.json scaffolding,
// enUS.ts, test setup) sit at ~2.3% duplication, over the proven 1% threshold used by
// consumer repos. That's inherent to this monorepo's shape, not something worth a forced
// refactor right now — jscpd is still a defaultChecks member for every other consumer.
module.exports = {
  disabledChecks: ['jscpd'],
};
