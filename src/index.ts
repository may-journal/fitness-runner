#!/usr/bin/env node
import { isMainModule } from './utils/isMainModule.js';
import { run, enUS } from './runner/index.js';

export { run, enUS };

if (isMainModule(import.meta.url)) run();
