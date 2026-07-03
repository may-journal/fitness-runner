import { mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { enUS, swiftlintCheck } from './index.js';

const PASS_OUTPUT = '[\n\n]\n';

const FAIL_OUTPUT = JSON.stringify([
  {
    character: 9,
    file: '/repo/Bad.swift',
    line: 4,
    reason: "Variable name 'x' should be between 3 and 40 characters long",
    rule_id: 'identifier_name',
    severity: 'Error',
    type: 'Identifier Name',
  },
]);

const COMMAND_NOT_FOUND_OUTPUT = '/bin/sh: swiftlint: command not found\n';
const NO_LINTABLE_FILES_OUTPUT = "Error: No lintable files found at paths: '.'\n";

function tempDir(): string {
  return mkdtempSync(join(tmpdir(), 'swiftlint-'));
}

describe('swiftlint top-level exports', () => {
  it('default export matches swiftlintCheck', async () => {
    const mod = await import('./index.js');
    expect(mod.default).toBe(mod.swiftlintCheck);
  });

  it('enUS exposes NotInstalled and FallbackRunHint', () => {
    expect(enUS.NotInstalled).toBe('SwiftLint not installed: brew install swiftlint');
    expect(enUS.FallbackRunHint).toBe('swiftlint reported an error (run: swiftlint lint --strict)');
  });

  describe('swiftlintCheck', () => {
    it('has name swiftlint', () => {
      expect(swiftlintCheck.name).toBe('swiftlint');
    });

    it('run() passes on an empty violation array', async () => {
      const dir = tempDir();
      const result = await swiftlintCheck.run(dir, { _execSync: () => PASS_OUTPUT });
      expect(result.ok).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('run() fails and formats violations from the JSON reporter', async () => {
      const dir = tempDir();
      const result = await swiftlintCheck.run(dir, {
        _execSync: () => {
          throw { status: 2, stdout: FAIL_OUTPUT };
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors).toEqual([
        "/repo/Bad.swift:4:9 - Variable name 'x' should be between 3 and 40 characters long (identifier_name)",
      ]);
    });

    it('run() reports NotInstalled when the binary is missing', async () => {
      const dir = tempDir();
      const result = await swiftlintCheck.run(dir, {
        _execSync: () => {
          throw { status: 127, stdout: COMMAND_NOT_FOUND_OUTPUT };
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors).toEqual([enUS.NotInstalled]);
    });

    it('run() skips clean when there are no Swift files', async () => {
      const dir = tempDir();
      const result = await swiftlintCheck.run(dir, {
        _execSync: () => {
          throw { status: 1, stdout: NO_LINTABLE_FILES_OUTPUT };
        },
      });
      expect(result.ok).toBe(true);
      expect(result.errors).toHaveLength(0);
      expect(result.meta?.filesChecked).toBe(0);
    });

    it('run() falls back to FallbackRunHint on malformed output with no parsed violations', async () => {
      const dir = tempDir();
      const result = await swiftlintCheck.run(dir, {
        _execSync: () => {
          throw { status: 1, stdout: 'swiftlint crashed unexpectedly' };
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors).toEqual([enUS.FallbackRunHint]);
    });
  });
});
