'use strict';
const { execSync } = require('node:child_process');
const { readFileSync, readdirSync, statSync, writeFileSync } = require('node:fs');
const { join } = require('node:path');

const CHANGELOG_TIMESTAMP_RE = /^(### )\d{4}\.\d{2}\.\d{2}\.\d{4}/m;
const VERSION_TIMESTAMP_RE = /-?\d{4}\.\d{2}\.\d{2}\.\d{4}$/;

/** @param {Date} date */
function formatTimestamp(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  const h = String(date.getHours()).padStart(2, '0');
  const min = String(date.getMinutes()).padStart(2, '0');
  return `${y}.${m}.${d}.${h}${min}`;
}

/** @param {string} content @param {string} ts */
function replaceChangelogTimestamp(content, ts) {
  return content.replace(CHANGELOG_TIMESTAMP_RE, `$1${ts}`);
}

/** @param {string} version @param {string} ts */
function bumpPackageVersion(version, ts) {
  return version.replace(VERSION_TIMESTAMP_RE, `-${ts}`);
}

/** @param {string[]} stagedFiles */
function isChangelogStaged(stagedFiles) {
  return stagedFiles.includes('CHANGELOG.md');
}

/** @param {string} dir @param {string[]} [paths] */
function collectPackageJsonPaths(dir, paths = []) {
  for (const entry of readdirSync(dir)) {
    // Skip node_modules: nested workspace deps (e.g. a non-hoisted chalk) are not our packages —
    // bumping/staging their package.json corrupts the dep and fails `git add` (it's gitignored).
    if (entry === 'node_modules') continue;
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

function main() {
  const root = join(__dirname, '../..');
  const staged = execSync('git diff --cached --name-only', { cwd: root, encoding: 'utf8' })
    .trim()
    .split('\n')
    .filter(Boolean);
  if (!isChangelogStaged(staged)) process.exit(0);

  const ts = formatTimestamp(new Date());

  const changelogPath = join(root, 'CHANGELOG.md');
  let content = readFileSync(changelogPath, 'utf8');
  content = replaceChangelogTimestamp(content, ts);
  writeFileSync(changelogPath, content);

  const packageJsonPaths = [
    join(root, 'package.json'),
    ...collectPackageJsonPaths(join(root, 'packages')),
  ];
  const bumpedPaths = [];
  for (const pkgPath of packageJsonPaths) {
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
    if (!pkg.version) continue;
    pkg.version = bumpPackageVersion(pkg.version, ts);
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
}

if (require.main === module) {
  main();
}

module.exports = {
  bumpPackageVersion,
  collectPackageJsonPaths,
  formatTimestamp,
  isChangelogStaged,
  replaceChangelogTimestamp,
};
