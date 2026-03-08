import { describe, it, expect } from 'vitest';
import { execSync } from 'node:child_process';
import { EXEC_OPTS, execSyncResult } from './execSyncResult.js';

describe('execSyncResult', () => {
  it('returns exitCode 0 and output on success', () => {
    const { exitCode, output } = execSyncResult(process.cwd(), 'echo hello', (cmd, opts) =>
      execSync(cmd, { ...opts, encoding: 'utf8' })
    );
    expect(exitCode).toBe(0);
    expect(output.trim()).toBe('hello');
  });
  it('returns non-zero exitCode and stderr+stdout on failure', () => {
    const { exitCode, output } = execSyncResult(
      process.cwd(),
      'node -e "process.exit(2)"',
      (cmd, opts) => execSync(cmd, { ...opts, encoding: 'utf8' })
    );
    expect(exitCode).toBe(2);
    expect(typeof output).toBe('string');
  });
});

describe('EXEC_OPTS', () => {
  it('has encoding utf8 and maxBuffer 1MB', () => {
    expect(EXEC_OPTS.encoding).toBe('utf8');
    expect(EXEC_OPTS.maxBuffer).toBe(1024 * 1024);
  });
});
