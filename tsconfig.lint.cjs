/** Compiler defaults for ESLint type-aware linting in consumer projects (used via temp tsconfig). */
module.exports = {
  compilerOptions: {
    module: 'NodeNext',
    moduleResolution: 'NodeNext',
    noEmit: true,
    skipLibCheck: true,
    strict: true,
    target: 'ES2022',
  },
};
