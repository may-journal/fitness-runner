'use strict';
const { execSync } = require('node:child_process');
const { readFileSync, readdirSync, statSync, writeFileSync } = require('node:fs');
const { join } = require('node:path');

const root = join(__dirname, '../..');
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

function collectPackageJsonPaths(dir, paths = []) {
  for (const entry of readdirSync(dir)) {
    const fullPath = join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      collectPackageJsonPaths(fullPath, paths);
      continue;
    }
    if (entry === 'package.json') paths.push(fullPath);
  }
  return paths;
}

const packageJsonPaths = [
  join(root, 'package.json'),
  ...collectPackageJsonPaths(join(root, 'packages')),
];
const bumpedPaths = [];
for (const pkgPath of packageJsonPaths) {
  const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
  if (!pkg.version) continue;
  pkg.version = pkg.version.replace(/-?\d{4}\.\d{2}\.\d{2}\.\d{4}$/, `-${ts}`);
  writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
  bumpedPaths.push(pkgPath);
}

execSync('npm install', { cwd: root, stdio: 'inherit' });
const prettierTargets = ['CHANGELOG.md', ...bumpedPaths].join(' ');
execSync(`npx prettier ${prettierTargets} --write`, { cwd: root, stdio: 'inherit' });
execSync(
  `git add CHANGELOG.md package-lock.json ${bumpedPaths.map((p) => p.replace(`${root}/`, '')).join(' ')}`,
  { cwd: root, stdio: 'inherit' }
);
