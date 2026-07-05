/** Shared TypeScript compiler defaults for fitness-runner configs. */
const base = {
  module: 'NodeNext',
  moduleResolution: 'NodeNext',
  skipLibCheck: true,
  strict: true,
  target: 'ES2022',
  // TypeScript 6 no longer auto-includes @types/node ambient globals under NodeNext; name it
  // explicitly so `node:*` imports and globals (process, Buffer) resolve. Tests import from
  // 'vitest' directly, so restricting `types` to node doesn't drop test globals.
  types: ['node'],
};

module.exports = {
  /** Runner package build (declaration emit + source maps). */
  build: { ...base, declaration: true, declarationMap: true, noEmit: false, sourceMap: true },
  /** Check and shared package build (declaration emit). */
  check: { ...base, declaration: true, declarationMap: true, noEmit: false },
  /** Type-aware lint / no-emit analysis. */
  lint: { ...base, noEmit: true },
};
