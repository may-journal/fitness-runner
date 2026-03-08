'use strict';
const { execSync } = require('node:child_process');
const { readFileSync, writeFileSync } = require('node:fs');
const { join } = require('node:path');

const root = join(__dirname, '..');
const staged = execSync('git diff --cached --name-only', { cwd: root, encoding: 'utf8' })
  .trim()
  .split('\n');
if (!staged.includes('CHANGELOG.md')) process.exit(0);

const now = new Date();
const y = now.getFullYear();
const m = String(now.getMonth() + 1).padStart(2, '0');
const d = String(now.getDate()).padStart(2, '0');
const h = String(now.getHours()).padStart(2, '0');
const min = String(now.getMinutes()).padStart(2, '0');
const ts = `${y}.${m}.${d}.${h}${min}`;

const changelogPath = join(root, 'CHANGELOG.md');
let content = readFileSync(changelogPath, 'utf8');
content = content.replace(/^(### )\d{4}\.\d{2}\.\d{2}\.\d{4}/m, `$1${ts}`);
writeFileSync(changelogPath, content);

const pkgPath = join(root, 'package.json');
const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
pkg.version = pkg.version.replace(/-?\d{4}\.\d{2}\.\d{2}\.\d{4}$/, `-${ts}`);
writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');

execSync('npm install', { cwd: root, stdio: 'inherit' });
execSync('npx prettier CHANGELOG.md package.json --write', { cwd: root, stdio: 'inherit' });
execSync('git add CHANGELOG.md package.json package-lock.json', { cwd: root, stdio: 'inherit' });
