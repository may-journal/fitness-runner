/** String literals shorter than this are ignored as noise (single chars, tiny tokens). */
export const MIN_LENGTH = 3;

/**
 * Idiomatic JS/Node tokens where the literal IS the clearest spelling — extracting a constant
 * would hurt readability. Closed sets only: buffer encodings, child_process stdio modes, and
 * `typeof` results. Anything domain-specific is still flagged.
 */
export const IDIOMATIC_VALUES = new Set([
  // Buffer/stream encodings (BufferEncoding)
  'ascii',
  'base64',
  'base64url',
  'binary',
  'hex',
  'latin1',
  'ucs2',
  'utf8',
  'utf-8',
  'utf16le',
  // child_process stdio modes
  'inherit',
  'ignore',
  'overlapped',
  'pipe',
  // typeof results
  'bigint',
  'boolean',
  'function',
  'number',
  'object',
  'string',
  'symbol',
  'undefined',
  // language directives (a directive prologue cannot be replaced by a constant)
  'use strict',
]);

/** Words that, immediately before a string, mark it a module specifier to skip (`import`/`require`/`from`). */
const MODULE_WORDS = new Set(['from', 'import', 'require']);

/** A scanned string literal: its text and 1-based line. */
export type Literal = { line: number; value: string };

/** Mutable scan position and accumulated state for the hand lexer. */
type Cursor = {
  content: string;
  i: number;
  lastWord: string; // last completed identifier — used to detect module specifiers
  line: number;
  out: Literal[];
  prev: string; // last significant char — used for regex-vs-division
  word: string; // identifier currently being read
};

/** True when a `/` following prevChar is a division operator, not the start of a regex literal. */
function isDivision(prevChar: string): boolean {
  return /[A-Za-z0-9_$)\]}'"`]/.test(prevChar);
}

/** True when a char can be part of an identifier/keyword word. */
function isWordChar(ch: string): boolean {
  return /[A-Za-z0-9_$]/.test(ch);
}

/** True for inline whitespace (newlines are handled separately for line counting). */
function isSpace(ch: string): boolean {
  return ch === ' ' || ch === '\t' || ch === '\r';
}

/** True for a regex flag char (a…z). */
function isFlag(ch: string): boolean {
  return /[a-z]/i.test(ch);
}

/** The char at idx, or '' when out of range. */
function charAt(content: string, idx: number): string {
  return content[idx] ?? '';
}

/** True when the source at the cursor starts with `token`. */
function at(cur: Cursor, token: string): boolean {
  return cur.content.startsWith(token, cur.i);
}

/** Finalize the in-progress word into lastWord. */
function endWord(cur: Cursor): void {
  if (cur.word) {
    cur.lastWord = cur.word;
    cur.word = '';
  }
}

/** Advance past a `//` line comment (the newline is handled on the next step). */
function skipLineComment(cur: Cursor): void {
  cur.i += 2;
  while (cur.i < cur.content.length && cur.content[cur.i] !== '\n') cur.i += 1;
}

/** Advance past a block comment (slash-star … star-slash), counting the newlines inside it. */
function skipBlockComment(cur: Cursor): void {
  cur.i += 2;
  while (cur.i < cur.content.length && !at(cur, '*/')) {
    if (cur.content[cur.i] === '\n') cur.line += 1;
    cur.i += 1;
  }
  cur.i += 2;
}

/** Next char-class nesting state given the current char. */
function nextClassState(ch: string, inClass: boolean): boolean {
  if (ch === '[') return true;
  if (ch === ']') return false;
  return inClass;
}

/** True when `ch` closes the regex body (an unescaped `/` outside a char class). */
function isRegexEnd(ch: string, inClass: boolean): boolean {
  return ch === '/' && !inClass;
}

/** Advance through a regex body up to and including the closing `/` (honoring char classes). */
function scanRegexBody(cur: Cursor): void {
  let inClass = false;
  while (cur.i < cur.content.length) {
    const ch = cur.content[cur.i];
    if (ch === '\\') {
      cur.i += 2;
      continue;
    }
    if (ch === '\n') break; // unterminated — bail safely
    cur.i += 1;
    if (isRegexEnd(ch, inClass)) break;
    inClass = nextClassState(ch, inClass);
  }
}

