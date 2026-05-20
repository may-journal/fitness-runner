#!/usr/bin/env node
import { isMainModule } from './utils/isMainModule.js';
import { run } from './runner/index.js';

if (isMainModule(import.meta.url)) run();
