const { ignorePaths } = require('./cspell.json');

module.exports = {
  compilerOptions: {
    declaration: true,
    declarationMap: true,
    module: 'NodeNext',
    moduleResolution: 'NodeNext',
    noEmit: false,
    outDir: '../../../packages/runner/dist',
    rootDir: '../../../packages/runner/src',
    skipLibCheck: true,
    sourceMap: true,
    strict: true,
    target: 'ES2022',
  },
  exclude: [
    ...ignorePaths,
    '../../runner/dist',
    '../../runner/dist/**',
    '../../../packages/shared/**',
    '../../../packages/checks/**',
    '../../../packages/checks-bundle/**',
    '../../../packages/runner/src/**/*.test.ts',
  ],
  include: ['../../../packages/runner/src/**/*.ts'],
};
