import { describe, it, expect, vi } from 'vitest';
import { readRepoFirstCheck, buildContextFeedback, getColumns } from './index.js';

describe('readRepoFirstCheck', () => {
  it('always passes', async () => {
    const result = await readRepoFirstCheck.run('/');
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('outputs feedback to stdout', async () => {
    const writeSpy = vi.spyOn(process.stdout, 'write').mockImplementation(() => true);
    try {
      await readRepoFirstCheck.run('/', { enabledCheckNames: ['changelog', 'eslint'] });
      expect(writeSpy).toHaveBeenCalledWith(expect.stringContaining('changelog'));
      expect(writeSpy).toHaveBeenCalledWith(expect.stringContaining('eslint'));
      expect(writeSpy).toHaveBeenCalledWith(expect.stringContaining('--no-verify'));
    } finally {
      writeSpy.mockRestore();
    }
  });

  it('includes enabledCheckNames in feedback when present', async () => {
    const feedback = buildContextFeedback(['changelog', 'eslint']);
    expect(feedback).toContain('changelog');
    expect(feedback).toContain('eslint');
  });

  it('outputs feedback without TTY requirement', async () => {
    const origIsTTY = process.stdin.isTTY;
    Object.defineProperty(process.stdin, 'isTTY', { value: false, configurable: true });
    try {
      const result = await readRepoFirstCheck.run('/');
      expect(result.ok).toBe(true);
    } finally {
      Object.defineProperty(process.stdin, 'isTTY', { value: origIsTTY, configurable: true });
    }
  });
});

describe('buildContextFeedback', () => {
  it('includes check names when provided', () => {
    const feedback = buildContextFeedback(['changelog', 'eslint']);
    expect(feedback).toContain('changelog');
    expect(feedback).toContain('eslint');
    expect(feedback).not.toContain('Y | N:');
  });

  it('omits table when empty', () => {
    const feedback = buildContextFeedback([]);
    expect(feedback).not.toMatch(/Enabled checks:/);
  });

  it('includes table with src links', () => {
    const feedback = buildContextFeedback(['changelog']);
    expect(feedback).toContain('changelog');
    expect(feedback).toContain('src/checks/changelog/README.md');
  });

  it('uses fallback width when stdout.columns missing or zero', () => {
    const orig = process.stdout.columns;
    for (const val of [undefined, 0]) {
      Object.defineProperty(process.stdout, 'columns', { value: val, configurable: true });
      try {
        const feedback = buildContextFeedback(['a', 'b']);
        expect(feedback).toContain('a');
        expect(feedback).toContain('b');
      } finally {
        Object.defineProperty(process.stdout, 'columns', { value: orig, configurable: true });
      }
    }
  });
});

describe('getColumns', () => {
  it('returns stdout.columns when valid number', () => {
    const orig = process.stdout.columns;
    Object.defineProperty(process.stdout, 'columns', { value: 120, configurable: true });
    try {
      expect(getColumns()).toBe(120);
    } finally {
      Object.defineProperty(process.stdout, 'columns', { value: orig, configurable: true });
    }
  });

  it('returns 80 when columns is undefined', () => {
    const orig = process.stdout.columns;
    Object.defineProperty(process.stdout, 'columns', { value: undefined, configurable: true });
    try {
      expect(getColumns()).toBe(80);
    } finally {
      Object.defineProperty(process.stdout, 'columns', { value: orig, configurable: true });
    }
  });

  it('returns 80 when columns is 0', () => {
    const orig = process.stdout.columns;
    Object.defineProperty(process.stdout, 'columns', { value: 0, configurable: true });
    try {
      expect(getColumns()).toBe(80);
    } finally {
      Object.defineProperty(process.stdout, 'columns', { value: orig, configurable: true });
    }
  });
});
