import { describe, it, expect } from 'vitest';
import { registry } from './index.js';
import { CheckName } from '../types/check-name.js';

/** Ensures CheckName enum and registry stay in sync (single source of truth: add to both when adding a check). */
describe('registry and CheckName sync', () => {
  const enumValues = Object.values(CheckName) as string[];
  const registryNames = registry.map((c) => c.name);

  it('every registry check name exists in CheckName enum', () => {
    for (const name of registryNames) {
      expect(enumValues).toContain(name);
    }
  });

  it('every CheckName enum value has a registry entry', () => {
    for (const name of enumValues) {
      expect(registryNames).toContain(name);
    }
  });
});
