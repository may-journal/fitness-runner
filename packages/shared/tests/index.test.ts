import { describe, expect, it } from 'vitest';
import * as shared from '@mayjournal/fitness-shared';

describe('shared package exports', () => {
  it('re-exports check helpers and config loaders from the package entry', () => {
    expect(shared.checkResult(true).ok).toBe(true);
    expect(typeof shared.loadConfig).toBe('function');
    expect(typeof shared.getFitnessRunnerRoot).toBe('function');
    expect(typeof shared.loadVitestConfig).toBe('function');
  });
});
