const { build } = require('./tsconfig.compiler.cjs');
const { ignorePaths } = require('./cspell.json');

module.exports = {
  compilerOptions: {
    ...build,
    outDir: '../../../packages/runner/dist',
    rootDir: '../../../packages/runner/src',
  },
  exclude: [
    ...ignorePaths,
    '../../runner/dist',
    '../../runner/dist/**',
    '../../../packages/shared/**',
    '../../../packages/checks/**',
    '../../../packages/checks-bundle/**',
    '../../../packages/runner/src/**/*.test.ts',
    '../../../packages/runner/src/**/*.bench.ts',
  ],
  include: ['../../../packages/runner/src/**/*.ts'],
};
