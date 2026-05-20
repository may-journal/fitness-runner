import { describe, it, expect } from 'vitest';
import { getColumns } from '@mayjournal/fitness-shared';

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
