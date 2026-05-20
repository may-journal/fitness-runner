'use strict';
const { writeFileSync } = require('node:fs');
const { join } = require('node:path');

/** Writes JSON tsconfigs from CJS sources (eslint and tsc need JSON project paths). */
function generateTsconfigJson(configDir = __dirname) {
  writeFileSync(
    join(configDir, 'tsconfig.check.json'),
    `${JSON.stringify(require('./tsconfig.check.cjs'), null, 2)}\n`
  );
}

module.exports = { generateTsconfigJson };
