'use strict';
const { writeFileSync } = require('node:fs');
const { join } = require('node:path');

/** Writes JSON tsconfigs from CJS sources (eslint and tsc need JSON project paths). */
function generateTsconfigJson(configDir = __dirname) {
  for (const name of ['check', 'checks']) {
    writeFileSync(
      join(configDir, `tsconfig.${name}.json`),
      `${JSON.stringify(require(`./tsconfig.${name}.cjs`), null, 2)}\n`
    );
  }
}

module.exports = { generateTsconfigJson };
