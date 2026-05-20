import { describe, it, expect } from 'vitest';
import { defaultChecks } from '@mayjournal/fitness-checks/defaultChecks';
import { CheckName } from '../types/check-name.js';

/** Ensures CheckName enum and bundle defaultChecks stay in sync. */
describe('defaultChecks and CheckName sync', () => {
  const enumValues = Object.values(CheckName) as string[];
  const bundleNames = [...defaultChecks];

  it('every defaultChecks name exists in CheckName enum', () => {
    for (const name of bundleNames) {
      expect(enumValues).toContain(name);
    }
  });

  it('every CheckName enum value is in defaultChecks', () => {
    for (const name of enumValues) {
      expect(bundleNames).toContain(name);
    }
  });
});
