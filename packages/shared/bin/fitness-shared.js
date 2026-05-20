#!/usr/bin/env node
import { runBuild } from './build.js';
import { runTest } from './test.js';

const sub = process.argv[2];
const args = process.argv.slice(3);

if (sub === 'build') {
  runBuild();
} else if (sub === 'test') {
  runTest(args);
} else {
  console.error('Usage: fitness-shared <build|test> [args...]');
  process.exit(1);
}
