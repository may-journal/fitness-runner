#!/usr/bin/env node
/** Lists @mayjournal workspace packages that are published to npm (non-private). */
import { readFileSync, readdirSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '../..');

/** @returns {{ dir: string, name: string }[]} */
export function listPublishablePackages() {
  const entries = [];
  const checkDirs = [
    'packages/runner',
    'packages/shared',
    'packages/checks-bundle',
    ...readdirSync(join(root, 'packages/checks'), { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => `packages/checks/${d.name}`),
  ];

  for (const dir of checkDirs) {
    const pkgPath = join(root, dir, 'package.json');
    if (!existsSync(pkgPath)) continue;
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
    if (pkg.private === true) continue;
    if (!pkg.name?.startsWith('@mayjournal/')) continue;
    entries.push({ dir, name: pkg.name });
  }

  const sortKey = (name) => {
    if (name === '@mayjournal/fitness-shared') return '0';
    if (name === '@mayjournal/fitness-checks') return '2';
    if (name === '@mayjournal/fitness') return '3';
    if (name.startsWith('@mayjournal/fitness-check-')) return `1-${name}`;
    return `9-${name}`;
  };

  entries.sort(
    (a, b) => sortKey(a.name).localeCompare(sortKey(b.name)) || a.name.localeCompare(b.name),
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