/** Advance past a regex literal and its flags. */
function skipRegex(cur: Cursor): void {
  cur.i += 1; // opening /
  scanRegexBody(cur);
  while (cur.i < cur.content.length && isFlag(cur.content[cur.i])) cur.i += 1;
  cur.prev = '/';
  cur.lastWord = '';
}

/** Record a scanned literal unless it's a module specifier, below the length floor, or idiomatic. */
function recordLiteral(cur: Cursor, value: string, line: number, isModule: boolean): void {
  if (isModule || value.length < MIN_LENGTH || IDIOMATIC_VALUES.has(value)) return;
  cur.out.push({ line, value });
}

/** Read a single- or double-quoted string literal, honoring escapes, and record it. */
function readString(cur: Cursor): void {
  endWord(cur);
  const quote = cur.content[cur.i];
  const startLine = cur.line;
  const isModule = MODULE_WORDS.has(cur.lastWord);
  cur.i += 1;
  let value = '';
  while (cur.i < cur.content.length && cur.content[cur.i] !== quote) {
    if (cur.content[cur.i] === '\\') {
      value += charAt(cur.content, cur.i + 1);
      cur.i += 2;
      continue;
    }
    if (cur.content[cur.i] === '\n') cur.line += 1;
    value += cur.content[cur.i];
    cur.i += 1;
  }
  cur.i += 1; // closing quote
  cur.prev = quote;
  cur.lastWord = '';
  recordLiteral(cur, value, startLine, isModule);
}

/** Skip a template literal (out of scope in v1), counting the newlines inside it. */
function skipTemplate(cur: Cursor): void {
  endWord(cur);
  cur.i += 1;
  while (cur.i < cur.content.length && cur.content[cur.i] !== '`') {
    if (cur.content[cur.i] === '\\') {
      cur.i += 2;
      continue;
    }
    if (cur.content[cur.i] === '\n') cur.line += 1;
    cur.i += 1;
  }
  cur.i += 1;
  cur.prev = '`';
  cur.lastWord = '';
}

/** Handle a `/`: line comment, block comment, regex literal, or division operator. */
function stepSlash(cur: Cursor): void {
  if (at(cur, '//')) return skipLineComment(cur);
  if (at(cur, '/*')) return skipBlockComment(cur);
  if (!isDivision(cur.prev)) return skipRegex(cur);
  endWord(cur);
  cur.prev = '/';
  cur.i += 1;
}

/** Handle a non-quote, non-slash char: newline, whitespace, word char, or other delimiter. */
function stepText(cur: Cursor, ch: string): void {
  if (ch === '\n') {
    endWord(cur);
    cur.line += 1;
    cur.i += 1;
    return;
  }
  if (isSpace(ch)) {
    endWord(cur);
    cur.i += 1;
    return;
  }
  if (isWordChar(ch)) {
    cur.word += ch;
    cur.prev = ch;
    cur.i += 1;
    return;
  }
  endWord(cur);
  cur.prev = ch;
  cur.i += 1;
}

/** Advance the cursor by one token, dispatching on the current char. */
function step(cur: Cursor): void {
  const ch = cur.content[cur.i];
  if (ch === '/') return stepSlash(cur);
  if (ch === "'" || ch === '"') return readString(cur);
  if (ch === '`') return skipTemplate(cur);
  return stepText(cur, ch);
}

/**
 * Extracts single- and double-quoted string literals from source, skipping line/block comments,
 * regex literals, and template literals, and dropping module specifiers (the string after
 * `import`/`require`/`from`). Template literals are out of scope in v1.
 */
export function scanStringLiterals(content: string): Literal[] {
  const cur: Cursor = { content, i: 0, lastWord: '', line: 1, out: [], prev: '', word: '' };
  while (cur.i < content.length) step(cur);
  return cur.out;
}
