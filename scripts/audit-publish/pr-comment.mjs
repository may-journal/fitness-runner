#!/usr/bin/env node
/** Reads audit JSON (stdin or file path) and prints a PR comment markdown body. */
import { readFileSync } from 'node:fs';
import { checkRunnerTarballGate, formatMarkdownReport } from './format-report.mjs';

const inputPath = process.argv[2];
const raw = inputPath ? readFileSync(inputPath, 'utf8') : readFileSync(0, 'utf8');
const data = JSON.parse(raw);
const rows = data.packages ?? data;
const gate = data.gate ?? checkRunnerTarballGate(rows, { runnerMaxTarball: data.runnerMaxTarball });
const attwRows = data.attw ?? [];

process.stdout.write(formatMarkdownReport(rows, { attwRows, gate }));
