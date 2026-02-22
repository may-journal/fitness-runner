#!/usr/bin/env node
import { run, enUS } from './runner/index.js';

export { run, enUS };

/* v8 ignore start */
if (!process.env.VITEST) run();
/* v8 ignore stop */
