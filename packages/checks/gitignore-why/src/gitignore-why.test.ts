import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import gitignoreWhyCheck, { classifyLine, findViolations, isExplanatoryComment } from './index.js';

/** Write a .gitignore with the given content into a fresh temp dir and return that dir. */
function tempRepo(content: string): string {
  const dir = mkdtempSync(join(tmpdir(), 'gitignore-why-'));
  writeFileSync(join(dir, '.gitignore'), content);
  return dir;
}

describe('gitignoreWhyCheck', () => {
  it('passes with filesChecked 0 when .gitignore is absent', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'gitignore-why-'));
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('passes when every pattern has an explanatory comment above it', async () => {
    const dir = tempRepo(
      '# dependencies\nnode_modules/\n\n# build output\ndist\n\n# keep this tracked\n!dist/keep.js\n'
    );
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails a pattern with a blank line above it', async () => {
    const dir = tempRepo('# deps\nnode_modules/\n\ndist\n');
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      '.gitignore:4: pattern "dist" has no explanatory # comment on the line above',
    ]);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails a pattern with another pattern directly above it', async () => {
    const dir = tempRepo('# deps\nnode_modules/\ndist\n');
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      '.gitignore:3: pattern "dist" has no explanatory # comment on the line above',
    ]);
  });

  it('fails a leading pattern on the very first line', async () => {
    const dir = tempRepo('node_modules/\n');
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      '.gitignore:1: pattern "node_modules/" has no explanatory # comment on the line above',
    ]);
  });

  it('fails a pattern whose comment above is a bare # with no text', async () => {
    const dir = tempRepo('#\nnode_modules/\n');
    const result = await gitignoreWhyCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      '.gitignore:2: pattern "node_modules/" has no explanatory # comment on the line above',
    ]);
  });
});

describe('classifyLine', () => {
  it('classifies blank, comment, and pattern lines', () => {
    expect(classifyLine('   ')).toBe('blank');
    expect(classifyLine('# why')).toBe('comment');
    expect(classifyLine('node_modules/')).toBe('pattern');
  });
});

describe('isExplanatoryComment', () => {
  it('is true only for a # with real text after it', () => {
    expect(isExplanatoryComment('# real reason')).toBe(true);
    expect(isExplanatoryComment('#')).toBe(false);
    expect(isExplanatoryComment('node_modules/')).toBe(false);
  });
});

describe('findViolations', () => {
  it('returns no violations when every pattern is explained', () => {
    expect(findViolations('# why\nnode_modules/\n')).toEqual([]);
  });
});
