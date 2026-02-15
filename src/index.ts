#!/usr/bin/env node
import { run } from './runner/index.js';

export { run };

/* v8 ignore start */
if (!process.env.VITEST) run();
/* v8 ignore stop */
