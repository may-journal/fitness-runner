#!/usr/bin/env node
import { isMainModule } from './utils/isMainModule.js';
import { run, enUS } from './runner/index.js';

export { run, enUS };
export * from './types/index.types.js';

if (isMainModule(import.meta.url)) run();
