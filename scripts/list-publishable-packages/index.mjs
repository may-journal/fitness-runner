#!/usr/bin/env node
/** Lists @mayjournal workspace packages that are published to npm (non-private). */
import { readFileSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '../..');

/** Monorepo workspaces published to npm (not individual check packages). */
const PUBLISHABLE_DIRS = ['packages/shared', 'packages/checks-bundle', 'packages/runner'];

/** @returns {{ dir: string, name: string }[]} */
export function listPublishablePackages() {
  const entries = [];

  for (const dir of PUBLISHABLE_DIRS) {
    const pkgPath = join(root, dir, 'package.json');
    if (!existsSync(pkgPath)) continue;
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
    if (pkg.private === true) continue;
    if (!pkg.name?.startsWith('@mayjournal/')) continue;
    entries.push({ dir, name: pkg.name });
  }

  const sortKey = (name) => {
    if (name === '@mayjournal/fitness-shared') return '0';
    if (name === '@mayjournal/fitness-checks') return '1';
    if (name === '@mayjournal/fitness') return '2';
    return `9-${name}`;
  };

  entries.sort(
    (a, b) => sortKey(a.name).localeCompare(sortKey(b.name)) || a.name.localeCompare(b.name)
  );
  return entries;
}

const isMain =
  process.argv[1] &&
  fileURLToPath(import.meta.url) === fileURLToPath(pathToFileURL(process.argv[1]).href);
if (isMain) {
  const format = process.argv[2] ?? 'json';
  const packages = listPublishablePackages();
  if (format === 'names') {
    for (const { name } of packages) console.log(name);
  } else if (format === 'dirs') {
    for (const { dir } of packages) console.log(dir);
  } else {
    console.log(JSON.stringify(packages));
  }
}
