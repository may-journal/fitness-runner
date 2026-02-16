import { describe, it, expect, vi } from 'vitest';
import { readRepoFirstCheck, buildPrompt, getColumns } from './index.js';

vi.mock('node:readline/promises', () => ({
  createInterface: vi.fn(() => ({
    question: vi.fn().mockResolvedValue('Y'),
    close: vi.fn(),
  })),
}));

describe('readRepoFirstCheck', () => {
  it('passes when _readAnswer is Y', async () => {
    const result = await readRepoFirstCheck.run('/', { _readAnswer: 'Y' });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when _readAnswer is y', async () => {
    const result = await readRepoFirstCheck.run('/', { _readAnswer: 'y' });
    expect(result.ok).toBe(true);
  });

  it('fails when _readAnswer is N', async () => {
    const result = await readRepoFirstCheck.run('/', { _readAnswer: 'N' });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('familiarize yourself');
  });

  it('fails when _readAnswer is n', async () => {
    const result = await readRepoFirstCheck.run('/', { _readAnswer: 'n' });
    expect(result.ok).toBe(false);
  });

  it('includes enabledCheckNames in prompt when present', async () => {
    const ctx = { _readAnswer: 'y', enabledCheckNames: ['changelog', 'eslint'] };
    const result = await readRepoFirstCheck.run('/', ctx);
    expect(result.ok).toBe(true);
  });

  it('fails when _readAnswer is invalid and not Y', async () => {
    const result = await readRepoFirstCheck.run('/', { _readAnswer: 'maybe' });
    expect(result.ok).toBe(false);
  });

  it('passes when FITNESS_READ_REPO_CONFIRMED=1', async () => {
    const orig = process.env.FITNESS_READ_REPO_CONFIRMED;
    process.env.FITNESS_READ_REPO_CONFIRMED = '1';
    try {
      const result = await readRepoFirstCheck.run('/');
      expect(result.ok).toBe(true);
    } finally {
      process.env.FITNESS_READ_REPO_CONFIRMED = orig;
    }
  });

  it('fails when not TTY and no bypass', async () => {
    const origEnv = process.env.FITNESS_READ_REPO_CONFIRMED;
    const origIsTTY = process.stdin.isTTY;
    delete process.env.FITNESS_READ_REPO_CONFIRMED;
    Object.defineProperty(process.stdin, 'isTTY', { value: false, configurable: true });
    try {
      const result = await readRepoFirstCheck.run('/');
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toContain('interactively');
    } finally {
      process.env.FITNESS_READ_REPO_CONFIRMED = origEnv;
      Object.defineProperty(process.stdin, 'isTTY', { value: origIsTTY, configurable: true });
    }
  });

  it('prompts via _readlineQuestion and passes on Y', async () => {
    const origEnv = process.env.FITNESS_READ_REPO_CONFIRMED;
    const origIsTTY = process.stdin.isTTY;
    delete process.env.FITNESS_READ_REPO_CONFIRMED;
    Object.defineProperty(process.stdin, 'isTTY', { value: true, configurable: true });
    const questionFn = vi.fn().mockResolvedValue('Y');
    try {
      const result = await readRepoFirstCheck.run('/', {
        enabledCheckNames: ['changelog', 'eslint'],
        _readlineQuestion: questionFn,
      });
      expect(result.ok).toBe(true);
      expect(questionFn).toHaveBeenCalledWith(expect.stringContaining('changelog'));
      expect(questionFn).toHaveBeenCalledWith(expect.stringContaining('eslint'));
    } finally {
      process.env.FITNESS_READ_REPO_CONFIRMED = origEnv;
      Object.defineProperty(process.stdin, 'isTTY', { value: origIsTTY, configurable: true });
    }
  });

  it('uses real readline when TTY and no override (mocked module)', async () => {
    const origEnv = process.env.FITNESS_READ_REPO_CONFIRMED;
    const origIsTTY = process.stdin.isTTY;
    delete process.env.FITNESS_READ_REPO_CONFIRMED;
    Object.defineProperty(process.stdin, 'isTTY', { value: true, configurable: true });
    try {
      const result = await readRepoFirstCheck.run('/', { enabledCheckNames: ['eslint'] });
      expect(result.ok).toBe(true);
    } finally {
      process.env.FITNESS_READ_REPO_CONFIRMED = origEnv;
      Object.defineProperty(process.stdin, 'isTTY', { value: origIsTTY, configurable: true });
    }
  });

  it('prompts via _readlineQuestion and fails on N', async () => {
    const origEnv = process.env.FITNESS_READ_REPO_CONFIRMED;
    const origIsTTY = process.stdin.isTTY;
    delete process.env.FITNESS_READ_REPO_CONFIRMED;
    Object.defineProperty(process.stdin, 'isTTY', { value: true, configurable: true });
    try {
      const result = await readRepoFirstCheck.run('/', {
        _readlineQuestion: async () => 'N',
      });
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toContain('familiarize yourself');
    } finally {
      process.env.FITNESS_READ_REPO_CONFIRMED = origEnv;
      Object.defineProperty(process.stdin, 'isTTY', { value: origIsTTY, configurable: true });
    }
  });
});

describe('buildPrompt', () => {
  it('includes check names when provided', () => {
    const prompt = buildPrompt(['changelog', 'eslint']);
    expect(prompt).toContain('changelog');
    expect(prompt).toContain('eslint');
    expect(prompt).toContain('Y | N:');
  });

  it('omits list when empty', () => {
    const prompt = buildPrompt([]);
    expect(prompt).toContain('Y | N:');
    expect(prompt).not.toMatch(/\([\s\S]*\)\?/);
  });

  it('includes box and styling', () => {
    const prompt = buildPrompt(['changelog']);
    expect(prompt).toContain('─');
    expect(prompt).toContain('changelog');
  });

  it('uses fallback width when stdout.columns missing or zero', () => {
    const orig = process.stdout.columns;
    for (const val of [undefined, 0]) {
      Object.defineProperty(process.stdout, 'columns', { value: val, configurable: true });
      try {
        const prompt = buildPrompt(['a', 'b']);
        expect(prompt).toContain('a');
        expect(prompt).toContain('b');
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
