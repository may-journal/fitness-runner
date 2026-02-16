#!/usr/bin/env node
import { run, UNKNOWN_CHECK_PREFIX, UNKNOWN_CHECK_SPEC_NONE, PLEASE_FIX_ITEMS, ERROR_BULLET } from './runner/index.js';

export { run, UNKNOWN_CHECK_PREFIX, UNKNOWN_CHECK_SPEC_NONE, PLEASE_FIX_ITEMS, ERROR_BULLET };

/* v8 ignore start */
if (!process.env.VITEST) run();
/* v8 ignore stop */
