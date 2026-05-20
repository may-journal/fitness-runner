const { check } = require('./tsconfig.compiler.cjs');

/** Compiler defaults for fitness check package builds (exported as JSON for extends). */
module.exports = { compilerOptions: check };
