// cspell:ignore packagejson
// Regenerates the committed plain-text word lists in this directory from the
// cspell dictionary packages installed under <repo>/node_modules/@cspell/.
//
// Run from the repository root:
//
//   node go/internal/spell/dict/generate.mjs
//
// Every output file is sorted, lowercased, deduplicated, one word per line,
// with a provenance header. All source packages are MIT licensed (Street Side
// Software cspell-dicts; dict-en_us derives from the Hunspell en_US
// dictionary, itself built from SCOWL — see the LICENSE file inside each
// package). The generator is intentionally lossy in ways the Go engine
// expects: forbidden entries (leading "!") and comments are dropped, the
// case-insensitive marker (leading "~") is stripped, entries containing
// spaces or slashes are dropped (they can never match a token), and
// everything is lowercased because the engine's lookup is case-insensitive.
import { readFileSync, writeFileSync } from 'node:fs';
import { gunzipSync } from 'node:zlib';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import { importTrie, iteratorTrieWords } from 'cspell-trie-lib';

const outDir = dirname(fileURLToPath(import.meta.url));
// The @cspell scope directory, found through module resolution (the packages'
// exports maps expose cspell-ext.json) — works from any cwd, no hand-built
// installation paths.
const requireFromHere = createRequire(import.meta.url);
const pkgRoot = dirname(dirname(requireFromHere.resolve('@cspell/dict-en_us/cspell-ext.json')));

/** Reads a dictionary source file, decompressing when the name ends in .gz. */
function readSource(relPath) {
  const raw = readFileSync(join(pkgRoot, relPath));
  return (relPath.endsWith('.gz') ? gunzipSync(raw) : raw).toString('utf8');
}

/** Words from a cspell plain-text dictionary file (one word per line). */
function wordsFromText(text) {
  return text.split('\n').map((line) => line.trim());
}

/** Words from a cspell .trie/.trie.gz dictionary via cspell-trie-lib. */
function wordsFromTrie(text) {
  return [...iteratorTrieWords(importTrie(text.split('\n')))];
}

/** Normalizes one raw dictionary entry; returns null when it should be dropped. */
function normalize(raw) {
  let word = raw.trim();
  if (word === '' || word.startsWith('#') || word.startsWith('!')) return null;
  word = word.replace(/^~/, '').replace(/’/g, "'");
  // Keep entries the engine can match: plain words (letters + apostrophes)
  // and compound entries (digits . _ + -) matched against extended tokens,
  // e.g. prettier-plugin-packagejson. Anything else (spaces, slashes) can
  // never match a token.
  if (!/^[\p{L}\p{M}\w'.+-]+$/u.test(word) || !/\p{L}/u.test(word)) return null;
  return word.toLowerCase();
}

/** Writes one word list: provenance header plus sorted deduplicated words. */
function emit(outName, sources, words) {
  const list = [...new Set(words.map(normalize).filter((w) => w !== null))].sort();
  const header = [
    `# ${outName} — generated word list for the fitness cspell check (Go port).`,
    `# Sources (installed under node_modules/@cspell/, all MIT licensed):`,
    ...sources.map((s) => `#   ${s.pkg}@${s.version}: ${s.files.join(', ')}`),
    `# Regenerate from the repository root with:`,
    `#   node go/internal/spell/dict/generate.mjs`,
    `# ${list.length} words, lowercased, sorted, one per line.`,
  ];
  writeFileSync(join(outDir, outName), header.concat(list, '').join('\n'));
  console.log(`${outName}: ${list.length} words`);
}

/** Version of an installed @cspell dictionary package. */
function versionOf(pkg) {
  return JSON.parse(readFileSync(join(pkgRoot, pkg.replace('@cspell/', ''), 'package.json')))
    .version;
}

/** Builds one word list from the given package-relative dictionary files. */
function build(outName, pkg, files, extract = wordsFromText) {
  const words = files.flatMap((f) => extract(readSource(f)));
  emit(outName, [{ pkg, version: versionOf(pkg), files }], words);
}

build('en-us.txt', '@cspell/dict-en_us', ['dict-en_us/en_US.trie.gz'], wordsFromTrie);
build('software-terms.txt', '@cspell/dict-software-terms', [
  'dict-software-terms/dict/softwareTerms.txt.gz',
  'dict-software-terms/dict/software-tools.txt',
  'dict-software-terms/dict/computing-acronyms.txt',
  'dict-software-terms/dict/coding-compound-terms.txt',
  'dict-software-terms/dict/networkingTerms.txt',
  'dict-software-terms/dict/webServices.txt',
  'dict-software-terms/dict/software-terms-alternative.txt',
]);
build('typescript.txt', '@cspell/dict-typescript', ['dict-typescript/dict/typescript.txt']);
build('node.txt', '@cspell/dict-node', ['dict-node/dict/node.txt']);
build('npm.txt', '@cspell/dict-npm', ['dict-npm/dict/npm.txt']);
build('html.txt', '@cspell/dict-html', ['dict-html/dict/html.txt.gz']);
build('css.txt', '@cspell/dict-css', ['dict-css/dict/css.txt']);
build('companies.txt', '@cspell/dict-companies', ['dict-companies/dict/companies.txt']);
build('filetypes.txt', '@cspell/dict-filetypes', ['dict-filetypes/filetypes.txt.gz']);
build('golang.txt', '@cspell/dict-golang', ['dict-golang/dict/go.txt']);
build('git.txt', '@cspell/dict-git', ['dict-git/dict/git-terms.txt']);
build('shell.txt', '@cspell/dict-shell', [
  'dict-shell/dict/bash-words.txt',
  'dict-shell/dict/shell-all-words.txt',
]);
build('fullstack.txt', '@cspell/dict-fullstack', ['dict-fullstack/dict/fullstack.txt']);
build('aws.txt', '@cspell/dict-aws', ['dict-aws/dict/aws.txt']);
// @cspell/dict-markdown is deliberately absent: after normalization (its
// entries are HTML-tag suffix rules) it contributes zero words.
