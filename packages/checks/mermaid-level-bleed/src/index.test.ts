import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import mermaidLevelBleedCheck, { levelDescriptions } from './index.js';

const diagram = ['```mermaid', 'Rel(a, b, "1")', '```'].join('\n');

/** A doc with one callout table of [#, Description] rows. */
function levelDoc(...descriptions: string[]): string {
  return [
    diagram,
    '',
    '| # | Description |',
    '| --- | --- |',
    ...descriptions.map((d, i) => `| ${i + 1} | ${d} |`),
  ].join('\n');
}

describe('levelDescriptions', () => {
  it('normalizes whitespace and case', () => {
    const set = levelDescriptions(levelDoc('The  App', 'THE api'));
    expect(set.has('the app')).toBe(true);
    expect(set.has('the api')).toBe(true);
  });
});

describe('mermaidLevelBleedCheck', () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'mermaid-bleed-'));
  });
  function write(rel: string, content: string): void {
    const full = join(dir, rel);
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, content);
  }

  it('passes when adjacent levels have distinct descriptions', async () => {
    write('architecture/01-context.md', levelDoc('The whole system'));
    write('architecture/02-containers.md', levelDoc('The macOS app', 'The sync engine'));
    const result = await mermaidLevelBleedCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(2);
  });

  it('flags a description repeated verbatim from the previous level', async () => {
    write('architecture/01-context.md', levelDoc('User interacts with the timeline'));
    write(
      'architecture/02-containers.md',
      levelDoc('User interacts with the timeline', 'New detail')
    );
    const result = await mermaidLevelBleedCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(
      result.errors.some(
        (e) => e.includes('02-containers.md') && e.includes('repeats level 1 verbatim')
      )
    ).toBe(true);
  });

  it('ignores markdown outside the architecture level convention', async () => {
    write('README.md', levelDoc('Anything'));
    const result = await mermaidLevelBleedCheck.run(dir);
    expect(result).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
  });
});
