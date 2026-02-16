const { ignorePaths } = require('./cspell.json');

module.exports = {
  compilerOptions: {
    declaration: true,
    declarationMap: true,
    module: 'NodeNext',
    moduleResolution: 'NodeNext',
    noEmit: false,
    outDir: './dist',
    rootDir: './src',
    skipLibCheck: true,
    sourceMap: true,
    strict: true,
    target: 'ES2022',
  },
  exclude: [...ignorePaths],
  include: ['**/*.ts'],
};
